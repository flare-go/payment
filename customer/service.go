package customer

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Service interface {
	Create(ctx context.Context, customer *models.Customer) error
	GetByID(ctx context.Context, id uint64) (*models.Customer, error)
	Update(ctx context.Context, customer *models.Customer) error
	Delete(ctx context.Context, id uint64) error
	List(ctx context.Context, limit, offset uint64) ([]*models.Customer, error)
	UpdateBalance(ctx context.Context, id, amount uint64) error
}

type service struct {
	repo               Repository
	transactionManager *driver.TransactionManager
	logger             *zap.Logger
}

func NewService(repo Repository, tm *driver.TransactionManager, logger *zap.Logger) Service {
	return &service{
		repo:               repo,
		transactionManager: tm,
		logger:             logger,
	}
}

func (s *service) Create(ctx context.Context, customer *models.Customer) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Create(ctx, tx, customer)
	})
}

func (s *service) GetByID(ctx context.Context, id uint64) (*models.Customer, error) {
	var customer *models.Customer
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		customer, err = s.repo.GetByID(ctx, tx, id)
		return err
	})
	return customer, err
}

func (s *service) Update(ctx context.Context, customer *models.Customer) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Update(ctx, tx, customer)
	})
}

func (s *service) Delete(ctx context.Context, id uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Delete(ctx, tx, id)
	})
}

func (s *service) List(ctx context.Context, limit, offset uint64) ([]*models.Customer, error) {
	var customers []*models.Customer
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		customers, err = s.repo.List(ctx, tx, limit, offset)
		return err
	})
	return customers, err
}

func (s *service) UpdateBalance(ctx context.Context, id, amount uint64) error {
	return s.transactionManager.ExecuteSerializableTransaction(ctx, func(tx pgx.Tx) error {
		customer, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get customer: %w", err)
		}

		newBalance := customer.Balance + int64(amount)
		if newBalance < 0 {
			return fmt.Errorf("insufficient balance")
		}

		return s.repo.UpdateBalance(ctx, tx, id, uint64(newBalance))
	})
}
