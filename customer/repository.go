package customer

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

var _ Repository = (*repository)(nil)

type Repository interface {
	Create(ctx context.Context, tx pgx.Tx, customer *models.Customer) error
	GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Customer, error)
	Update(ctx context.Context, tx pgx.Tx, customer *models.Customer) error
	Upsert(ctx context.Context, tx pgx.Tx, customer *models.PartialCustomer) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
	List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.Customer, error)
	UpdateBalance(ctx context.Context, tx pgx.Tx, id string, amount uint64) error
}

type repository struct {
	conn        driver.PostgresPool
	logger      *zap.Logger
	cache       *ember.MultiCache
	poolManager ignite.Manager
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger, cache *ember.MultiCache, poolManager ignite.Manager) (Repository, error) {
	if err := poolManager.RegisterPool(reflect.TypeOf(&models.Customer{}), ignite.Config[any]{
		InitialSize: 10,
		MaxSize:     100,
		MaxIdleTime: 10 * time.Minute,
		Factory: func() (any, error) {
			return models.NewCustomer(), nil
		},
		Reset: func(obj any) error {
			c := obj.(*models.Customer)
			*c = models.Customer{}
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to register customer pool: %w", err)
	}

	return &repository{
		conn:        conn,
		logger:      logger,
		cache:       cache,
		poolManager: poolManager,
	}, nil
}

func (r *repository) getFromPool(ctx context.Context) (*models.Customer, func(), error) {
	pool, err := r.poolManager.GetPool(reflect.TypeOf(&models.Customer{}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool: %w", err)
	}

	objWrapper, err := pool.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object from pool: %w", err)
	}

	customer := objWrapper.Object.(*models.Customer)
	release := func() {
		pool.Put(objWrapper)
	}

	return customer, release, nil
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, customer *models.Customer) error {

	sqlcCustomer, err := sqlc.New(r.conn).WithTx(tx).CreateCustomer(ctx, sqlc.CreateCustomerParams{
		ID:        customer.ID,
		UserEmail: customer.Email,
		Balance:   customer.Balance,
	})
	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}

	// 將新創建的客戶加入緩存
	customer.ID = sqlcCustomer.ID
	customer.CreatedAt = sqlcCustomer.CreatedAt.Time
	customer.UpdatedAt = sqlcCustomer.UpdatedAt.Time
	cacheKey := fmt.Sprintf("customer:%s", customer.ID)
	if err = r.cache.Set(ctx, cacheKey, &customer); err != nil {
		r.logger.Warn("Failed to cache new customer", zap.Error(err), zap.String("id", customer.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Customer, error) {
	cacheKey := fmt.Sprintf("customer:%s", id)

	// 嘗試從緩存中獲取
	var cachedCustomer models.Customer
	found, err := r.cache.Get(ctx, cacheKey, &cachedCustomer)
	if err != nil {
		r.logger.Warn("Failed to get customer from cache", zap.Error(err), zap.String("id", id))
	} else if found {
		return &cachedCustomer, nil
	}

	// 如果緩存中沒有，從數據庫獲取
	sqlcCustomer, err := sqlc.New(r.conn).WithTx(tx).GetCustomer(ctx, &id)
	if err != nil {
		r.logger.Error("error getting customer", zap.Error(err))
		return nil, err
	}

	customer, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer from pool: %w", err)
	}
	defer release()

	*customer = *models.NewCustomer().ConvertFromSQLCCustomer(sqlcCustomer)

	// 更新緩存
	if err = r.cache.Set(ctx, cacheKey, customer); err != nil {
		r.logger.Warn("Failed to cache customer", zap.Error(err), zap.String("id", id))
	}

	return customer, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, customer *models.Customer) error {
	if err := sqlc.New(r.conn).WithTx(tx).UpdateCustomer(ctx, sqlc.UpdateCustomerParams{
		ID:      customer.ID,
		Balance: customer.Balance,
	}); err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	// 更新緩存
	cacheKey := fmt.Sprintf("customer:%s", customer.ID)
	if err := r.cache.Set(ctx, cacheKey, customer); err != nil {
		r.logger.Warn("Failed to update customer in cache", zap.Error(err), zap.String("id", customer.ID))
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	if err := sqlc.New(r.conn).WithTx(tx).DeleteCustomer(ctx, id); err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	// 從緩存中刪除
	cacheKey := fmt.Sprintf("customer:%s", id)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete customer from cache", zap.Error(err), zap.String("id", id))
	}

	return r.cache.Delete(ctx, cacheKey)
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.Customer, error) {

	column1 := int64(limit)
	column2 := int64(offset)
	sqlcCustomers, err := sqlc.New(r.conn).WithTx(tx).ListCustomers(ctx, sqlc.ListCustomersParams{
		Column1: &column1,
		Column2: &column2,
	})
	if err != nil {
		r.logger.Error("error listing customers", zap.Error(err))
		return nil, err
	}

	customers := make([]*models.Customer, 0, len(sqlcCustomers))
	for _, sqlcCustomer := range sqlcCustomers {
		customer, release, err := r.getFromPool(ctx)
		if err != nil {
			r.logger.Error("Failed to get customer from pool", zap.Error(err))
			continue
		}

		*customer = *models.NewCustomer().ConvertFromSQLCCustomer(sqlcCustomer)
		customers = append(customers, customer)

		// 更新每個客戶的緩存
		cacheKey := fmt.Sprintf("customer:%s", customer.ID)
		if err := r.cache.Set(ctx, cacheKey, customer); err != nil {
			r.logger.Warn("Failed to cache customer during list", zap.Error(err), zap.String("id", customer.ID))
		}
		release()
	}

	return customers, nil
}

func (r *repository) UpdateBalance(ctx context.Context, tx pgx.Tx, id string, balance uint64) error {
	if err := sqlc.New(r.conn).WithTx(tx).UpdateCustomerBalance(ctx, sqlc.UpdateCustomerBalanceParams{
		ID:      id,
		Balance: int64(balance),
	}); err != nil {
		return fmt.Errorf("failed to update customer balance: %w", err)
	}

	// 更新緩存中的餘額
	cacheKey := fmt.Sprintf("customer:%s", id)
	var cachedCustomer models.Customer
	found, err := r.cache.Get(ctx, cacheKey, &cachedCustomer)
	if err != nil {
		r.logger.Warn("Failed to get customer from cache for balance update", zap.Error(err), zap.String("id", id))
	} else if found {
		cachedCustomer.Balance = int64(balance)
		if err := r.cache.Set(ctx, cacheKey, &cachedCustomer); err != nil {
			r.logger.Warn("Failed to update customer balance in cache", zap.Error(err), zap.String("id", id))
		}
	}

	return nil
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, customer *models.PartialCustomer) error {
	const query = `
    INSERT INTO customers (id, user_email, balance, created_at, updated_at)
    VALUES (@id, @user_email, @balance, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        balance = COALESCE(@balance, customers.balance),
        updated_at = @updated_at
    WHERE customers.id = @id
    `

	var balance int64
	if customer.Balance != nil {
		balance = *customer.Balance
	}

	now := time.Now()
	args := pgx.NamedArgs{
		"id":         customer.ID,
		"user_email": customer.Email,
		"balance":    balance,
		"created_at": customer.CreatedAt,
		"updated_at": now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert customer: %w", err)
	}

	// 更新或清除緩存
	cacheKey := fmt.Sprintf("customer:%s", customer.ID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete customer from cache", zap.Error(err), zap.String("id", customer.ID))
	}

	return nil
}
