package product

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Service interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id uint64) (*models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, id uint64) error
	List(ctx context.Context, limit, offset uint64) ([]*models.Product, error)
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

func (s *service) Create(ctx context.Context, product *models.Product) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Create(ctx, tx, product)
	})
}

func (s *service) GetByID(ctx context.Context, id uint64) (*models.Product, error) {
	var product *models.Product
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		product, err = s.repo.GetByID(ctx, tx, id)
		return err
	})
	return product, err
}

func (s *service) Update(ctx context.Context, product *models.Product) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		existingProduct, err := s.repo.GetByID(ctx, tx, product.ID)
		if err != nil {
			return fmt.Errorf("failed to get existing product: %w", err)
		}

		// 更新非空字段
		if product.Name != "" {
			existingProduct.Name = product.Name
		}
		if product.Description != "" {
			existingProduct.Description = product.Description
		}
		existingProduct.Active = product.Active
		if product.Metadata != nil {
			for k, v := range product.Metadata {
				if existingProduct.Metadata == nil {
					existingProduct.Metadata = make(map[string]string)
				}
				existingProduct.Metadata[k] = v
			}
		}
		if product.StripeID != "" {
			existingProduct.StripeID = product.StripeID
		}

		return s.repo.Update(ctx, tx, existingProduct)
	})
}

func (s *service) Delete(ctx context.Context, id uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Delete(ctx, tx, id)
	})
}

func (s *service) List(ctx context.Context, limit, offset uint64) ([]*models.Product, error) {
	var products []*models.Product
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		products, err = s.repo.List(ctx, tx, limit, offset)
		return err
	})
	return products, err
}
