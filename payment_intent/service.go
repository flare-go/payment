package payment_intent

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/models/enum"
)

type Service interface {
	Create(ctx context.Context, paymentIntent *models.PaymentIntent) error
	GetByID(ctx context.Context, id uint64) (*models.PaymentIntent, error)
	Update(ctx context.Context, paymentIntent *models.PaymentIntent) error
	List(ctx context.Context, customerID uint64, limit, offset uint64) ([]*models.PaymentIntent, error)
	Confirm(ctx context.Context, id uint64, paymentMethodID *uint64) error
	Cancel(ctx context.Context, id uint64) error
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

func (s *service) GetByID(ctx context.Context, id uint64) (*models.PaymentIntent, error) {
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
		existingPaymentIntent.Status = paymentIntent.Status
		existingPaymentIntent.PaymentMethodID = paymentIntent.PaymentMethodID
		existingPaymentIntent.SetupFutureUsage = paymentIntent.SetupFutureUsage
		existingPaymentIntent.StripeID = paymentIntent.StripeID
		existingPaymentIntent.ClientSecret = paymentIntent.ClientSecret

		return s.repo.Update(ctx, tx, existingPaymentIntent)
	})
}

func (s *service) List(ctx context.Context, customerID uint64, limit, offset uint64) ([]*models.PaymentIntent, error) {
	var paymentIntents []*models.PaymentIntent
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		paymentIntents, err = s.repo.List(ctx, tx, customerID, limit, offset)
		return err
	})
	return paymentIntents, err
}

func (s *service) Confirm(ctx context.Context, id uint64, paymentMethodID *uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		paymentIntent, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get payment intent: %w", err)
		}

		if paymentIntent.Status != enum.PaymentIntentStatusRequiresPaymentMethod &&
			paymentIntent.Status != enum.PaymentIntentStatusRequiresConfirmation {
			return fmt.Errorf("payment intent cannot be confirmed in its current status: %s", paymentIntent.Status)
		}

		paymentIntent.Status = enum.PaymentIntentStatusSucceeded
		paymentIntent.PaymentMethodID = paymentMethodID

		// Here you would typically integrate with your payment provider (e.g., Stripe)
		// to actually process the payment. This is just a placeholder.
		// stripeConfirmation, err := s.stripeClient.ConfirmPaymentIntent(paymentIntent.StripeID)
		// if err != nil {
		//     return fmt.Errorf("failed to confirm payment with Stripe: %w", err)
		// }
		// paymentIntent.StripeID = stripeConfirmation.ID

		return s.repo.Update(ctx, tx, paymentIntent)
	})
}

func (s *service) Cancel(ctx context.Context, id uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		paymentIntent, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get payment intent: %w", err)
		}

		if paymentIntent.Status == enum.PaymentIntentStatusSucceeded ||
			paymentIntent.Status == enum.PaymentIntentStatusCanceled {
			return fmt.Errorf("payment intent cannot be canceled in its current status: %s", paymentIntent.Status)
		}

		paymentIntent.Status = enum.PaymentIntentStatusCanceled

		// Here you would typically integrate with your payment provider (e.g., Stripe)
		// to cancel the payment intent on their side as well. This is just a placeholder.
		// err = s.stripeClient.CancelPaymentIntent(paymentIntent.StripeID)
		// if err != nil {
		//     return fmt.Errorf("failed to cancel payment with Stripe: %w", err)
		// }

		return s.repo.Update(ctx, tx, paymentIntent)
	})
}
