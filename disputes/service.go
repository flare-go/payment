package disputes

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Service interface {
	Create(ctx context.Context, dispute *models.Dispute) error
	GetByID(ctx context.Context, id string) (*models.Dispute, error)
	Update(ctx context.Context, dispute *models.Dispute) error
	Close(ctx context.Context, stripeID string) error
	Upsert(ctx context.Context, dispute *models.PartialDispute) error
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

func (s *service) Create(ctx context.Context, dispute *models.Dispute) error {
	if err := s.repo.Create(ctx, dispute); err != nil {
		s.logger.Error("Failed to create dispute", zap.Error(err))
		return fmt.Errorf("failed to create dispute: %w", err)
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id string) (*models.Dispute, error) {
	dispute, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get dispute by ID", zap.Error(err), zap.String("id", id))
		return nil, fmt.Errorf("failed to get dispute: %w", err)
	}
	return dispute, nil
}

func (s *service) Update(ctx context.Context, dispute *models.Dispute) error {
	if err := s.repo.Update(ctx, dispute); err != nil {
		s.logger.Error("Failed to update dispute", zap.Error(err), zap.String("id", dispute.ID))
		return fmt.Errorf("failed to update dispute: %w", err)
	}
	return nil
}

func (s *service) Close(ctx context.Context, id string) error {
	return s.repo.Close(ctx, id)
}

func (s *service) Upsert(ctx context.Context, dispute *models.PartialDispute) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Upsert(ctx, tx, dispute)
	})
}
