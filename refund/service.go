package refund

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v79"
	"go.uber.org/zap"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Service interface {
	Create(ctx context.Context, refund *models.Refund) error
	GetByID(ctx context.Context, id string) (*models.Refund, error)
	UpdateStatus(ctx context.Context, id string, status stripe.RefundStatus, reason stripe.RefundReason) error
	List(ctx context.Context, chargeID string, limit, offset uint64) ([]*models.Refund, error)
	ListByChargeID(ctx context.Context, chargeID string) ([]*models.Refund, error)
	Upsert(ctx context.Context, refund *models.PartialRefund) error
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

func (s *service) Create(ctx context.Context, refund *models.Refund) error {
	if err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Create(ctx, tx, refund)
	}); err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id string) (*models.Refund, error) {
	var refund *models.Refund
	if err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		refund, err = s.repo.GetByID(ctx, tx, id)
		return err
	}); err != nil {
		return nil, fmt.Errorf("failed to get refund: %w", err)
	}
	return refund, nil
}

func (s *service) UpdateStatus(ctx context.Context, id string, status stripe.RefundStatus, reason stripe.RefundReason) error {
	var refund *models.Refund
	if err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		refund, err = s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return err
		}
		refund.Status = status
		refund.Reason = reason
		return s.repo.Update(ctx, tx, refund)
	}); err != nil {
		return fmt.Errorf("failed to update refund status: %w", err)
	}
	return nil
}

func (s *service) List(ctx context.Context, chargeID string, limit, offset uint64) ([]*models.Refund, error) {
	var refunds []*models.Refund
	if err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		refunds, err = s.repo.List(ctx, tx, chargeID, limit, offset)
		return err
	}); err != nil {
		return nil, fmt.Errorf("failed to list refunds: %w", err)
	}
	return refunds, nil
}

func (s *service) ListByChargeID(ctx context.Context, chargeID string) ([]*models.Refund, error) {

	var refunds []*models.Refund
	if err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		refunds, err = s.repo.ListByChargeID(ctx, chargeID)
		return err
	}); err != nil {
		return nil, fmt.Errorf("failed to list refunds: %w", err)
	}
	return refunds, nil
}

func (s *service) Upsert(ctx context.Context, refund *models.PartialRefund) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Upsert(ctx, tx, refund)
	})
}
