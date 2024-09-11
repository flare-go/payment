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
	GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Price, error)
	Update(ctx context.Context, tx pgx.Tx, price *models.Price) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
	List(ctx context.Context, tx pgx.Tx, productID string) ([]*models.Price, error)
	ListActive(ctx context.Context, tx pgx.Tx, productID string) ([]*models.Price, error)
	Upsert(ctx context.Context, tx pgx.Tx, price *models.PartialPrice) error
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
		MaxIdleTime: 10 * time.Minute,
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
	if err := sqlc.New(r.conn).WithTx(tx).CreatePrice(ctx, sqlc.CreatePriceParams{
		ID:                     price.ID,
		ProductID:              price.ProductID,
		Type:                   sqlc.PriceType(price.Type),
		Currency:               sqlc.Currency(price.Currency),
		UnitAmount:             price.UnitAmount,
		RecurringInterval:      sqlc.NullPriceRecurringInterval{PriceRecurringInterval: sqlc.PriceRecurringInterval(price.RecurringInterval), Valid: price.RecurringInterval != ""},
		RecurringIntervalCount: price.RecurringIntervalCount,
		TrialPeriodDays:        price.TrialPeriodDays,
	}); err != nil {
		return fmt.Errorf("failed to create price: %w", err)
	}

	// 將新創建的價格加入緩存
	cacheKey := fmt.Sprintf("price:%s", price.ID)
	if err := r.cache.Set(ctx, cacheKey, price); err != nil {
		r.logger.Warn("Failed to cache new price", zap.Error(err), zap.String("id", price.ID))
	}
	cacheKey = fmt.Sprintf("prices:product:%s", price.ProductID)
	var cachedPrices []*models.Price
	if _, err := r.cache.Get(ctx, cacheKey, &cachedPrices); err == nil {
		cachedPrices = append(cachedPrices, price)
		if err = r.cache.Set(ctx, cacheKey, cachedPrices); err != nil {
			r.logger.Warn("Failed to cache prices list", zap.Error(err), zap.String("productID", price.ProductID))
		}
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Price, error) {
	cacheKey := fmt.Sprintf("price:%s", id)

	price, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	found, err := r.cache.Get(ctx, cacheKey, price)
	if err != nil {
		r.logger.Warn("Failed to get price from cache", zap.Error(err), zap.String("id", id))
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
	if err = r.cache.Set(ctx, cacheKey, price); err != nil {
		r.logger.Warn("Failed to cache price", zap.Error(err), zap.String("id", id))
	}

	return price, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, price *models.Price) error {
	if err := sqlc.New(r.conn).WithTx(tx).UpdatePrice(ctx, sqlc.UpdatePriceParams{
		ID:                     price.ID,
		ProductID:              price.ProductID,
		Type:                   sqlc.PriceType(price.Type),
		Currency:               sqlc.Currency(price.Currency),
		UnitAmount:             price.UnitAmount,
		RecurringInterval:      sqlc.NullPriceRecurringInterval{PriceRecurringInterval: sqlc.PriceRecurringInterval(price.RecurringInterval), Valid: price.RecurringInterval != ""},
		RecurringIntervalCount: price.RecurringIntervalCount,
		TrialPeriodDays:        price.TrialPeriodDays,
		Active:                 price.Active,
	}); err != nil {
		return fmt.Errorf("failed to update price: %w", err)
	}

	// 更新緩存中的價格信息
	cacheKey := fmt.Sprintf("price:%s", price.ID)
	if err := r.cache.Set(ctx, cacheKey, price); err != nil {
		r.logger.Warn("Failed to update price in cache", zap.Error(err), zap.String("id", price.ID))
	}
	cacheKey = fmt.Sprintf("prices:product:%s", price.ProductID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete price from cache", zap.Error(err), zap.String("productID", price.ProductID))
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	productID, err := sqlc.New(r.conn).WithTx(tx).DeletePrice(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete price: %w", err)
	}

	// 從緩存中刪除價格
	cacheKey := fmt.Sprintf("price:%s", id)
	if err = r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete price from cache", zap.Error(err), zap.String("id", id))
	}
	cacheKey = fmt.Sprintf("prices:product:%s", productID)
	if err = r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete price from cache", zap.Error(err), zap.String("id", id))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, productID string) ([]*models.Price, error) {
	cacheKey := fmt.Sprintf("prices:product:%s", productID)
	var cachedPrices []*models.Price
	found, err := r.cache.Get(ctx, cacheKey, &cachedPrices)
	if err != nil {
		r.logger.Warn("Failed to get prices from cache", zap.Error(err), zap.String("productID", productID))
	} else if found {
		return cachedPrices, nil
	}

	sqlcPrices, err := sqlc.New(r.conn).WithTx(tx).ListPrices(ctx, productID)
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
		singleCacheKey := fmt.Sprintf("price:%s", price.ID)
		if err = r.cache.Set(ctx, singleCacheKey, price); err != nil {
			r.logger.Warn("Failed to cache single price", zap.Error(err), zap.String("id", price.ID))
		}

		release()
	}

	// 緩存整個列表結果
	if err = r.cache.Set(ctx, cacheKey, prices); err != nil {
		r.logger.Warn("Failed to cache prices list", zap.Error(err), zap.String("productID", productID))
	}

	return prices, nil
}

