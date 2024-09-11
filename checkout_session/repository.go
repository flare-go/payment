package checkout_session

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
	const query = `
    INSERT INTO checkout_sessions (id, customer_id, payment_intent_id, status, mode, success_url, cancel_url, amount_total, currency, created_at, updated_at)
    VALUES (@id, @customer_id, @payment_intent_id, @status, @mode, @success_url, @cancel_url, @amount_total, @currency, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        customer_id = COALESCE(@customer_id, checkout_sessions.customer_id),
        payment_intent_id = COALESCE(@payment_intent_id, checkout_sessions.payment_intent_id),
        status = COALESCE(@status, checkout_sessions.status),
        mode = COALESCE(@mode, checkout_sessions.mode),
        success_url = COALESCE(@success_url, checkout_sessions.success_url),
        cancel_url = COALESCE(@cancel_url, checkout_sessions.cancel_url),
        amount_total = COALESCE(@amount_total, checkout_sessions.amount_total),
        currency = COALESCE(@currency, checkout_sessions.currency),
        updated_at = @updated_at
    WHERE checkout_sessions.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":                session.ID,
		"customer_id":       session.CustomerID,
		"payment_intent_id": session.PaymentIntentID,
		"status":            session.Status,
		"mode":              session.Mode,
		"success_url":       session.SuccessURL,
		"cancel_url":        session.CancelURL,
		"amount_total":      session.AmountTotal,
		"currency":          session.Currency,
		"created_at":        session.CreatedAt,
		"updated_at":        now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert checkout session: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteCheckOutSession(ctx, id)
}
