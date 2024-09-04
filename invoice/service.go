package invoice

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/models/enum"
	"time"
)

type Service interface {
	Create(ctx context.Context, invoice *models.Invoice) error
	GetByID(ctx context.Context, id uint64) (*models.Invoice, error)
	Update(ctx context.Context, invoice *models.Invoice) error
	List(ctx context.Context, customerID uint64, limit, offset uint64) ([]*models.Invoice, error)
	Delete(ctx context.Context, id uint64) error
	PayInvoice(ctx context.Context, id, amount uint64) error
	CreateInvoiceItem(ctx context.Context, item *models.InvoiceItem) error
	UpdateInvoiceItem(ctx context.Context, item *models.InvoiceItem) error
	DeleteInvoiceItem(ctx context.Context, id uint64) error
	ListInvoiceItems(ctx context.Context, invoiceID uint64) ([]*models.InvoiceItem, error)
	ListByStripeID(ctx context.Context, stripeID string) ([]*models.Invoice, error)
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

func (s *service) Create(ctx context.Context, invoice *models.Invoice) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.repo.Create(ctx, tx, invoice); err != nil {
			return fmt.Errorf("failed to create invoice: %w", err)
		}

		for _, item := range invoice.InvoiceItems {
			item.InvoiceID = invoice.ID
			if err := s.repo.CreateInvoiceItem(ctx, tx, item); err != nil {
				return fmt.Errorf("failed to create invoice item: %w", err)
			}
		}

		return nil
	})
}

func (s *service) GetByID(ctx context.Context, id uint64) (*models.Invoice, error) {
	var invoice *models.Invoice
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		invoice, err = s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get invoice: %w", err)
		}

		invoice.InvoiceItems, err = s.repo.ListInvoiceItems(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to list invoice items: %w", err)
		}

		return nil
	})
	return invoice, err
}

func (s *service) Update(ctx context.Context, invoice *models.Invoice) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		existingInvoice, err := s.repo.GetByID(ctx, tx, invoice.ID)
		if err != nil {
			return fmt.Errorf("failed to get existing invoice: %w", err)
		}

		// Update only allowed fields
		existingInvoice.Status = invoice.Status
		existingInvoice.AmountPaid = invoice.AmountPaid
		existingInvoice.AmountRemaining = invoice.AmountRemaining
		existingInvoice.PaidAt = invoice.PaidAt
		existingInvoice.StripeID = invoice.StripeID

		if err := s.repo.Update(ctx, tx, existingInvoice); err != nil {
			return fmt.Errorf("failed to update invoice: %w", err)
		}

		return nil
	})
}

func (s *service) List(ctx context.Context, customerID uint64, limit, offset uint64) ([]*models.Invoice, error) {
	var invoices []*models.Invoice
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		invoices, err = s.repo.List(ctx, tx, customerID, limit, offset)
		if err != nil {
			return fmt.Errorf("failed to list invoices: %w", err)
		}
		return nil
	})
	return invoices, err
}

func (s *service) Delete(ctx context.Context, id uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.repo.Delete(ctx, tx, id); err != nil {
			return fmt.Errorf("failed to delete invoice: %w", err)
		}
		return nil
	})
}

func (s *service) PayInvoice(ctx context.Context, id, amount uint64) error {
	return s.transactionManager.ExecuteSerializableTransaction(ctx, func(tx pgx.Tx) error {
		invoice, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get invoice: %w", err)
		}

		if invoice.Status == enum.InvoiceStatusPaid {
			return fmt.Errorf("invoice is already paid")
		}

		if amount > invoice.AmountRemaining {
			return fmt.Errorf("payment amount exceeds remaining amount")
		}

		invoice.AmountPaid += amount
		invoice.AmountRemaining -= amount
		invoice.PaidAt = time.Now()

		if invoice.AmountRemaining == 0 {
			invoice.Status = enum.InvoiceStatusPaid
		} else {
			invoice.Status = enum.InvoiceStatusPartiallyPaid
		}

		if err := s.repo.Update(ctx, tx, invoice); err != nil {
			return fmt.Errorf("failed to update invoice: %w", err)
		}

		return nil
	})
}

