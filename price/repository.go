package price

import (
	"context"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
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
	conn   driver.PostgresPool
	logger *zap.Logger
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger) Repository {
	return &repository{
		conn:   conn,
		logger: logger,
	}
}

func (r *repository) Create(ctx context.Context, tx pgx.Tx, price *models.Price) error {
	return sqlc.New(r.conn).WithTx(tx).CreatePrice(ctx, sqlc.CreatePriceParams{
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
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Price, error) {
	sqlcPrice, err := sqlc.New(r.conn).WithTx(tx).GetPrice(ctx, id)
	if err != nil {
		r.logger.Error("error getting price", zap.Error(err))
		return nil, err
	}

	return models.NewPrice().ConvertFromSQLCPrice(sqlcPrice), nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, price *models.Price) error {
	return sqlc.New(r.conn).WithTx(tx).UpdatePrice(ctx, sqlc.UpdatePriceParams{
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
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id uint64) error {
	return sqlc.New(r.conn).WithTx(tx).DeletePrice(ctx, id)
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, productID uint64, limit, offset uint64, activeOnly bool) ([]*models.Price, error) {
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
		price := models.NewPrice().ConvertFromSQLCPrice(sqlcPrice)
		prices = append(prices, price)
	}

	return prices, nil
}
