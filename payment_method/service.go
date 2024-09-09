package payment_method

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Service interface {
	Create(ctx context.Context, paymentMethod *models.PaymentMethod) error
	GetByID(ctx context.Context, id string) (*models.PaymentMethod, error)
	Update(ctx context.Context, paymentMethod *models.PaymentMethod) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, customerID string, limit, offset uint64) ([]*models.PaymentMethod, error)
	SetDefault(ctx context.Context, customerID, paymentMethodID string) error
	Upsert(ctx context.Context, paymentMethod *models.PartialPaymentMethod) error
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

func (s *service) Create(ctx context.Context, paymentMethod *models.PaymentMethod) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		existingMethods, err := s.repo.List(ctx, tx, paymentMethod.CustomerID, 1, 0)
		if err != nil {
			return fmt.Errorf("failed to check existing payment methods: %w", err)
		}
		defer existingMethods.release()

		if len(existingMethods.PaymentMethods) == 0 {
			paymentMethod.IsDefault = true
		}

		return s.repo.Create(ctx, tx, paymentMethod)
	})
}

func (s *service) GetByID(ctx context.Context, id string) (*models.PaymentMethod, error) {
	var result *models.PaymentMethod
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		autoReleasePaymentMethod, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return err
		}
		defer autoReleasePaymentMethod.release()
		result = autoReleasePaymentMethod.PaymentMethod
		return nil
	})
	return result, err
}

func (s *service) Update(ctx context.Context, paymentMethod *models.PaymentMethod) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		existing, err := s.repo.GetByID(ctx, tx, paymentMethod.ID)
		if err != nil {
			return fmt.Errorf("failed to get existing payment method: %w", err)
		}
		defer existing.release()

		existing.ID = paymentMethod.ID
		existing.PaymentMethod.CardLast4 = paymentMethod.CardLast4
		existing.PaymentMethod.CardBrand = paymentMethod.CardBrand
		existing.PaymentMethod.CardExpMonth = paymentMethod.CardExpMonth
		existing.PaymentMethod.CardExpYear = paymentMethod.CardExpYear
		existing.PaymentMethod.BankAccountLast4 = paymentMethod.BankAccountLast4
		existing.PaymentMethod.BankAccountBankName = paymentMethod.BankAccountBankName

		return s.repo.Update(ctx, tx, existing.PaymentMethod)
	})
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		paymentMethod, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get payment method: %w", err)
		}
		defer paymentMethod.release()

		if paymentMethod.PaymentMethod.IsDefault {
			return errors.New("cannot delete default payment method")
		}

		return s.repo.Delete(ctx, tx, id)
	})
}

func (s *service) List(ctx context.Context, customerID string, limit, offset uint64) ([]*models.PaymentMethod, error) {
	var result []*models.PaymentMethod
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		autoReleasePaymentMethods, err := s.repo.List(ctx, tx, customerID, limit, offset)
		if err != nil {
			return err
		}
		defer autoReleasePaymentMethods.release()
		result = autoReleasePaymentMethods.PaymentMethods
		return nil
	})
	return result, err
}

func (s *service) SetDefault(ctx context.Context, customerID, paymentMethodID string) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		paymentMethods, err := s.repo.List(ctx, tx, customerID, 1000, 0)
		if err != nil {
			return fmt.Errorf("failed to list payment methods: %w", err)
		}
		defer paymentMethods.release()

		var targetMethod *models.PaymentMethod
		for _, pm := range paymentMethods.PaymentMethods {
			if pm.ID == paymentMethodID {
				targetMethod = pm
			}
			if pm.IsDefault {
				pm.IsDefault = false
				if err := s.repo.Update(ctx, tx, pm); err != nil {
					return fmt.Errorf("failed to unset default payment method: %w", err)
				}
			}
		}

		if targetMethod == nil {
			return errors.New("payment method not found")
		}

		targetMethod.IsDefault = true
		if err = s.repo.Update(ctx, tx, targetMethod); err != nil {
			return fmt.Errorf("failed to set default payment method: %w", err)
		}

		return nil
	})
}

func (s *service) Upsert(ctx context.Context, paymentMethod *models.PartialPaymentMethod) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Upsert(ctx, tx, paymentMethod)
	})
}
