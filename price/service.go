package price

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Service interface {
	Create(ctx context.Context, price *models.Price) error
	GetByID(ctx context.Context, id uint64) (*models.Price, error)
	Update(ctx context.Context, price *models.Price) error
	Delete(ctx context.Context, id uint64) error
	List(ctx context.Context, productID uint64) ([]*models.Price, error)
	ListActive(ctx context.Context, productID uint64) ([]*models.Price, error)
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

func (s *service) Create(ctx context.Context, price *models.Price) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Create(ctx, tx, price)
	})
}

func (s *service) GetByID(ctx context.Context, id uint64) (*models.Price, error) {
	var price *models.Price
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		price, err = s.repo.GetByID(ctx, tx, id)
		return err
	})
	return price, err
}

func (s *service) Update(ctx context.Context, price *models.Price) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		existingPrice, err := s.repo.GetByID(ctx, tx, price.ID)
		if err != nil {
			return fmt.Errorf("failed to get existing price: %w", err)
		}

		// Update non-empty fields
		if price.ProductID != 0 {
			existingPrice.ProductID = price.ProductID
		}
		if price.Type != "" {
			existingPrice.Type = price.Type
		}
		if price.Currency != "" {
			existingPrice.Currency = price.Currency
		}
		if price.UnitAmount != 0 {
			existingPrice.UnitAmount = price.UnitAmount
		}
		if price.RecurringInterval != "" {
			existingPrice.RecurringInterval = price.RecurringInterval
		}
		if price.RecurringIntervalCount != 0 {
			existingPrice.RecurringIntervalCount = price.RecurringIntervalCount
		}
		if price.TrialPeriodDays != 0 {
			existingPrice.TrialPeriodDays = price.TrialPeriodDays
		}
		existingPrice.Active = price.Active
		if price.StripeID != "" {
			existingPrice.StripeID = price.StripeID
		}

		return s.repo.Update(ctx, tx, existingPrice)
	})
}

func (s *service) Delete(ctx context.Context, id uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Delete(ctx, tx, id)
	})
}

func (s *service) List(ctx context.Context, productID uint64) ([]*models.Price, error) {
	var prices []*models.Price
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		prices, err = s.repo.List(ctx, tx, productID)
		return err
	})
	return prices, err
}

func (s *service) ListActive(ctx context.Context, productID uint64) ([]*models.Price, error) {
	var prices []*models.Price
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		prices, err = s.repo.ListActive(ctx, tx, productID)
		return err
	})
	return prices, err
}
