package invoice

import (
	"context"
	"fmt"
	"reflect"
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
	Create(ctx context.Context, tx pgx.Tx, invoice *models.Invoice) error
	GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Invoice, error)
	Update(ctx context.Context, tx pgx.Tx, invoice *models.Invoice) error
	List(ctx context.Context, tx pgx.Tx, customerID string, limit, offset uint64) ([]*models.Invoice, error)
	Delete(ctx context.Context, tx pgx.Tx, id string) error
	CreateInvoiceItem(ctx context.Context, tx pgx.Tx, item *models.InvoiceItem) error
	GetInvoiceItemByID(ctx context.Context, tx pgx.Tx, id string) (*models.InvoiceItem, error)
	UpdateInvoiceItem(ctx context.Context, tx pgx.Tx, item *models.InvoiceItem) error
	DeleteInvoiceItem(ctx context.Context, tx pgx.Tx, id string) error
	ListInvoiceItems(ctx context.Context, tx pgx.Tx, invoiceID string) ([]*models.InvoiceItem, error)
	Upsert(ctx context.Context, tx pgx.Tx, invoice *models.PartialInvoice) error
}

type repository struct {
	conn        driver.PostgresPool
	logger      *zap.Logger
	cache       *ember.MultiCache
	poolManager ignite.Manager
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger, cache *ember.MultiCache, poolManager ignite.Manager) (Repository, error) {

	if err := poolManager.RegisterPool(reflect.TypeOf(&models.Invoice{}), ignite.Config[any]{
		InitialSize: 10,
		MaxSize:     100,
		MaxIdleTime: 10 * time.Minute,
		Factory:     func() (any, error) { return models.NewInvoice(), nil },
		Reset:       func(obj any) error { *obj.(*models.Invoice) = models.Invoice{}; return nil },
	}); err != nil {
		return nil, fmt.Errorf("failed to register invoice pool: %w", err)
	}

	if err := poolManager.RegisterPool(reflect.TypeOf(&models.InvoiceItem{}), ignite.Config[any]{
		InitialSize: 20,
		MaxSize:     200,
		MaxIdleTime: 10 * time.Minute,
		Factory:     func() (any, error) { return models.NewInvoiceItem(), nil },
		Reset:       func(obj any) error { *obj.(*models.InvoiceItem) = models.InvoiceItem{}; return nil },
	}); err != nil {
		return nil, fmt.Errorf("failed to register invoice item pool: %w", err)
	}

	return &repository{
		conn:        conn,
		logger:      logger,
		cache:       cache,
		poolManager: poolManager,
	}, nil
}