func (r *repository) ListActive(ctx context.Context, tx pgx.Tx, productID string) ([]*models.Price, error) {
	cacheKey := fmt.Sprintf("prices:product:%s", productID)
	var cachedPrices []*models.Price
	found, err := r.cache.Get(ctx, cacheKey, &cachedPrices)
	if err != nil {
		r.logger.Warn("Failed to get prices from cache", zap.Error(err), zap.String("productID", productID))
	} else if found {
		return cachedPrices, nil
	}

	sqlcPrices, err := sqlc.New(r.conn).WithTx(tx).ListActivePrices(ctx, productID)
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
		singleCacheKey := fmt.Sprintf("price:%s", price.ID)
		if err := r.cache.Set(ctx, singleCacheKey, price); err != nil {
			r.logger.Warn("Failed to cache single price", zap.Error(err), zap.String("id", price.ID))
		}

		release()
	}

	// 緩存整個列表結果
	if err = r.cache.Set(ctx, cacheKey, prices); err != nil {
		r.logger.Warn("Failed to cache prices list", zap.Error(err), zap.String("productID", productID))
	}

	return prices, nil
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, price *models.PartialPrice) error {
	const query = `
    INSERT INTO prices (id, product_id, active, currency, unit_amount, type, recurring_interval, recurring_interval_count, trial_period_days, created_at, updated_at)
    VALUES (@id, @product_id, @active, @currency, @unit_amount, @type, @recurring_interval, COALESCE(@recurring_interval_count, 1), @trial_period_days, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        product_id = COALESCE(@product_id, prices.product_id),
        active = COALESCE(@active, prices.active),
        currency = COALESCE(@currency, prices.currency),
        unit_amount = COALESCE(@unit_amount, prices.unit_amount),
        type = COALESCE(@type, prices.type),
        recurring_interval = COALESCE(@recurring_interval, prices.recurring_interval),
        recurring_interval_count = COALESCE(@recurring_interval_count, prices.recurring_interval_count, 1),
        trial_period_days = COALESCE(@trial_period_days, prices.trial_period_days),
        updated_at = @updated_at
    WHERE prices.id = @id
    `

	now := time.Now()
	ric := 1
	var tpd int32
	if price.RecurringIntervalCount != nil {
		ric = int(*price.RecurringIntervalCount)
	}
	if price.TrialPeriodDays != nil {
		tpd = *price.TrialPeriodDays
	}
	args := pgx.NamedArgs{
		"id":                       price.ID,
		"product_id":               price.ProductID,
		"active":                   price.Active,
		"currency":                 price.Currency,
		"unit_amount":              price.UnitAmount,
		"type":                     price.Type,
		"recurring_interval":       price.RecurringInterval,
		"recurring_interval_count": ric,
		"trial_period_days":        tpd,
		"created_at":               now,
		"updated_at":               now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert price: %w", err)
	}

	return nil
}
