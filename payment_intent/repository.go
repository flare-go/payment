package payment_intent

import (
	"context"
	"fmt"
	"reflect"
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
	GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.PaymentIntent, error)
	Update(ctx context.Context, tx pgx.Tx, paymentIntent *models.PaymentIntent) error
	List(ctx context.Context, tx pgx.Tx, customerID uint64, limit, offset uint64) ([]*models.PaymentIntent, error)
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
	var paymentMethodID uint64
	var setupFutureUsage string

	if paymentIntent.PaymentMethodID != nil {
		paymentMethodID = *paymentIntent.PaymentMethodID
	}
	if paymentIntent.SetupFutureUsage != nil {
		setupFutureUsage = *paymentIntent.SetupFutureUsage
	}

	err := sqlc.New(r.conn).WithTx(tx).CreatePaymentIntent(ctx, sqlc.CreatePaymentIntentParams{
		CustomerID:       paymentIntent.CustomerID,
		Amount:           paymentIntent.Amount,
		Currency:         sqlc.Currency(paymentIntent.Currency),
		Status:           sqlc.PaymentIntentStatus(paymentIntent.Status),
		PaymentMethodID:  paymentMethodID,
		SetupFutureUsage: setupFutureUsage,
		StripeID:         paymentIntent.StripeID,
		ClientSecret:     paymentIntent.ClientSecret,
	})
	if err != nil {
		return fmt.Errorf("failed to create payment intent: %w", err)
	}

	cacheKey := fmt.Sprintf("payment_intent:%d", paymentIntent.ID)
	if err := r.cache.Set(ctx, cacheKey, paymentIntent, config.NewConfig().DefaultExpiration); err != nil {
		r.logger.Warn("Failed to cache payment intent", zap.Error(err), zap.Uint64("id", paymentIntent.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.PaymentIntent, error) {
	cacheKey := fmt.Sprintf("payment_intent:%d", id)

	paymentIntent, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	found, err := r.cache.Get(ctx, cacheKey, paymentIntent)
	if err != nil {
		r.logger.Warn("Failed to get payment intent from cache", zap.Error(err), zap.Uint64("id", id))
	} else if found {
		return paymentIntent, nil
	}

	sqlcPaymentIntent, err := sqlc.New(r.conn).WithTx(tx).GetPaymentIntent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}

	*paymentIntent = *models.NewPaymentIntent().ConvertFromSQLCPaymentIntent(sqlcPaymentIntent)

	if err := r.cache.Set(ctx, cacheKey, paymentIntent, config.NewConfig().DefaultExpiration); err != nil {
		r.logger.Warn("Failed to cache payment intent", zap.Error(err), zap.Uint64("id", id))
	}

	return paymentIntent, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, paymentIntent *models.PaymentIntent) error {
	var paymentMethodID uint64
	var setupFutureUsage string

	if paymentIntent.PaymentMethodID != nil {
		paymentMethodID = *paymentIntent.PaymentMethodID
	}
	if paymentIntent.SetupFutureUsage != nil {
		setupFutureUsage = *paymentIntent.SetupFutureUsage
	}

	err := sqlc.New(r.conn).WithTx(tx).UpdatePaymentIntent(ctx, sqlc.UpdatePaymentIntentParams{
		ID:               paymentIntent.ID,
		Status:           sqlc.PaymentIntentStatus(paymentIntent.Status),
		PaymentMethodID:  paymentMethodID,
		SetupFutureUsage: setupFutureUsage,
		StripeID:         paymentIntent.StripeID,
		ClientSecret:     paymentIntent.ClientSecret,
	})
	if err != nil {
		return fmt.Errorf("failed to update payment intent: %w", err)
	}

	cacheKey := fmt.Sprintf("payment_intent:%d", paymentIntent.ID)
	if err := r.cache.Set(ctx, cacheKey, paymentIntent, config.NewConfig().DefaultExpiration); err != nil {
		r.logger.Warn("Failed to update payment intent in cache", zap.Error(err), zap.Uint64("id", paymentIntent.ID))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, customerID uint64, limit, offset uint64) ([]*models.PaymentIntent, error) {
	cacheKey := fmt.Sprintf("payment_intents:customer:%d:limit:%d:offset:%d", customerID, limit, offset)
	var cachedPaymentIntents []*models.PaymentIntent
	found, err := r.cache.Get(ctx, cacheKey, &cachedPaymentIntents)
	if err != nil {
		r.logger.Warn("Failed to get payment intents from cache", zap.Error(err), zap.Uint64("customerID", customerID))
	} else if found {
		return cachedPaymentIntents, nil
	}

	sqlcPaymentIntents, err := sqlc.New(r.conn).WithTx(tx).ListPaymentIntents(ctx, sqlc.ListPaymentIntentsParams{
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
		singleCacheKey := fmt.Sprintf("payment_intent:%d", pi.ID)
		if err := r.cache.Set(ctx, singleCacheKey, pi, config.NewConfig().DefaultExpiration); err != nil {
			r.logger.Warn("Failed to cache single payment intent", zap.Error(err), zap.Uint64("id", pi.ID))
		}

		release()
	}

	if err := r.cache.Set(ctx, cacheKey, paymentIntents, 5*time.Minute); err != nil {
		r.logger.Warn("Failed to cache payment intents list", zap.Error(err), zap.Uint64("customerID", customerID))
	}

	return paymentIntents, nil
}
