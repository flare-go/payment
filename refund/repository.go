package refund

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"goflare.io/ember"
	"goflare.io/ignite"
	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
)

type Repository interface {
	Create(ctx context.Context, tx pgx.Tx, refund *models.Refund) error
	GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Refund, error)
	Update(ctx context.Context, tx pgx.Tx, refund *models.Refund) error
	List(ctx context.Context, tx pgx.Tx, chargeID string, limit, offset uint64) ([]*models.Refund, error)
	ListByChargeID(ctx context.Context, chargeID string) ([]*models.Refund, error)
	Upsert(ctx context.Context, tx pgx.Tx, refund *models.PartialRefund) error
}

type repository struct {
	conn        driver.PostgresPool
	logger      *zap.Logger
	cache       *ember.MultiCache
	poolManager ignite.Manager
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger, cache *ember.MultiCache, poolManager ignite.Manager) (Repository, error) {
	if err := poolManager.RegisterPool(reflect.TypeOf(&models.Refund{}), ignite.Config[any]{
		InitialSize: 10,
		MaxSize:     100,
		MaxIdleTime: 10 * time.Minute,
		Factory: func() (any, error) {
			return models.NewRefund(), nil
		},
		Reset: func(obj any) error {
			r := obj.(*models.Refund)
			*r = models.Refund{}
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to register refund pool: %w", err)
	}

	return &repository{
		conn:        conn,
		logger:      logger,
		cache:       cache,
		poolManager: poolManager,
	}, nil
}

func (r *repository) getFromPool(ctx context.Context) (*models.Refund, func(), error) {
	pool, err := r.poolManager.GetPool(reflect.TypeOf(&models.Refund{}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool: %w", err)
	}

	objWrapper, err := pool.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object from pool: %w", err)
	}

	refund := objWrapper.Object.(*models.Refund)
	release := func() {
		pool.Put(objWrapper)
	}

	return refund, release, nil
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, refund *models.Refund) error {
	if err := sqlc.New(r.conn).WithTx(tx).CreateRefund(ctx, sqlc.CreateRefundParams{
		ID:       refund.ID,
		ChargeID: refund.ChargeID,
		Amount:   refund.Amount,
		Status:   sqlc.RefundStatus(refund.Status),
		Reason:   &refund.Reason,
	}); err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	cacheKey := fmt.Sprintf("refund:%s", refund.ID)
	if err := r.cache.Set(ctx, cacheKey, refund); err != nil {
		r.logger.Warn("Failed to cache refund", zap.Error(err), zap.String("id", refund.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Refund, error) {
	cacheKey := fmt.Sprintf("refund:%s", id)

	refund, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	found, err := r.cache.Get(ctx, cacheKey, refund)
	if err != nil {
		r.logger.Warn("Failed to get refund from cache", zap.Error(err), zap.String("id", id))
	} else if found {
		return refund, nil
	}

	sqlcRefund, err := sqlc.New(r.conn).WithTx(tx).GetRefund(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get refund: %w", err)
	}

	*refund = *models.NewRefund().ConvertFromSQLCRefund(sqlcRefund)

	if err = r.cache.Set(ctx, cacheKey, refund); err != nil {
		r.logger.Warn("Failed to cache refund", zap.Error(err), zap.String("id", id))
	}

	return refund, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, refund *models.Refund) error {
	if err := sqlc.New(r.conn).WithTx(tx).UpdateRefund(ctx, sqlc.UpdateRefundParams{
		ID:     refund.ID,
		Status: sqlc.RefundStatus(refund.Status),
		Reason: &refund.Reason,
	}); err != nil {
		return fmt.Errorf("failed to update refund: %w", err)
	}

	cacheKey := fmt.Sprintf("refund:%s", refund.ID)
	if err := r.cache.Set(ctx, cacheKey, refund); err != nil {
		r.logger.Warn("Failed to update refund in cache", zap.Error(err), zap.String("id", refund.ID))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, chargeID string, limit, offset uint64) ([]*models.Refund, error) {
	cacheKey := fmt.Sprintf("refunds:chargeID:%s:limit:%d:offset:%d", chargeID, limit, offset)
	var cachedRefunds []*models.Refund
	found, err := r.cache.Get(ctx, cacheKey, &cachedRefunds)
	if err != nil {
		r.logger.Warn("Failed to get refunds from cache", zap.Error(err), zap.String("chargeID", chargeID))
	} else if found {
		return cachedRefunds, nil
	}

	sqlcRefunds, err := sqlc.New(r.conn).WithTx(tx).ListRefunds(ctx, sqlc.ListRefundsParams{
		ChargeID: chargeID,
		Limit:    int64(limit),
		Offset:   int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list refunds: %w", err)
	}

	refunds := make([]*models.Refund, 0, len(sqlcRefunds))
	for _, sqlcRefund := range sqlcRefunds {
		refund, release, err := r.getFromPool(ctx)
		if err != nil {
			r.logger.Error("Failed to get refund from pool", zap.Error(err))
			continue
		}

		*refund = *models.NewRefund().ConvertFromSQLCRefund(sqlcRefund)
		refunds = append(refunds, refund)

		// Update cache for individual refund
		individualCacheKey := fmt.Sprintf("refund:%s", refund.ID)
		if err = r.cache.Set(ctx, individualCacheKey, refund); err != nil {
			r.logger.Warn("Failed to cache individual refund", zap.Error(err), zap.String("id", refund.ID))
		}

		release()
	}

	// Cache the list of refunds
	if err = r.cache.Set(ctx, cacheKey, refunds); err != nil {
		r.logger.Warn("Failed to cache refunds list", zap.Error(err), zap.String("chargeID", chargeID))
	}

	return refunds, nil
}

func (r *repository) ListByChargeID(ctx context.Context, chargeID string) ([]*models.Refund, error) {
	cacheKey := fmt.Sprintf("refunds:chargeID:%s", chargeID)
	var cachedRefunds []*models.Refund
	found, err := r.cache.Get(ctx, cacheKey, &cachedRefunds)
	if err != nil {
		r.logger.Warn("Failed to get refunds from cache", zap.Error(err), zap.String("chargeID", chargeID))
	} else if found {
		return cachedRefunds, nil
	}

	sqlcRefunds, err := sqlc.New(r.conn).ListByChargeID(ctx, chargeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list refunds by Stripe ID: %w", err)
	}

	refunds := make([]*models.Refund, 0, len(sqlcRefunds))
	for _, sqlcRefund := range sqlcRefunds {
		refund, release, err := r.getFromPool(ctx)
		if err != nil {
			r.logger.Error("Failed to get refund from pool", zap.Error(err))
			continue
		}

		*refund = *models.NewRefund().ConvertFromSQLCRefund(sqlcRefund)
		refunds = append(refunds, refund)

		// Update cache for individual refund
		individualCacheKey := fmt.Sprintf("refund:%s", refund.ID)
		if err = r.cache.Set(ctx, individualCacheKey, refund); err != nil {
			r.logger.Warn("Failed to cache individual refund", zap.Error(err), zap.String("id", refund.ID))
		}

		release()
	}

	// Cache the list of refunds
	if err = r.cache.Set(ctx, cacheKey, refunds); err != nil {
		r.logger.Warn("Failed to cache refunds list", zap.Error(err), zap.String("chargeID", chargeID))
	}

	return refunds, nil
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, refund *models.PartialRefund) error {
	query := `
    INSERT INTO refunds (id, charge_id, amount, status, reason, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{refund.ID}
	var updateClauses []string
	argIndex := 2

	if refund.ChargeID != nil {
		args = append(args, *refund.ChargeID)
		updateClauses = append(updateClauses, fmt.Sprintf("charge_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if refund.Amount != nil {
		args = append(args, *refund.Amount)
		updateClauses = append(updateClauses, fmt.Sprintf("amount = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if refund.Status != nil {
		args = append(args, *refund.Status)
		updateClauses = append(updateClauses, fmt.Sprintf("status = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if refund.Reason != nil {
		args = append(args, *refund.Reason)
		updateClauses = append(updateClauses, fmt.Sprintf("reason = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if refund.CreatedAt != nil {
		args = append(args, *refund.CreatedAt)
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
		return fmt.Errorf("failed to upsert refund: %w", err)
	}

	return nil
}
