package charge

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
)

type Repository interface {
	Upsert(ctx context.Context, tx pgx.Tx, charge *models.PartialCharge) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, charge *models.PartialCharge) error {
	const query = `
    INSERT INTO charges (id, customer_id, payment_intent_id, amount, currency, status, paid, refunded, failure_code, failure_message, created_at, updated_at)
    VALUES (@id, @customer_id, @payment_intent_id, @amount, @currency, @status, @paid, @refunded, @failure_code, @failure_message, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        customer_id = COALESCE(@customer_id, charges.customer_id),
        payment_intent_id = COALESCE(@payment_intent_id, charges.payment_intent_id),
        amount = COALESCE(@amount, charges.amount),
        currency = COALESCE(@currency, charges.currency),
        status = COALESCE(@status, charges.status),
        paid = COALESCE(@paid, charges.paid),
        refunded = COALESCE(@refunded, charges.refunded),
        failure_code = COALESCE(@failure_code, charges.failure_code),
        failure_message = COALESCE(@failure_message, charges.failure_message),
        updated_at = @updated_at
    WHERE charges.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":                charge.ID,
		"customer_id":       charge.CustomerID,
		"payment_intent_id": charge.PaymentIntentID,
		"amount":            charge.Amount,
		"currency":          charge.Currency,
		"status":            charge.Status,
		"paid":              charge.Paid,
		"refunded":          charge.Refunded,
		"failure_code":      charge.FailureCode,
		"failure_message":   charge.FailureMessage,
		"created_at":        charge.CreatedAt,
		"updated_at":        now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert charge: %w", err)
	}

	return nil
}
