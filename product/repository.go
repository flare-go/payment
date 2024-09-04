package product

import (
	"context"
	"encoding/json"
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
	Create(ctx context.Context, tx pgx.Tx, product *models.Product) error
	GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Product, error)
	Update(ctx context.Context, tx pgx.Tx, product *models.Product) error
	Delete(ctx context.Context, tx pgx.Tx, id uint64) error
	List(ctx context.Context, tx pgx.Tx, limit, offset uint64, activeOnly bool) ([]*models.Product, error)
}

type repository struct {
	conn        driver.PostgresPool
	logger      *zap.Logger
	cache       *ember.MultiCache
	poolManager ignite.Manager
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger, cache *ember.MultiCache, poolManager ignite.Manager) (Repository, error) {
	err := poolManager.RegisterPool(reflect.TypeOf(&models.Product{}), ignite.Config[any]{
		InitialSize: 10,
		MaxSize:     100,
		Factory: func() (any, error) {
			return models.NewProduct(), nil
		},
		Reset: func(obj any) error {
			p := obj.(*models.Product)
			*p = models.Product{}
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register product pool: %w", err)
	}

	return &repository{
		conn:        conn,
		logger:      logger,
		cache:       cache,
		poolManager: poolManager,
	}, nil
}

func (r *repository) getFromPool(ctx context.Context) (*models.Product, func(), error) {
	pool, err := r.poolManager.GetPool(reflect.TypeOf(&models.Product{}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool: %w", err)
	}

	objWrapper, err := pool.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object from pool: %w", err)
	}

	product := objWrapper.Object.(*models.Product)
	release := func() {
		pool.Put(objWrapper)
	}

	return product, release, nil
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, product *models.Product) error {
	metadata, err := json.Marshal(product.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	err = sqlc.New(r.conn).WithTx(tx).CreateProduct(ctx, sqlc.CreateProductParams{
		Name:        product.Name,
		Description: &product.Description,
		Active:      product.Active,
		Metadata:    metadata,
		StripeID:    product.StripeID,
	})
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	// 將新創建的產品加入緩存
	cacheKey := fmt.Sprintf("product:%d", product.ID)
	if err := r.cache.Set(ctx, cacheKey, product, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache new product", zap.Error(err), zap.Uint64("id", product.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Product, error) {
	cacheKey := fmt.Sprintf("product:%d", id)

	product, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	found, err := r.cache.Get(ctx, cacheKey, product)
	if err != nil {
		r.logger.Warn("Failed to get product from cache", zap.Error(err), zap.Uint64("id", id))
	} else if found {
		return product, nil
	}

	sqlcProduct, err := sqlc.New(r.conn).WithTx(tx).GetProduct(ctx, id)
	if err != nil {
		r.logger.Error("error getting product", zap.Error(err))
		return nil, err
	}

	*product = *models.NewProduct().ConvertFromSQLCProduct(sqlcProduct)

	// 更新緩存
	if err := r.cache.Set(ctx, cacheKey, product, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache product", zap.Error(err), zap.Uint64("id", id))
	}

	return product, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, product *models.Product) error {
	metadata, err := json.Marshal(product.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	err = sqlc.New(r.conn).WithTx(tx).UpdateProduct(ctx, sqlc.UpdateProductParams{
		ID:          product.ID,
		Name:        product.Name,
		Description: &product.Description,
		Active:      product.Active,
		Metadata:    metadata,
		StripeID:    product.StripeID,
	})
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	// 更新緩存中的產品信息
	cacheKey := fmt.Sprintf("product:%d", product.ID)
	if err := r.cache.Set(ctx, cacheKey, product, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to update product in cache", zap.Error(err), zap.Uint64("id", product.ID))
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id uint64) error {
	err := sqlc.New(r.conn).WithTx(tx).DeleteProduct(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	// 從緩存中刪除產品
	cacheKey := fmt.Sprintf("product:%d", id)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete product from cache", zap.Error(err), zap.Uint64("id", id))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, limit, offset uint64, activeOnly bool) ([]*models.Product, error) {
	cacheKey := fmt.Sprintf("products:limit:%d:offset:%d:active:%v", limit, offset, activeOnly)
	var cachedProducts []*models.Product
	found, err := r.cache.Get(ctx, cacheKey, &cachedProducts)
	if err != nil {
		r.logger.Warn("Failed to get products from cache", zap.Error(err))
	} else if found {
		return cachedProducts, nil
	}

	sqlcProducts, err := sqlc.New(r.conn).WithTx(tx).ListProducts(ctx, sqlc.ListProductsParams{
		Active: activeOnly,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		r.logger.Error("error listing products", zap.Error(err))
		return nil, err
	}

	products := make([]*models.Product, 0, len(sqlcProducts))
	for _, sqlcProduct := range sqlcProducts {
		product, release, err := r.getFromPool(ctx)
		if err != nil {
			r.logger.Error("Failed to get product from pool", zap.Error(err))
			continue
		}

		*product = *models.NewProduct().ConvertFromSQLCProduct(sqlcProduct)
		products = append(products, product)

		// 更新單個產品的緩存
		singleCacheKey := fmt.Sprintf("product:%d", product.ID)
		if err := r.cache.Set(ctx, singleCacheKey, product, 30*time.Minute); err != nil {
			r.logger.Warn("Failed to cache single product", zap.Error(err), zap.Uint64("id", product.ID))
		}

		release()
	}

	// 緩存整個列表結果
	if err := r.cache.Set(ctx, cacheKey, products, 5*time.Minute); err != nil {
		r.logger.Warn("Failed to cache products list", zap.Error(err))
	}

	return products, nil
}
