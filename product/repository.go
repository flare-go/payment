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
	GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Product, error)
	Update(ctx context.Context, tx pgx.Tx, product *models.Product) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
	List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.Product, error)
	Upsert(ctx context.Context, tx pgx.Tx, product *models.PartialProduct) error
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
		MaxIdleTime: 10 * time.Minute,
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
	var metadata []byte
	if product.Metadata != nil {
		var err error
		metadata, err = json.Marshal(product.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	sqlcProduct, err := sqlc.New(r.conn).WithTx(tx).CreateProduct(ctx, sqlc.CreateProductParams{
		ID:          product.ID,
		Name:        product.Name,
		Description: &product.Description,
		Active:      product.Active,
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	// 將新創建的產品加入緩存
	product.ID = sqlcProduct.ID
	product.CreatedAt = sqlcProduct.CreatedAt.Time
	product.UpdatedAt = sqlcProduct.UpdatedAt.Time
	cacheKey := fmt.Sprintf("product:%s", product.ID)
	if err = r.cache.Set(ctx, cacheKey, product); err != nil {
		r.logger.Warn("Failed to cache new product", zap.Error(err), zap.String("id", product.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Product, error) {
	cacheKey := fmt.Sprintf("product:%s", id)
	product, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	found, err := r.cache.Get(ctx, cacheKey, product)
	if err != nil {
		r.logger.Warn("Failed to get product from cache", zap.Error(err), zap.String("id", id))
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
	if err = r.cache.Set(ctx, cacheKey, product); err != nil {
		r.logger.Warn("Failed to cache product", zap.Error(err), zap.String("id", id))
	}

	return product, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, product *models.Product) error {
	var metadata []byte
	if product.Metadata != nil {
		var err error
		metadata, err = json.Marshal(product.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	sqlcProduct, err := sqlc.New(r.conn).WithTx(tx).UpdateProduct(ctx, sqlc.UpdateProductParams{
		ID:          product.ID,
		Name:        product.Name,
		Description: &product.Description,
		Active:      product.Active,
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	// 更新緩存中的產品信息
	product.CreatedAt = sqlcProduct.CreatedAt.Time
	product.UpdatedAt = sqlcProduct.UpdatedAt.Time
	cacheKey := fmt.Sprintf("product:%s", product.ID)
	if err = r.cache.Set(ctx, cacheKey, product); err != nil {
		r.logger.Warn("Failed to update product in cache", zap.Error(err), zap.String("id", product.ID))
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	err := sqlc.New(r.conn).WithTx(tx).DeleteProduct(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	// 從緩存中刪除產品
	cacheKey := fmt.Sprintf("product:%s", id)
	if err = r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete product from cache", zap.Error(err), zap.String("id", id))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.Product, error) {
	cacheKey := fmt.Sprintf("products:limit:%d:offset:%d", limit, offset)
	var cachedProducts []*models.Product
	found, err := r.cache.Get(ctx, cacheKey, &cachedProducts)
	if err != nil {
		r.logger.Warn("Failed to get products from cache", zap.Error(err))
	} else if found {
		return cachedProducts, nil
	}

	sqlcProducts, err := sqlc.New(r.conn).WithTx(tx).ListProducts(ctx, sqlc.ListProductsParams{
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
		singleCacheKey := fmt.Sprintf("product:%s", product.ID)
		if err = r.cache.Set(ctx, singleCacheKey, product); err != nil {
			r.logger.Warn("Failed to cache single product", zap.Error(err), zap.String("id", product.ID))
		}

		release()
	}

	// 緩存整個列表結果
	if err = r.cache.Set(ctx, cacheKey, products); err != nil {
		r.logger.Warn("Failed to cache products list", zap.Error(err))
	}

	return products, nil
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, product *models.PartialProduct) error {
	const query = `
    INSERT INTO products (id, name, description, active, metadata, created_at, updated_at)
    VALUES (@id, @name, @description, @active, @metadata, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        name = COALESCE(@name, products.name),
        description = COALESCE(@description, products.description),
        active = COALESCE(@active, products.active),
        metadata = COALESCE(@metadata, products.metadata),
        updated_at = @updated_at
    WHERE products.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":          product.ID,
		"name":        product.Name,
		"description": product.Description,
		"active":      product.Active,
		"metadata":    product.Metadata,
		"created_at":  now,
		"updated_at":  now,
	}

	_, err := tx.Exec(ctx, query, args)
	if err != nil {
		return fmt.Errorf("failed to upsert product: %w", err)
	}

	return nil
}
