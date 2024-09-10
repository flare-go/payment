package payment_link

import (
	"context"

	"github.com/jackc/pgx/v5"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Service interface {
	Upsert(ctx context.Context, coupon *models.PartialPaymentLink) error
	Delete(ctx context.Context, id string) error
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

func (s *service) Upsert(ctx context.Context, pl *models.PartialPaymentLink) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Upsert(ctx, tx, pl)
	})
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Delete(ctx, tx, id)
	})
}
