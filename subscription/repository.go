package subscription

import (
	"context"
	"fmt"
	"github.com/stripe/stripe-go/v79"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"goflare.io/ember"
	"goflare.io/ignite"
	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
)

type Repository interface {
	Create(ctx context.Context, tx pgx.Tx, subscription *models.Subscription) error
	GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Subscription, error)
	Update(ctx context.Context, tx pgx.Tx, subscription *models.Subscription) error
	Cancel(ctx context.Context, tx pgx.Tx, id string, cancelAtPeriodEnd bool) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
	List(ctx context.Context, tx pgx.Tx, customerID string, limit, offset uint64) ([]*models.Subscription, error)
	GetExpiringSubscriptions(ctx context.Context, tx pgx.Tx, expirationDate time.Time) ([]*models.Subscription, error)
	Upsert(ctx context.Context, tx pgx.Tx, subscription *models.PartialSubscription) error
}

type repository struct {
	conn        driver.PostgresPool
	logger      *zap.Logger
	cache       *ember.MultiCache
	poolManager ignite.Manager
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger, cache *ember.MultiCache, poolManager ignite.Manager) (Repository, error) {
	if err := poolManager.RegisterPool(reflect.TypeOf(&models.Subscription{}), ignite.Config[any]{
		InitialSize: 10,
		MaxSize:     100,
		MaxIdleTime: 10 * time.Minute,
		Factory: func() (any, error) {
			return models.NewSubscription(), nil
		},
		Reset: func(obj any) error {
			s := obj.(*models.Subscription)
			*s = models.Subscription{}
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to register subscription pool: %w", err)
	}

	return &repository{
		conn:        conn,
		logger:      logger,
		cache:       cache,
		poolManager: poolManager,
	}, nil
}

func (r *repository) getFromPool(ctx context.Context) (*models.Subscription, func(), error) {
	pool, err := r.poolManager.GetPool(reflect.TypeOf(&models.Subscription{}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool: %w", err)
	}

	objWrapper, err := pool.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object from pool: %w", err)
	}

	subscription := objWrapper.Object.(*models.Subscription)
	release := func() {
		pool.Put(objWrapper)
	}

	return subscription, release, nil
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, subscription *models.Subscription) error {

	var trialStart, trialEnd time.Time

	if subscription.TrialStart != nil {
		trialStart = *subscription.TrialStart
	}

	if subscription.TrialEnd != nil {
		trialEnd = *subscription.TrialEnd
	}

	if err := sqlc.New(r.conn).WithTx(tx).CreateSubscription(ctx, sqlc.CreateSubscriptionParams{
		ID:                 subscription.ID,
		CustomerID:         subscription.CustomerID,
		PriceID:            subscription.PriceID,
		Status:             sqlc.SubscriptionStatus(subscription.Status),
		CurrentPeriodStart: pgtype.Timestamptz{Time: subscription.CurrentPeriodStart},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: subscription.CurrentPeriodEnd},
		CancelAtPeriodEnd:  subscription.CancelAtPeriodEnd,
		TrialStart:         pgtype.Timestamptz{Time: trialStart},
		TrialEnd:           pgtype.Timestamptz{Time: trialEnd},
	}); err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	cacheKey := fmt.Sprintf("subscription:%s", subscription.ID)
	if err := r.cache.Set(ctx, cacheKey, subscription); err != nil {
		r.logger.Warn("Failed to cache new subscription", zap.Error(err), zap.String("id", subscription.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Subscription, error) {
	cacheKey := fmt.Sprintf("subscription:%s", id)

	subscription, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	found, err := r.cache.Get(ctx, cacheKey, subscription)
	if err != nil {
		r.logger.Warn("Failed to get subscription from cache", zap.Error(err), zap.String("id", id))
	} else if found {
		return subscription, nil
	}

	sqlcSubscription, err := sqlc.New(r.conn).WithTx(tx).GetSubscription(ctx, id)
	if err != nil {
		r.logger.Error("error getting subscription", zap.Error(err))
		return nil, err
	}

	*subscription = *models.NewSubscription().ConvertFromSQLCSubscription(sqlcSubscription)

	if err = r.cache.Set(ctx, cacheKey, subscription, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache subscription", zap.Error(err), zap.String("id", id))
	}

	return subscription, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, subscription *models.Subscription) error {
	var cancelAt, trialStart, trialEnd time.Time
	if subscription.CanceledAt != nil {
		cancelAt = *subscription.CanceledAt
	}
	if subscription.TrialStart != nil {
		trialStart = *subscription.TrialStart
	}
	if subscription.TrialEnd != nil {
		trialStart = *subscription.TrialEnd
	}

	if err := sqlc.New(r.conn).WithTx(tx).UpdateSubscription(ctx, sqlc.UpdateSubscriptionParams{
		ID:                 subscription.ID,
		PriceID:            subscription.PriceID,
		Status:             sqlc.SubscriptionStatus(subscription.Status),
		CurrentPeriodStart: pgtype.Timestamptz{Time: subscription.CurrentPeriodStart},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: subscription.CurrentPeriodEnd},
		CanceledAt:         pgtype.Timestamptz{Time: cancelAt},
		CancelAtPeriodEnd:  subscription.CancelAtPeriodEnd,
		TrialStart:         pgtype.Timestamptz{Time: trialStart},
		TrialEnd:           pgtype.Timestamptz{Time: trialEnd},
	}); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	cacheKey := fmt.Sprintf("subscription:%s", subscription.ID)
	if err := r.cache.Set(ctx, cacheKey, subscription); err != nil {
		r.logger.Warn("Failed to update subscription in cache", zap.Error(err), zap.String("id", subscription.ID))
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteSubscription(ctx, id)
}

func (r *repository) Cancel(ctx context.Context, tx pgx.Tx, id string, cancelAtPeriodEnd bool) error {
	subscription, err := r.GetByID(ctx, tx, id)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	now := time.Now()
	subscription.Status = stripe.SubscriptionStatusCanceled
	subscription.CanceledAt = &now
	subscription.CancelAtPeriodEnd = cancelAtPeriodEnd

	return r.Update(ctx, tx, subscription)
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, customerID string, limit, offset uint64) ([]*models.Subscription, error) {
	cacheKey := fmt.Sprintf("subscriptions:customer:%s:limit:%d:offset:%d", customerID, limit, offset)
	var cachedSubscriptions []*models.Subscription
	found, err := r.cache.Get(ctx, cacheKey, &cachedSubscriptions)
	if err != nil {
		r.logger.Warn("Failed to get subscriptions from cache", zap.Error(err), zap.String("customerID", customerID))
	} else if found {
		return cachedSubscriptions, nil
	}

	sqlcSubscriptions, err := sqlc.New(r.conn).WithTx(tx).ListSubscriptions(ctx, sqlc.ListSubscriptionsParams{
		CustomerID: customerID,
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
	if err != nil {
		r.logger.Error("error listing subscriptions", zap.Error(err))
		return nil, err
	}

	subscriptions := make([]*models.Subscription, 0, len(sqlcSubscriptions))
	for _, sqlcSubscription := range sqlcSubscriptions {
		subscription, release, err := r.getFromPool(ctx)
		if err != nil {
			r.logger.Error("Failed to get subscription from pool", zap.Error(err))
			continue
		}

		*subscription = *models.NewSubscription().ConvertFromSQLCSubscription(sqlcSubscription)
		subscriptions = append(subscriptions, subscription)

		singleCacheKey := fmt.Sprintf("subscription:%s", subscription.ID)
		if err = r.cache.Set(ctx, singleCacheKey, subscription, 30*time.Minute); err != nil {
			r.logger.Warn("Failed to cache single subscription", zap.Error(err), zap.String("id", subscription.ID))
		}

		release()
	}

	if err = r.cache.Set(ctx, cacheKey, subscriptions); err != nil {
		r.logger.Warn("Failed to cache subscriptions list", zap.Error(err), zap.String("customerID", customerID))
	}

	return subscriptions, nil
}

func (r *repository) GetExpiringSubscriptions(ctx context.Context, tx pgx.Tx, expirationDate time.Time) ([]*models.Subscription, error) {
	// 使用 sqlc 生成的查詢方法
	sqlcSubscriptions, err := sqlc.New(r.conn).WithTx(tx).GetExpiringSubscriptions(ctx, sqlc.GetExpiringSubscriptionsParams{
		CurrentPeriodEnd: pgtype.Timestamptz{Time: expirationDate},
		Status:           sqlc.SubscriptionStatus(stripe.SubscriptionStatusActive),
	})
	if err != nil {
		r.logger.Error("error getting expiring subscriptions", zap.Error(err))
		return nil, fmt.Errorf("failed to get expiring subscriptions: %w", err)
	}

	subscriptions := make([]*models.Subscription, 0, len(sqlcSubscriptions))
	for _, sqlcSubscription := range sqlcSubscriptions {
		subscription, release, err := r.getFromPool(ctx)
		if err != nil {
			r.logger.Error("Failed to get subscription from pool", zap.Error(err))
			continue
		}

		*subscription = *models.NewSubscription().ConvertFromSQLCSubscription(sqlcSubscription)
		subscriptions = append(subscriptions, subscription)

		// 可以考慮在這裡更新單個訂閱的緩存，但要注意平衡效能
		// singleCacheKey := fmt.Sprintf("subscription:%d", subscription.ID)
		// if err := r.cache.Set(ctx, singleCacheKey, subscription, 30*time.Minute); err != nil {
		//     r.logger.Warn("Failed to cache single subscription", zap.Error(err), zap.Uint64("id", subscription.ID))
		// }
		release()
	}

	return subscriptions, nil
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, subscription *models.PartialSubscription) error {
	query := `
    INSERT INTO subscriptions (id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{subscription.ID}
	updateClauses := []string{}
	argIndex := 2

	if subscription.CustomerID != nil {
		args = append(args, *subscription.CustomerID)
		updateClauses = append(updateClauses, fmt.Sprintf("customer_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if subscription.PriceID != nil {
		args = append(args, *subscription.PriceID)
		updateClauses = append(updateClauses, fmt.Sprintf("price_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if subscription.Status != nil {
		args = append(args, *subscription.Status)
		updateClauses = append(updateClauses, fmt.Sprintf("status = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if subscription.CurrentPeriodStart != nil {
		args = append(args, *subscription.CurrentPeriodStart)
		updateClauses = append(updateClauses, fmt.Sprintf("current_period_start = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if subscription.CurrentPeriodEnd != nil {
		args = append(args, *subscription.CurrentPeriodEnd)
		updateClauses = append(updateClauses, fmt.Sprintf("current_period_end = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if subscription.CanceledAt != nil {
		args = append(args, *subscription.CanceledAt)
		updateClauses = append(updateClauses, fmt.Sprintf("canceled_at = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if subscription.CancelAtPeriodEnd != nil {
		args = append(args, *subscription.CancelAtPeriodEnd)
		updateClauses = append(updateClauses, fmt.Sprintf("cancel_at_period_end = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if subscription.TrialStart != nil {
		args = append(args, *subscription.TrialStart)
		updateClauses = append(updateClauses, fmt.Sprintf("trial_start = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if subscription.TrialEnd != nil {
		args = append(args, *subscription.TrialEnd)
		updateClauses = append(updateClauses, fmt.Sprintf("trial_end = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if subscription.CreatedAt != nil {
		args = append(args, *subscription.CreatedAt)
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
		return fmt.Errorf("failed to upsert subscription: %w", err)
	}

	return nil
}
