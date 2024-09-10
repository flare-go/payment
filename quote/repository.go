package quote

import (
	"context"
	"fmt"
	"goflare.io/payment/sqlc"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
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
	query := `
    INSERT INTO quotes (id, customer_id, status, amount_total, currency, valid_until, accepted_at, canceled_at, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{quote.ID}
	var updateClauses []string
	argIndex := 2

	if quote.CustomerID != nil {
		args = append(args, *quote.CustomerID)
		updateClauses = append(updateClauses, fmt.Sprintf("customer_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if quote.Status != nil {
		args = append(args, *quote.Status)
		updateClauses = append(updateClauses, fmt.Sprintf("status = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if quote.AmountTotal != nil {
		args = append(args, *quote.AmountTotal)
		updateClauses = append(updateClauses, fmt.Sprintf("amount_total = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if quote.Currency != nil {
		args = append(args, *quote.Currency)
		updateClauses = append(updateClauses, fmt.Sprintf("currency = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if quote.ValidUntil != nil {
		args = append(args, *quote.ValidUntil)
		updateClauses = append(updateClauses, fmt.Sprintf("valid_until = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if quote.AcceptedAt != nil {
		args = append(args, *quote.AcceptedAt)
		updateClauses = append(updateClauses, fmt.Sprintf("accepted_at = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if quote.CanceledAt != nil {
		args = append(args, *quote.CanceledAt)
		updateClauses = append(updateClauses, fmt.Sprintf("canceled_at = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if quote.CreatedAt != nil {
		args = append(args, *quote.CreatedAt)
		updateClauses = append(updateClauses, fmt.Sprintf("created_at = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	args = append(args, time.Now())
	updateClauses = append(updateClauses, fmt.Sprintf("updated_at = $%d", argIndex))

	if len(updateClauses) > 0 {
		query += strings.Join(updateClauses, ", ")
	}
	query += " WHERE id = $1"

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to upsert quote: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteQuote(ctx, id)
}
