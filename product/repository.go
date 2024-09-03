package product

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
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
	conn   driver.PostgresPool
	logger *zap.Logger
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger) Repository {
	return &repository{
		conn:   conn,
		logger: logger,
	}
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, product *models.Product) error {

	metadata, err := json.Marshal(product.Metadata)
	if err != nil {
		return err
	}

	return sqlc.New(r.conn).WithTx(tx).CreateProduct(ctx, sqlc.CreateProductParams{
		Name:        product.Name,
		Description: &product.Description,
		Active:      product.Active,
		Metadata:    metadata,
		StripeID:    product.StripeID,
	})
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Product, error) {
	sqlcProduct, err := sqlc.New(r.conn).WithTx(tx).GetProduct(ctx, id)
	if err != nil {
		r.logger.Error("error getting product", zap.Error(err))
		return nil, err
	}

	return models.NewProduct().ConvertFromSQLCProduct(sqlcProduct), nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, product *models.Product) error {

	metadata, err := json.Marshal(product.Metadata)
	if err != nil {
		return err
	}

	return sqlc.New(r.conn).WithTx(tx).UpdateProduct(ctx, sqlc.UpdateProductParams{
		ID:          product.ID,
		Name:        product.Name,
		Description: &product.Description,
		Active:      product.Active,
		Metadata:    metadata,
		StripeID:    product.StripeID,
	})
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id uint64) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteProduct(ctx, id)
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, limit, offset uint64, activeOnly bool) ([]*models.Product, error) {
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
		product := models.NewProduct().ConvertFromSQLCProduct(sqlcProduct)
		products = append(products, product)
	}

	return products, nil
}
