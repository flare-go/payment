package payment_intent

import (
	"context"
	"fmt"
	"github.com/stripe/stripe-go/v79"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Service interface {
	Create(ctx context.Context, paymentIntent *models.PaymentIntent) error
	GetByID(ctx context.Context, id string) (*models.PaymentIntent, error)
	Update(ctx context.Context, paymentIntent *models.PaymentIntent) error
	List(ctx context.Context, limit, offset uint64) ([]*models.PaymentIntent, error)
	ListByCustomer(ctx context.Context, customerID string, limit, offset uint64) ([]*models.PaymentIntent, error)
	Confirm(ctx context.Context, id string, paymentMethodID string) error
	Failed(ctx context.Context, id string, paymentMethodID string) error
	Cancel(ctx context.Context, id string) error
	Upsert(ctx context.Context, paymentIntent *models.PartialPaymentIntent) error
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

func (s *service) Create(ctx context.Context, paymentIntent *models.PaymentIntent) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Create(ctx, tx, paymentIntent)
	})
}

func (s *service) GetByID(ctx context.Context, id string) (*models.PaymentIntent, error) {
	var paymentIntent *models.PaymentIntent
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		paymentIntent, err = s.repo.GetByID(ctx, tx, id)
		return err
	})
	return paymentIntent, err
}

func (s *service) Update(ctx context.Context, paymentIntent *models.PaymentIntent) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		existingPaymentIntent, err := s.repo.GetByID(ctx, tx, paymentIntent.ID)
		if err != nil {
			return fmt.Errorf("failed to get existing payment intent: %w", err)
		}

		// Update only allowed fields
		existingPaymentIntent.ID = paymentIntent.ID
		existingPaymentIntent.Status = paymentIntent.Status
		existingPaymentIntent.PaymentMethodID = paymentIntent.PaymentMethodID
		existingPaymentIntent.SetupFutureUsage = paymentIntent.SetupFutureUsage
		existingPaymentIntent.ClientSecret = paymentIntent.ClientSecret

		return s.repo.Update(ctx, tx, existingPaymentIntent)
	})
}

func (s *service) List(ctx context.Context, limit, offset uint64) ([]*models.PaymentIntent, error) {
	var paymentIntents []*models.PaymentIntent
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		paymentIntents, err = s.repo.List(ctx, tx, limit, offset)
		return err
	})
	return paymentIntents, err
}

func (s *service) ListByCustomer(ctx context.Context, customerID string, limit, offset uint64) ([]*models.PaymentIntent, error) {
	var paymentIntents []*models.PaymentIntent
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		paymentIntents, err = s.repo.ListByCustomer(ctx, tx, customerID, limit, offset)
		return err
	})
	return paymentIntents, err
}

func (s *service) Confirm(ctx context.Context, id, paymentMethodID string) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		paymentIntent, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get payment intent: %w", err)
		}

		if paymentIntent.Status != stripe.PaymentIntentStatusRequiresPaymentMethod &&
			paymentIntent.Status != stripe.PaymentIntentStatusRequiresConfirmation {
			return fmt.Errorf("payment intent cannot be confirmed in its current status: %s", paymentIntent.Status)
		}

		paymentIntent.Status = stripe.PaymentIntentStatusSucceeded
		paymentIntent.PaymentMethodID = paymentMethodID

		return s.repo.Update(ctx, tx, paymentIntent)
	})
}

func (s *service) Failed(ctx context.Context, id, paymentMethodID string) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		paymentIntent, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get payment intent: %w", err)
		}

		if paymentIntent.Status != stripe.PaymentIntentStatusRequiresPaymentMethod &&
			paymentIntent.Status != stripe.PaymentIntentStatusRequiresConfirmation {
			return fmt.Errorf("payment intent cannot be confirmed in its current status: %s", paymentIntent.Status)
		}

		paymentIntent.Status = stripe.PaymentIntentStatusCanceled
		paymentIntent.PaymentMethodID = paymentMethodID

		return s.repo.Update(ctx, tx, paymentIntent)
	})
}

func (s *service) Cancel(ctx context.Context, id string) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		paymentIntent, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get payment intent: %w", err)
		}

		if paymentIntent.Status == stripe.PaymentIntentStatusSucceeded ||
			paymentIntent.Status == stripe.PaymentIntentStatusCanceled {
			return fmt.Errorf("payment intent cannot be canceled in its current status: %s", paymentIntent.Status)
		}

		paymentIntent.Status = stripe.PaymentIntentStatusCanceled

		return s.repo.Update(ctx, tx, paymentIntent)
	})
}

func (s *service) Upsert(ctx context.Context, paymentIntent *models.PartialPaymentIntent) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Upsert(ctx, tx, paymentIntent)
	})
}
