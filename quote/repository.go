package quote

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
	Upsert(ctx context.Context, tx pgx.Tx, quote *models.PartialQuote) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, quote *models.PartialQuote) error {
	const query = `
    INSERT INTO quotes (id, customer_id, status, amount_total, currency, valid_until, accepted_at, canceled_at, created_at, updated_at)
    VALUES (@id, @customer_id, @status, @amount_total, @currency, @valid_until, @accepted_at, @canceled_at, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        customer_id = COALESCE(@customer_id, quotes.customer_id),
        status = COALESCE(@status, quotes.status),
        amount_total = COALESCE(@amount_total, quotes.amount_total),
        currency = COALESCE(@currency, quotes.currency),
        valid_until = COALESCE(@valid_until, quotes.valid_until),
        accepted_at = COALESCE(@accepted_at, quotes.accepted_at),
        canceled_at = COALESCE(@canceled_at, quotes.canceled_at),
        updated_at = @updated_at
    WHERE quotes.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":           quote.ID,
		"customer_id":  quote.CustomerID,
		"status":       quote.Status,
		"amount_total": quote.AmountTotal,
		"currency":     quote.Currency,
		"valid_until":  quote.ValidUntil,
		"accepted_at":  quote.AcceptedAt,
		"canceled_at":  quote.CanceledAt,
		"created_at":   quote.CreatedAt,
		"updated_at":   now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert quote: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteQuote(ctx, id)
}
