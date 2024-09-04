package price

import (
	"context"
	"fmt"
	"reflect"
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
	Create(ctx context.Context, tx pgx.Tx, price *models.Price) error
	GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Price, error)
	Update(ctx context.Context, tx pgx.Tx, price *models.Price) error
	Delete(ctx context.Context, tx pgx.Tx, id uint64) error
	List(ctx context.Context, tx pgx.Tx, productID uint64, limit, offset uint64, activeOnly bool) ([]*models.Price, error)
}

type repository struct {
	conn        driver.PostgresPool
	logger      *zap.Logger
	cache       *ember.MultiCache
	poolManager ignite.Manager
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger, cache *ember.MultiCache, poolManager ignite.Manager) (Repository, error) {
	err := poolManager.RegisterPool(reflect.TypeOf(&models.Price{}), ignite.Config[any]{
		InitialSize: 10,
		MaxSize:     100,
		Factory: func() (any, error) {
			return models.NewPrice(), nil
		},
		Reset: func(obj any) error {
			p := obj.(*models.Price)
			*p = models.Price{}
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register price pool: %w", err)
	}

	return &repository{
		conn:        conn,
		logger:      logger,
		cache:       cache,
		poolManager: poolManager,
	}, nil
}

func (r *repository) getFromPool(ctx context.Context) (*models.Price, func(), error) {
	pool, err := r.poolManager.GetPool(reflect.TypeOf(&models.Price{}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool: %w", err)
	}

	objWrapper, err := pool.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object from pool: %w", err)
	}

	price := objWrapper.Object.(*models.Price)
	release := func() {
		pool.Put(objWrapper)
	}

	return price, release, nil
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, price *models.Price) error {
	err := sqlc.New(r.conn).WithTx(tx).CreatePrice(ctx, sqlc.CreatePriceParams{
		ProductID:              price.ProductID,
		Type:                   sqlc.PriceType(price.Type),
		Currency:               sqlc.Currency(price.Currency),
		UnitAmount:             price.UnitAmount,
		RecurringInterval:      sqlc.NullIntervalType{IntervalType: sqlc.IntervalType(price.RecurringInterval), Valid: price.RecurringInterval != ""},
		RecurringIntervalCount: price.RecurringIntervalCount,
		TrialPeriodDays:        price.TrialPeriodDays,
		Active:                 price.Active,
		StripeID:               price.StripeID,
	})
	if err != nil {
		return fmt.Errorf("failed to create price: %w", err)
	}

	// 將新創建的價格加入緩存
	cacheKey := fmt.Sprintf("price:%d", price.ID)
	if err := r.cache.Set(ctx, cacheKey, price, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache new price", zap.Error(err), zap.Uint64("id", price.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Price, error) {
	cacheKey := fmt.Sprintf("price:%d", id)

	price, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	found, err := r.cache.Get(ctx, cacheKey, price)
	if err != nil {
		r.logger.Warn("Failed to get price from cache", zap.Error(err), zap.Uint64("id", id))
	} else if found {
		return price, nil
	}

	sqlcPrice, err := sqlc.New(r.conn).WithTx(tx).GetPrice(ctx, id)
	if err != nil {
		r.logger.Error("error getting price", zap.Error(err))
		return nil, err
	}

	*price = *models.NewPrice().ConvertFromSQLCPrice(sqlcPrice)

	// 更新緩存
	if err := r.cache.Set(ctx, cacheKey, price, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache price", zap.Error(err), zap.Uint64("id", id))
	}

	return price, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, price *models.Price) error {
	err := sqlc.New(r.conn).WithTx(tx).UpdatePrice(ctx, sqlc.UpdatePriceParams{
		ID:                     price.ID,
		ProductID:              price.ProductID,
		Type:                   sqlc.PriceType(price.Type),
		Currency:               sqlc.Currency(price.Currency),
		UnitAmount:             price.UnitAmount,
		RecurringInterval:      sqlc.NullIntervalType{IntervalType: sqlc.IntervalType(price.RecurringInterval), Valid: price.RecurringInterval != ""},
		RecurringIntervalCount: price.RecurringIntervalCount,
		TrialPeriodDays:        price.TrialPeriodDays,
		Active:                 price.Active,
		StripeID:               price.StripeID,
	})
	if err != nil {
		return fmt.Errorf("failed to update price: %w", err)
	}

	// 更新緩存中的價格信息
	cacheKey := fmt.Sprintf("price:%d", price.ID)
	if err := r.cache.Set(ctx, cacheKey, price, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to update price in cache", zap.Error(err), zap.Uint64("id", price.ID))
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id uint64) error {
	err := sqlc.New(r.conn).WithTx(tx).DeletePrice(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete price: %w", err)
	}

	// 從緩存中刪除價格
	cacheKey := fmt.Sprintf("price:%d", id)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete price from cache", zap.Error(err), zap.Uint64("id", id))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, productID uint64, limit, offset uint64, activeOnly bool) ([]*models.Price, error) {
	cacheKey := fmt.Sprintf("prices:product:%d:limit:%d:offset:%d:active:%v", productID, limit, offset, activeOnly)
	var cachedPrices []*models.Price
	found, err := r.cache.Get(ctx, cacheKey, &cachedPrices)
	if err != nil {
		r.logger.Warn("Failed to get prices from cache", zap.Error(err), zap.Uint64("productID", productID))
	} else if found {
		return cachedPrices, nil
	}

	sqlcPrices, err := sqlc.New(r.conn).WithTx(tx).ListPrices(ctx, sqlc.ListPricesParams{
		ProductID: productID,
		Active:    activeOnly,
		Limit:     int64(limit),
		Offset:    int64(offset),
	})
	if err != nil {
		r.logger.Error("error listing prices", zap.Error(err))
		return nil, err
	}

	prices := make([]*models.Price, 0, len(sqlcPrices))
	for _, sqlcPrice := range sqlcPrices {
		price, release, err := r.getFromPool(ctx)
		if err != nil {
			r.logger.Error("Failed to get price from pool", zap.Error(err))
			continue
		}

		*price = *models.NewPrice().ConvertFromSQLCPrice(sqlcPrice)
		prices = append(prices, price)

		// 更新單個價格的緩存
		singleCacheKey := fmt.Sprintf("price:%d", price.ID)
		if err := r.cache.Set(ctx, singleCacheKey, price, 30*time.Minute); err != nil {
			r.logger.Warn("Failed to cache single price", zap.Error(err), zap.Uint64("id", price.ID))
		}

		release()
	}

	// 緩存整個列表結果
	if err := r.cache.Set(ctx, cacheKey, prices, 5*time.Minute); err != nil {
		r.logger.Warn("Failed to cache prices list", zap.Error(err), zap.Uint64("productID", productID))
	}

	return prices, nil
}