func (r *repository) getFromPool(ctx context.Context, t reflect.Type) (any, func(), error) {
	pool, err := r.poolManager.GetPool(t)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool: %w", err)
	}

	objWrapper, err := pool.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object from pool: %w", err)
	}

	return objWrapper.Object, func() { pool.Put(objWrapper) }, nil
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, invoice *models.Invoice) error {

	if err := sqlc.New(r.conn).WithTx(tx).CreateInvoice(ctx, sqlc.CreateInvoiceParams{
		ID:              invoice.ID,
		CustomerID:      invoice.CustomerID,
		SubscriptionID:  invoice.SubscriptionID,
		Status:          sqlc.InvoiceStatus(invoice.Status),
		Currency:        sqlc.Currency(invoice.Currency),
		AmountDue:       invoice.AmountDue,
		AmountPaid:      invoice.AmountPaid,
		AmountRemaining: invoice.AmountRemaining,
		DueDate:         pgtype.Timestamptz{Time: invoice.DueDate, Valid: true},
		PaidAt:          pgtype.Timestamptz{Time: invoice.PaidAt, Valid: !invoice.PaidAt.IsZero()},
	}); err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}

	// 將新創建的發票加入緩存
	cacheKey := fmt.Sprintf("invoice:%s", invoice.ID)
	if err := r.cache.Set(ctx, cacheKey, invoice); err != nil {
		r.logger.Warn("Failed to cache new invoice", zap.Error(err), zap.String("id", invoice.ID))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id string) (*models.Invoice, error) {
	cacheKey := fmt.Sprintf("invoice:%s", id)

	// 嘗試從緩存中獲取
	var cachedInvoice models.Invoice
	found, err := r.cache.Get(ctx, cacheKey, &cachedInvoice)
	if err != nil {
		r.logger.Warn("Failed to get invoice from cache", zap.Error(err), zap.String("id", id))
	} else if found {
		return &cachedInvoice, nil
	}

	// 如果緩存中沒有，從數據庫獲取
	sqlcInvoice, err := sqlc.New(r.conn).WithTx(tx).GetInvoice(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	invoiceObj, release, err := r.getFromPool(ctx, reflect.TypeOf(&models.Invoice{}))
	if err != nil {
		return nil, err
	}
	defer release()

	invoice := invoiceObj.(*models.Invoice)
	*invoice = *models.NewInvoice().ConvertFromSQLCInvoice(sqlcInvoice)

	// 更新緩存
	if err = r.cache.Set(ctx, cacheKey, invoice); err != nil {
		r.logger.Warn("Failed to cache invoice", zap.Error(err), zap.String("id", id))
	}

	return invoice, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, invoice *models.Invoice) error {
	err := sqlc.New(r.conn).WithTx(tx).UpdateInvoice(ctx, sqlc.UpdateInvoiceParams{
		ID:              invoice.ID,
		Status:          sqlc.InvoiceStatus(invoice.Status),
		AmountPaid:      invoice.AmountPaid,
		AmountRemaining: invoice.AmountRemaining,
		PaidAt:          pgtype.Timestamptz{Time: invoice.PaidAt, Valid: !invoice.PaidAt.IsZero()},
	})
	if err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}

	// 更新緩存
	cacheKey := fmt.Sprintf("invoice:%s", invoice.ID)
	if err = r.cache.Set(ctx, cacheKey, invoice); err != nil {
		r.logger.Warn("Failed to update invoice in cache", zap.Error(err), zap.String("id", invoice.ID))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, customerID string, limit, offset uint64) ([]*models.Invoice, error) {
	sqlcInvoices, err := sqlc.New(r.conn).WithTx(tx).ListInvoicesByCustomerID(ctx, sqlc.ListInvoicesByCustomerIDParams{
		CustomerID: customerID,
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	invoices := make([]*models.Invoice, 0, len(sqlcInvoices))
	for _, sqlcInvoice := range sqlcInvoices {
		invoiceObj, release, err := r.getFromPool(ctx, reflect.TypeOf(&models.Invoice{}))
		if err != nil {
			return nil, err
		}

		invoice := invoiceObj.(*models.Invoice)
		*invoice = *models.NewInvoice().ConvertFromSQLCInvoice(sqlcInvoice)
		invoices = append(invoices, invoice)

		// 更新每個發票的緩存
		cacheKey := fmt.Sprintf("invoice:%s", invoice.ID)
		if err = r.cache.Set(ctx, cacheKey, invoice); err != nil {
			r.logger.Warn("Failed to cache invoice during list", zap.Error(err), zap.String("id", invoice.ID))
		}

		release()
	}

	return invoices, nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	err := sqlc.New(r.conn).WithTx(tx).DeleteInvoice(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete invoice: %w", err)
	}

	// 從緩存中刪除
	cacheKey := fmt.Sprintf("invoice:%s", id)
	if err = r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete invoice from cache", zap.Error(err), zap.String("id", id))
	}

	return nil
}

func (r *repository) CreateInvoiceItem(ctx context.Context, tx pgx.Tx, item *models.InvoiceItem) error {
	err := sqlc.New(r.conn).WithTx(tx).CreateInvoiceItem(ctx, sqlc.CreateInvoiceItemParams{
		InvoiceID:   item.InvoiceID,
		Amount:      item.Amount,
		Description: &item.Description,
	})
	if err != nil {
		return fmt.Errorf("failed to create invoice item: %w", err)
	}

	// 將新創建的發票項目加入緩存
	cacheKey := fmt.Sprintf("invoice_item:%s", item.ID)
	if err = r.cache.Set(ctx, cacheKey, item); err != nil {
		r.logger.Warn("Failed to cache new invoice item", zap.Error(err), zap.String("id", item.ID))
	}

	return nil
}

func (r *repository) GetInvoiceItemByID(ctx context.Context, tx pgx.Tx, id string) (*models.InvoiceItem, error) {
	cacheKey := fmt.Sprintf("invoice_item:%s", id)
	// 嘗試從緩存中獲取
	var cachedItem models.InvoiceItem
	found, err := r.cache.Get(ctx, cacheKey, &cachedItem)
	if err != nil {
		r.logger.Warn("Failed to get invoice item from cache", zap.Error(err), zap.String("id", id))
	} else if found {
		return &cachedItem, nil
	}

	// 如果緩存中沒有，從數據庫獲取
	sqlcItem, err := sqlc.New(r.conn).WithTx(tx).GetInvoiceItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice item: %w", err)
	}

	itemObj, release, err := r.getFromPool(ctx, reflect.TypeOf(&models.InvoiceItem{}))
	if err != nil {
		return nil, err
	}
	defer release()

	item := itemObj.(*models.InvoiceItem)
	*item = *models.NewInvoiceItem().ConvertFromSQLCInvoiceItem(sqlcItem)

	// 更新緩存
	if err = r.cache.Set(ctx, cacheKey, item); err != nil {
		r.logger.Warn("Failed to cache invoice item", zap.Error(err), zap.String("id", id))
	}

	return item, nil
}

func (r *repository) UpdateInvoiceItem(ctx context.Context, tx pgx.Tx, item *models.InvoiceItem) error {
	return sqlc.New(r.conn).WithTx(tx).UpdateInvoiceItem(ctx, sqlc.UpdateInvoiceItemParams{
		ID:          item.ID,
		Amount:      item.Amount,
		Description: &item.Description,
	})
}

func (r *repository) DeleteInvoiceItem(ctx context.Context, tx pgx.Tx, id string) error {

	if err := sqlc.New(r.conn).WithTx(tx).DeleteInvoiceItem(ctx, id); err != nil {
		return fmt.Errorf("failed to delete invoice item: %w", err)
	}

	cacheKey := fmt.Sprintf("invoice_item:%s", id)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete invoice from cache", zap.Error(err), zap.String("id", id))
	}

	return nil
}

