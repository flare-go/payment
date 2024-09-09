package charge

import (
	"context"
	"github.com/jackc/pgx/v5"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Service interface {
	Upsert(ctx context.Context, charge *models.PartialCharge) error
}

type service struct {
	repo               Repository
	transactionManager *driver.TransactionManager
}

func NewService(repo Repository, tm *driver.TransactionManager) Service {
	return &service{
		repo:               repo,
		transactionManager: tm,
	}
}

func (s *service) Upsert(ctx context.Context, charge *models.PartialCharge) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Upsert(ctx, tx, charge)
	})
}
