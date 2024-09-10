package payment_intent

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"goflare.io/ember"
	"goflare.io/ember/config"
	"goflare.io/ignite"
	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
)

type Repository interface {
	Create(ctx context.Context, tx pgx.Tx, paymentIntent *models.PaymentIntent) error
	GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.PaymentIntent, error)
	Update(ctx context.Context, tx pgx.Tx, paymentIntent *models.PaymentIntent) error
	List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.PaymentIntent, error)
	ListByCustomer(ctx context.Context, tx pgx.Tx, customerID string, limit, offset uint64) ([]*models.PaymentIntent, error)
	Upsert(ctx context.Context, tx pgx.Tx, paymentIntent *models.PartialPaymentIntent) error
}

type repository struct {
	conn        driver.PostgresPool
	logger      *zap.Logger
	cache       *ember.MultiCache
	poolManager ignite.Manager
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger, cache *ember.MultiCache, poolManager ignite.Manager) (Repository, error) {
	err := poolManager.RegisterPool(reflect.TypeOf(&models.PaymentIntent{}), ignite.Config[any]{
		InitialSize: 10,
		MaxSize:     100,
		MaxIdleTime: 10 * time.Minute,
		Factory: func() (any, error) {
			return models.NewPaymentIntent(), nil
		},
		Reset: func(obj any) error {
			pi := obj.(*models.PaymentIntent)
			*pi = models.PaymentIntent{}
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register payment intent pool: %w", err)
	}

	return &repository{
		conn:        conn,
		logger:      logger,
		cache:       cache,
		poolManager: poolManager,
	}, nil
}

func (r *repository) getFromPool(ctx context.Context) (*models.PaymentIntent, func(), error) {
	pool, err := r.poolManager.GetPool(reflect.TypeOf(&models.PaymentIntent{}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool: %w", err)
	}

	objWrapper, err := pool.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object from pool: %w", err)
	}

	pi := objWrapper.Object.(*models.PaymentIntent)
	release := func() {
		pool.Put(objWrapper)
	}

	return pi, release, nil
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, paymentIntent *models.PaymentIntent) error {

	if err := sqlc.New(r.conn).WithTx(tx).CreatePaymentIntent(ctx, sqlc.CreatePaymentIntentParams{
		ID:               paymentIntent.ID,
		CustomerID:       paymentIntent.CustomerID,
		Amount:           paymentIntent.Amount,
		Currency:         sqlc.Currency(paymentIntent.Currency),
		Status:           sqlc.PaymentIntentStatus(paymentIntent.Status),
		PaymentMethodID:  &paymentIntent.PaymentMethodID,
		SetupFutureUsage: sqlc.NullPaymentIntentSetupFutureUsage{PaymentIntentSetupFutureUsage: sqlc.PaymentIntentSetupFutureUsage(paymentIntent.SetupFutureUsage), Valid: paymentIntent.SetupFutureUsage != ""},
		ClientSecret:     paymentIntent.ClientSecret,
	}); err != nil {
		r.logger.Error(fmt.Sprintf("failed to create payment intent: %s", err.Error()))
		return fmt.Errorf("failed to create payment intent: %w", err)
	}

	cacheKey := fmt.Sprintf("payment_intent:%s", paymentIntent.ID)
	if err := r.cache.Set(ctx, cacheKey, paymentIntent, config.NewConfig().DefaultExpiration); err != nil {
		r.logger.Warn("Failed to cache payment intent", zap.Error(err), zap.String("id", paymentIntent.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.PaymentIntent, error) {
	cacheKey := fmt.Sprintf("payment_intent:%s", id)

	paymentIntent, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	found, err := r.cache.Get(ctx, cacheKey, paymentIntent)
	if err != nil {
		r.logger.Warn("Failed to get payment intent from cache", zap.Error(err), zap.String("id", id))
	} else if found {
		return paymentIntent, nil
	}

	sqlcPaymentIntent, err := sqlc.New(r.conn).WithTx(tx).GetPaymentIntent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}

	*paymentIntent = *models.NewPaymentIntent().ConvertFromSQLCPaymentIntent(sqlcPaymentIntent)

	if err = r.cache.Set(ctx, cacheKey, paymentIntent, config.NewConfig().DefaultExpiration); err != nil {
		r.logger.Warn("Failed to cache payment intent", zap.Error(err), zap.String("id", id))
	}

	return paymentIntent, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, paymentIntent *models.PaymentIntent) error {

	err := sqlc.New(r.conn).WithTx(tx).UpdatePaymentIntent(ctx, sqlc.UpdatePaymentIntentParams{
		ID:               paymentIntent.ID,
		Status:           sqlc.PaymentIntentStatus(paymentIntent.Status),
		PaymentMethodID:  &paymentIntent.PaymentMethodID,
		SetupFutureUsage: sqlc.NullPaymentIntentSetupFutureUsage{PaymentIntentSetupFutureUsage: sqlc.PaymentIntentSetupFutureUsage(paymentIntent.SetupFutureUsage), Valid: paymentIntent.SetupFutureUsage != ""},
		ClientSecret:     paymentIntent.ClientSecret,
	})
	if err != nil {
		return fmt.Errorf("failed to update payment intent: %w", err)
	}

	cacheKey := fmt.Sprintf("payment_intent:%s", paymentIntent.ID)
	if err = r.cache.Set(ctx, cacheKey, paymentIntent, config.NewConfig().DefaultExpiration); err != nil {
		r.logger.Warn("Failed to update payment intent in cache", zap.Error(err), zap.String("id", paymentIntent.ID))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.PaymentIntent, error) {
	cacheKey := fmt.Sprintf("payment_intents:limit:%d:offset:%d", limit, offset)
	var cachedPaymentIntents []*models.PaymentIntent
	found, err := r.cache.Get(ctx, cacheKey, &cachedPaymentIntents)
	if err != nil {
		r.logger.Warn("Failed to get payment intents from cache", zap.Error(err))
	} else if found {
		return cachedPaymentIntents, nil
	}

	sqlcPaymentIntents, err := sqlc.New(r.conn).WithTx(tx).ListPaymentIntents(ctx, sqlc.ListPaymentIntentsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list payment intents: %w", err)
	}

	paymentIntents := make([]*models.PaymentIntent, 0, len(sqlcPaymentIntents))
	for _, sqlcPaymentIntent := range sqlcPaymentIntents {
		pi, release, err := r.getFromPool(ctx)
		if err != nil {
			r.logger.Error("Failed to get payment intent from pool", zap.Error(err))
			continue
		}

		*pi = *models.NewPaymentIntent().ConvertFromSQLCPaymentIntent(sqlcPaymentIntent)
		paymentIntents = append(paymentIntents, pi)

		// 更新單個 PaymentIntent 的緩存
		singleCacheKey := fmt.Sprintf("payment_intent:%s", pi.ID)
		if err = r.cache.Set(ctx, singleCacheKey, pi, config.NewConfig().DefaultExpiration); err != nil {
			r.logger.Warn("Failed to cache single payment intent", zap.Error(err), zap.String("id", pi.ID))
		}

		release()
	}

	if err = r.cache.Set(ctx, cacheKey, paymentIntents); err != nil {
		r.logger.Warn("Failed to cache payment intents list", zap.Error(err))
	}

	return paymentIntents, nil
}

func (r *repository) ListByCustomer(ctx context.Context, tx pgx.Tx, customerID string, limit, offset uint64) ([]*models.PaymentIntent, error) {
	cacheKey := fmt.Sprintf("payment_intents:customer:%s:limit:%d:offset:%d", customerID, limit, offset)
	var cachedPaymentIntents []*models.PaymentIntent
	found, err := r.cache.Get(ctx, cacheKey, &cachedPaymentIntents)
	if err != nil {
		r.logger.Warn("Failed to get payment intents from cache", zap.Error(err), zap.String("customerID", customerID))
	} else if found {
		return cachedPaymentIntents, nil
	}

	sqlcPaymentIntents, err := sqlc.New(r.conn).WithTx(tx).ListPaymentIntentsByCustomer(ctx, sqlc.ListPaymentIntentsByCustomerParams{
		CustomerID: customerID,
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list payment intents: %w", err)
	}

	paymentIntents := make([]*models.PaymentIntent, 0, len(sqlcPaymentIntents))
	for _, sqlcPaymentIntent := range sqlcPaymentIntents {
		pi, release, err := r.getFromPool(ctx)
		if err != nil {
			r.logger.Error("Failed to get payment intent from pool", zap.Error(err))
			continue
		}

		*pi = *models.NewPaymentIntent().ConvertFromSQLCPaymentIntent(sqlcPaymentIntent)
		paymentIntents = append(paymentIntents, pi)

		// 更新單個 PaymentIntent 的緩存
		singleCacheKey := fmt.Sprintf("payment_intent:%s", pi.ID)
		if err = r.cache.Set(ctx, singleCacheKey, pi, config.NewConfig().DefaultExpiration); err != nil {
			r.logger.Warn("Failed to cache single payment intent", zap.Error(err), zap.String("id", pi.ID))
		}

		release()
	}

	if err = r.cache.Set(ctx, cacheKey, paymentIntents, 5*time.Minute); err != nil {
		r.logger.Warn("Failed to cache payment intents list", zap.Error(err), zap.String("customerID", customerID))
	}

	return paymentIntents, nil
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, paymentIntent *models.PartialPaymentIntent) error {
	query := `
    INSERT INTO payment_intents (id, customer_id, amount, currency, status, payment_method_id, setup_future_usage, client_secret, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{paymentIntent.ID}
	updateClauses := []string{}
	argIndex := 2

	if paymentIntent.CustomerID != nil {
		args = append(args, *paymentIntent.CustomerID)
		updateClauses = append(updateClauses, fmt.Sprintf("customer_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentIntent.Amount != nil {
		args = append(args, *paymentIntent.Amount)
		updateClauses = append(updateClauses, fmt.Sprintf("amount = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentIntent.Currency != nil {
		args = append(args, *paymentIntent.Currency)
		updateClauses = append(updateClauses, fmt.Sprintf("currency = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentIntent.Status != nil {
		args = append(args, *paymentIntent.Status)
		updateClauses = append(updateClauses, fmt.Sprintf("status = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentIntent.PaymentMethodID != nil {
		args = append(args, *paymentIntent.PaymentMethodID)
		updateClauses = append(updateClauses, fmt.Sprintf("payment_method_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentIntent.SetupFutureUsage != nil {
		args = append(args, *paymentIntent.SetupFutureUsage)
		updateClauses = append(updateClauses, fmt.Sprintf("setup_future_usage = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentIntent.ClientSecret != nil {
		args = append(args, *paymentIntent.ClientSecret)
		updateClauses = append(updateClauses, fmt.Sprintf("client_secret = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentIntent.CreatedAt != nil {
		args = append(args, *paymentIntent.CreatedAt)
		updateClauses = append(updateClauses, fmt.Sprintf("created_at = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	args = append(args, time.Now())
	updateClauses = append(updateClauses, fmt.Sprintf("updated_at = $%d", argIndex))

	if len(updateClauses) > 0 {
		query += strings.Join(updateClauses, ", ")
	}
	query += " WHERE id = $1"

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to upsert payment intent: %w", err)
	}

	return nil
}
