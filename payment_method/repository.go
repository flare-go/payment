package payment_method

import (
	"context"
	"errors"
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
	Create(ctx context.Context, tx pgx.Tx, paymentMethod *models.PaymentMethod) error
	GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*AutoReleasePaymentMethod, error)
	Update(ctx context.Context, tx pgx.Tx, paymentMethod *models.PaymentMethod) error
	Delete(ctx context.Context, tx pgx.Tx, id uint64) error
	List(ctx context.Context, tx pgx.Tx, customerID uint64, limit, offset uint64) (*AutoReleasePaymentMethods, error)
}

type repository struct {
	conn        driver.PostgresPool
	logger      *zap.Logger
	cache       *ember.MultiCache
	poolManager ignite.Manager
}

type AutoReleasePaymentMethod struct {
	*models.PaymentMethod
	release func()
}

type AutoReleasePaymentMethods struct {
	PaymentMethods []*models.PaymentMethod
	release        func()
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger, cache *ember.MultiCache, poolManager ignite.Manager) (Repository, error) {
	err := poolManager.RegisterPool(reflect.TypeOf(&models.PaymentMethod{}), ignite.Config[any]{
		InitialSize: 10,
		MaxSize:     100,
		MaxIdleTime: 10 * time.Minute,
		Factory: func() (any, error) {
			return models.NewPaymentMethod(), nil
		},
		Reset: func(obj any) error {
			pm := obj.(*models.PaymentMethod)
			*pm = models.PaymentMethod{}
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register payment method pool: %w", err)
	}

	return &repository{
		conn:        conn,
		logger:      logger,
		cache:       cache,
		poolManager: poolManager,
	}, nil
}

func (r *repository) getFromPool(ctx context.Context) (*models.PaymentMethod, func(), error) {
	pool, err := r.poolManager.GetPool(reflect.TypeOf(&models.PaymentMethod{}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool: %w", err)
	}

	objWrapper, err := pool.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object from pool: %w", err)
	}

	pm := objWrapper.Object.(*models.PaymentMethod)
	release := func() {
		pool.Put(objWrapper)
	}

	return pm, release, nil
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, paymentMethod *models.PaymentMethod) error {
	err := sqlc.New(r.conn).WithTx(tx).CreatePaymentMethod(ctx, sqlc.CreatePaymentMethodParams{
		CustomerID:          paymentMethod.CustomerID,
		Type:                sqlc.PaymentMethodType(paymentMethod.Type),
		CardLast4:           &paymentMethod.CardLast4,
		CardBrand:           &paymentMethod.CardBrand,
		CardExpMonth:        &paymentMethod.CardExpMonth,
		CardExpYear:         &paymentMethod.CardExpYear,
		BankAccountLast4:    &paymentMethod.BankAccountLast4,
		BankAccountBankName: &paymentMethod.BankAccountBankName,
		IsDefault:           paymentMethod.IsDefault,
		StripeID:            paymentMethod.StripeID,
	})
	if err != nil {
		return fmt.Errorf("failed to create payment method: %w", err)
	}

	// 將新創建的支付方法加入緩存
	cacheKey := fmt.Sprintf("payment_method:%d", paymentMethod.ID)
	if err = r.cache.Set(ctx, cacheKey, paymentMethod); err != nil {
		r.logger.Warn("Failed to cache new payment method", zap.Error(err), zap.Uint64("id", paymentMethod.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*AutoReleasePaymentMethod, error) {
	cacheKey := fmt.Sprintf("payment_method:%d", id)

	// 嘗試從緩存中獲取
	var cachedPM models.PaymentMethod
	found, err := r.cache.Get(ctx, cacheKey, &cachedPM)
	if err != nil {
		r.logger.Warn("Failed to get payment method from cache", zap.Error(err), zap.Uint64("id", id))
	} else if found {
		pm, release, err := r.getFromPool(ctx)
		if err != nil {
			return nil, err
		}
		*pm = cachedPM
		return &AutoReleasePaymentMethod{
			PaymentMethod: pm,
			release:       release,
		}, nil
	}

	// 如果緩存中沒有，從數據庫獲取
	pm, release, err := r.getFromPool(ctx)
	if err != nil {
		return nil, err
	}

	sqlcPaymentMethod, err := sqlc.New(r.conn).WithTx(tx).GetPaymentMethod(ctx, id)
	if err != nil {
		release()
		return nil, fmt.Errorf("failed to get payment method: %w", err)
	}

	*pm = *models.NewPaymentMethod().ConvertFromSQLCPaymentMethod(sqlcPaymentMethod)

	// 更新緩存
	if err := r.cache.Set(ctx, cacheKey, pm); err != nil {
		r.logger.Warn("Failed to cache payment method", zap.Error(err), zap.Uint64("id", id))
	}

	return &AutoReleasePaymentMethod{
		PaymentMethod: pm,
		release:       release,
	}, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, paymentMethod *models.PaymentMethod) error {
	// 首先獲取當前的支付方法，以獲取 updated_at 值
	currentPM, err := sqlc.New(r.conn).WithTx(tx).GetPaymentMethod(ctx, paymentMethod.ID)
	if err != nil {
		return fmt.Errorf("failed to get current payment method: %w", err)
	}

	// 嘗試更新支付方法
	err = sqlc.New(r.conn).WithTx(tx).UpdatePaymentMethod(ctx, sqlc.UpdatePaymentMethodParams{
		ID:                  paymentMethod.ID,
		Type:                sqlc.PaymentMethodType(paymentMethod.Type),
		CardLast4:           &paymentMethod.CardLast4,
		CardBrand:           &paymentMethod.CardBrand,
		CardExpMonth:        &paymentMethod.CardExpMonth,
		CardExpYear:         &paymentMethod.CardExpYear,
		BankAccountLast4:    &paymentMethod.BankAccountLast4,
		BankAccountBankName: &paymentMethod.BankAccountBankName,
		IsDefault:           paymentMethod.IsDefault,
		StripeID:            paymentMethod.StripeID,
		UpdatedAt:           currentPM.UpdatedAt, // 使用當前的 updated_at 值
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("payment method not found: %w", err)
		}
		return fmt.Errorf("failed to update payment method: %w", err)
	}

	// 更新緩存
	cacheKey := fmt.Sprintf("payment_method:%d", paymentMethod.ID)
	if err := r.cache.Set(ctx, cacheKey, paymentMethod); err != nil {
		r.logger.Warn("Failed to update payment method in cache", zap.Error(err), zap.Uint64("id", paymentMethod.ID))
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id uint64) error {
	err := sqlc.New(r.conn).WithTx(tx).DeletePaymentMethod(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete payment method: %w", err)
	}

	// 從緩存中刪除
	cacheKey := fmt.Sprintf("payment_method:%d", id)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete payment method from cache", zap.Error(err), zap.Uint64("id", id))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, customerID uint64, limit, offset uint64) (*AutoReleasePaymentMethods, error) {

	sqlcPaymentMethods, err := sqlc.New(r.conn).WithTx(tx).ListPaymentMethods(ctx, sqlc.ListPaymentMethodsParams{
		CustomerID: customerID,
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}

	paymentMethods := make([]*models.PaymentMethod, 0, len(sqlcPaymentMethods))
	var releaseFunc []func()

	for _, sqlcPM := range sqlcPaymentMethods {
		pm, release, err := r.getFromPool(ctx)
		if err != nil {
			for _, rf := range releaseFunc {
				rf()
			}
			return nil, fmt.Errorf("failed to get payment method from pool: %w", err)
		}
		*pm = *models.NewPaymentMethod().ConvertFromSQLCPaymentMethod(sqlcPM)
		paymentMethods = append(paymentMethods, pm)
		releaseFunc = append(releaseFunc, release)

		// 更新每個 PaymentMethod 的緩存
		cacheKey := fmt.Sprintf("payment_method:%d", pm.ID)
		if err = r.cache.Set(ctx, cacheKey, pm); err != nil {
			r.logger.Warn("Failed to cache payment method during list", zap.Error(err), zap.Uint64("id", pm.ID))
		}
	}

	return &AutoReleasePaymentMethods{
		PaymentMethods: paymentMethods,
		release: func() {
			for _, rf := range releaseFunc {
				rf()
			}
		},
	}, nil
}
