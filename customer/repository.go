package customer

import (
	"context"
	"github.com/jackc/pgx/v5"
	"goflare.io/payment/driver"

	"go.uber.org/zap"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
)

var _ Repository = (*repository)(nil)

type Repository interface {
	Create(ctx context.Context, tx pgx.Tx, customer *models.Customer) error
	GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Customer, error)
	Update(ctx context.Context, tx pgx.Tx, customer *models.Customer) error
	Delete(ctx context.Context, tx pgx.Tx, id uint64) error
	List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.Customer, error)
	UpdateBalance(ctx context.Context, tx pgx.Tx, id, amount uint64) error
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

func (r *repository) Create(ctx context.Context, tx pgx.Tx, customer *models.Customer) error {
	return sqlc.New(r.conn).WithTx(tx).CreateCustomer(ctx, sqlc.CreateCustomerParams{
		UserID:   customer.UserID,
		Balance:  customer.Balance,
		StripeID: customer.StripeID,
	})
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Customer, error) {

	sqlcCustomer, err := sqlc.New(r.conn).WithTx(tx).GetCustomer(ctx, id)
	if err != nil {
		r.logger.Error("error getting customer", zap.Error(err))
		return nil, err
	}

	customer := models.NewCustomer().ConvertFromSQLCCustomer(sqlcCustomer)

	return customer, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, customer *models.Customer) error {
	return sqlc.New(r.conn).WithTx(tx).UpdateCustomer(ctx, sqlc.UpdateCustomerParams{
		ID:       customer.ID,
		Balance:  customer.Balance,
		StripeID: customer.StripeID,
	})
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id uint64) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteCustomer(ctx, id)
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.Customer, error) {

	sqlcCustomers, err := sqlc.New(r.conn).WithTx(tx).ListCustomers(ctx, sqlc.ListCustomersParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		r.logger.Error("error listing customers", zap.Error(err))
		return nil, err
	}

	customers := make([]*models.Customer, 0, len(sqlcCustomers))
	for _, sqlcCustomer := range sqlcCustomers {
		customer := models.NewCustomer().ConvertFromSQLCCustomer(sqlcCustomer)
		customers = append(customers, customer)
	}

	return customers, nil
}

func (r *repository) UpdateBalance(ctx context.Context, tx pgx.Tx, id, amount uint64) error {
	return sqlc.New(r.conn).WithTx(tx).UpdateCustomerBalance(ctx, sqlc.UpdateCustomerBalanceParams{
		ID:      id,
		Balance: int64(amount),
	})
}