func (r *repository) ListInvoiceItems(ctx context.Context, tx pgx.Tx, invoiceID string) ([]*models.InvoiceItem, error) {
	sqlcItems, err := sqlc.New(r.conn).WithTx(tx).ListInvoiceItems(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoice items: %w", err)
	}

	items := make([]*models.InvoiceItem, 0, len(sqlcItems))
	for _, sqlcItem := range sqlcItems {
		item := models.NewInvoiceItem().ConvertFromSQLCInvoiceItem(sqlcItem)
		items = append(items, item)

		cacheKey := fmt.Sprintf("invoice_item:%s", item.ID)
		if err = r.cache.Set(ctx, cacheKey, item); err != nil {
			r.logger.Warn("Failed to cache invoice item during list", zap.Error(err), zap.String("id", item.ID))
		}
	}

	return items, nil
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, invoice *models.PartialInvoice) error {

	query := `
    INSERT INTO invoices (id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, created_at, updated_at)
    VALUES (@id, @customer_id, @subscription_id, @status, @currency, @amount_due, @amount_paid, @amount_remaining, @due_date, @paid_at, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        customer_id = COALESCE(@customer_id, invoices.customer_id),
        subscription_id = COALESCE(@subscription_id, invoices.subscription_id),
        status = COALESCE(@status, invoices.status),
        currency = COALESCE(@currency, invoices.currency),
        amount_due = COALESCE(@amount_due, invoices.amount_due),
        amount_paid = COALESCE(@amount_paid, invoices.amount_paid),
        amount_remaining = COALESCE(@amount_remaining, invoices.amount_remaining),
        due_date = COALESCE(@due_date, invoices.due_date),
        paid_at = COALESCE(@paid_at, invoices.paid_at),
        updated_at = @updated_at
    WHERE invoices.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":               invoice.ID,
		"customer_id":      invoice.CustomerID,
		"subscription_id":  invoice.SubscriptionID,
		"status":           invoice.Status,
		"currency":         invoice.Currency,
		"amount_due":       invoice.AmountDue,
		"amount_paid":      invoice.AmountPaid,
		"amount_remaining": invoice.AmountRemaining,
		"due_date":         invoice.DueDate,
		"paid_at":          invoice.PaidAt,
		"created_at":       invoice.CreatedAt,
		"updated_at":       now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert invoice: %w", err)
	}

	return nil
}
