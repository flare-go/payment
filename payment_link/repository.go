package payment_link

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
)

type Repository interface {
	Upsert(ctx context.Context, tx pgx.Tx, paymentLink *models.PartialPaymentLink) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, paymentLink *models.PartialPaymentLink) error {
	const query = `
    INSERT INTO payment_links (id, active, url, amount, currency, created_at, updated_at)
    VALUES (@id, @active, @url, @amount, @currency, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        active = COALESCE(@active, payment_links.active),
        url = COALESCE(@url, payment_links.url),
        amount = COALESCE(@amount, payment_links.amount),
        currency = COALESCE(@currency, payment_links.currency),
        updated_at = @updated_at
    WHERE payment_links.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":         paymentLink.ID,
		"active":     paymentLink.Active,
		"url":        paymentLink.URL,
		"amount":     paymentLink.Amount,
		"currency":   paymentLink.Currency,
		"created_at": paymentLink.CreatedAt,
		"updated_at": now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert payment link: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeletePaymentLink(ctx, id)
}