func (s *service) CreateInvoiceItem(ctx context.Context, item *models.InvoiceItem) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 檢查相關的發票是否存在
		invoice, err := s.repo.GetByID(ctx, tx, item.InvoiceID)
		if err != nil {
			return fmt.Errorf("failed to get invoice for item: %w", err)
		}

		if invoice.Status == enum.InvoiceStatusPaid {
			return fmt.Errorf("cannot add item to a paid invoice")
		}

		if err := s.repo.CreateInvoiceItem(ctx, tx, item); err != nil {
			return fmt.Errorf("failed to create invoice item: %w", err)
		}

		// 更新發票總額
		invoice.AmountDue += item.Amount
		invoice.AmountRemaining += item.Amount
		if err := s.repo.Update(ctx, tx, invoice); err != nil {
			return fmt.Errorf("failed to update invoice after adding item: %w", err)
		}

		return nil
	})
}

func (s *service) UpdateInvoiceItem(ctx context.Context, item *models.InvoiceItem) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 獲取原始的發票項目
		originalItem, err := s.repo.GetInvoiceItemByID(ctx, tx, item.ID)
		if err != nil {
			return fmt.Errorf("failed to get original invoice item: %w", err)
		}

		// 檢查相關的發票是否可以修改
		invoice, err := s.repo.GetByID(ctx, tx, originalItem.InvoiceID)
		if err != nil {
			return fmt.Errorf("failed to get invoice for item: %w", err)
		}

		if invoice.Status == enum.InvoiceStatusPaid {
			return fmt.Errorf("cannot update item of a paid invoice")
		}

		// 計算金額差異
		amountDifference := item.Amount - originalItem.Amount

		// 更新發票項目
		if err := s.repo.UpdateInvoiceItem(ctx, tx, item); err != nil {
			return fmt.Errorf("failed to update invoice item: %w", err)
		}

		// 更新發票總額
		invoice.AmountDue += amountDifference
		invoice.AmountRemaining += amountDifference
		if err := s.repo.Update(ctx, tx, invoice); err != nil {
			return fmt.Errorf("failed to update invoice after updating item: %w", err)
		}

		return nil
	})
}

func (s *service) DeleteInvoiceItem(ctx context.Context, id uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 獲取要刪除的發票項目
		item, err := s.repo.GetInvoiceItemByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get invoice item: %w", err)
		}

		// 檢查相關的發票是否可以修改
		invoice, err := s.repo.GetByID(ctx, tx, item.InvoiceID)
		if err != nil {
			return fmt.Errorf("failed to get invoice for item: %w", err)
		}

		if invoice.Status == enum.InvoiceStatusPaid {
			return fmt.Errorf("cannot delete item from a paid invoice")
		}

		// 刪除發票項目
		if err := s.repo.DeleteInvoiceItem(ctx, tx, id); err != nil {
			return fmt.Errorf("failed to delete invoice item: %w", err)
		}

		// 更新發票總額
		invoice.AmountDue -= item.Amount
		invoice.AmountRemaining -= item.Amount
		if err = s.repo.Update(ctx, tx, invoice); err != nil {
			return fmt.Errorf("failed to update invoice after deleting item: %w", err)
		}

		return nil
	})
}

func (s *service) ListInvoiceItems(ctx context.Context, invoiceID uint64) ([]*models.InvoiceItem, error) {
	var items []*models.InvoiceItem
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		items, err = s.repo.ListInvoiceItems(ctx, tx, invoiceID)
		if err != nil {
			return fmt.Errorf("failed to list invoice items: %w", err)
		}
		return nil
	})
	return items, err
}

func (s *service) ListByStripeID(ctx context.Context, stripeID string) ([]*models.Invoice, error) {
	var invoices []*models.Invoice
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		invoices, err = s.repo.ListByStripeID(ctx, tx, stripeID)
		if err != nil {
			return fmt.Errorf("failed to list invoices: %w", err)
		}
		return nil
	})
	return invoices, err
}
