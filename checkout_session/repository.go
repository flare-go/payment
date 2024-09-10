package checkout_session

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
	"strings"
	"time"
)

type Repository interface {
	Upsert(ctx context.Context, tx pgx.Tx, session *models.PartialCheckoutSession) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, session *models.PartialCheckoutSession) error {
	query := `
    INSERT INTO checkout_sessions (id, customer_id, payment_intent_id, status, mode, success_url, cancel_url, amount_total, currency, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{session.ID}
	var updateClauses []string
	argIndex := 2

	if session.CustomerID != nil {
		args = append(args, *session.CustomerID)
		updateClauses = append(updateClauses, fmt.Sprintf("customer_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if session.PaymentIntentID != nil {
		args = append(args, *session.PaymentIntentID)
		updateClauses = append(updateClauses, fmt.Sprintf("payment_intent_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if session.Status != nil {
		args = append(args, *session.Status)
		updateClauses = append(updateClauses, fmt.Sprintf("status = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if session.Mode != nil {
		args = append(args, *session.Mode)
		updateClauses = append(updateClauses, fmt.Sprintf("mode = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if session.SuccessURL != nil {
		args = append(args, *session.SuccessURL)
		updateClauses = append(updateClauses, fmt.Sprintf("success_url = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if session.CancelURL != nil {
		args = append(args, *session.CancelURL)
		updateClauses = append(updateClauses, fmt.Sprintf("cancel_url = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if session.AmountTotal != nil {
		args = append(args, *session.AmountTotal)
		updateClauses = append(updateClauses, fmt.Sprintf("amount_total = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if session.Currency != nil {
		args = append(args, *session.Currency)
		updateClauses = append(updateClauses, fmt.Sprintf("currency = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if session.CreatedAt != nil {
		args = append(args, *session.CreatedAt)
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
		return fmt.Errorf("failed to upsert checkout session: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteCheckOutSession(ctx, id)
}
