package payment_link

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
	query := `
    INSERT INTO payment_links (id, active, url, amount, currency, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{paymentLink.ID}
	var updateClauses []string
	argIndex := 2

	if paymentLink.Active != nil {
		args = append(args, *paymentLink.Active)
		updateClauses = append(updateClauses, fmt.Sprintf("active = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentLink.URL != nil {
		args = append(args, *paymentLink.URL)
		updateClauses = append(updateClauses, fmt.Sprintf("url = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentLink.Amount != nil {
		args = append(args, *paymentLink.Amount)
		updateClauses = append(updateClauses, fmt.Sprintf("amount = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentLink.Currency != nil {
		args = append(args, *paymentLink.Currency)
		updateClauses = append(updateClauses, fmt.Sprintf("currency = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if paymentLink.CreatedAt != nil {
		args = append(args, *paymentLink.CreatedAt)
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
		return fmt.Errorf("failed to upsert payment link: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeletePaymentLink(ctx, id)
}
