package charge

import (
	"context"
	"fmt"
	"strings"
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
	query := `
    INSERT INTO charges (id, customer_id, payment_intent_id, amount, currency, status, paid, refunded, failure_code, failure_message, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{charge.ID}
	var updateClauses []string
	argIndex := 2

	if charge.CustomerID != nil {
		args = append(args, *charge.CustomerID)
		updateClauses = append(updateClauses, fmt.Sprintf("customer_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if charge.PaymentIntentID != nil {
		args = append(args, *charge.PaymentIntentID)
		updateClauses = append(updateClauses, fmt.Sprintf("payment_intent_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if charge.Amount != nil {
		args = append(args, *charge.Amount)
		updateClauses = append(updateClauses, fmt.Sprintf("amount = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if charge.Currency != nil {
		args = append(args, *charge.Currency)
		updateClauses = append(updateClauses, fmt.Sprintf("currency = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if charge.Status != nil {
		args = append(args, *charge.Status)
		updateClauses = append(updateClauses, fmt.Sprintf("status = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if charge.Paid != nil {
		args = append(args, *charge.Paid)
		updateClauses = append(updateClauses, fmt.Sprintf("paid = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if charge.Refunded != nil {
		args = append(args, *charge.Refunded)
		updateClauses = append(updateClauses, fmt.Sprintf("refunded = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if charge.FailureCode != nil {
		args = append(args, *charge.FailureCode)
		updateClauses = append(updateClauses, fmt.Sprintf("failure_code = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if charge.FailureMessage != nil {
		args = append(args, *charge.FailureMessage)
		updateClauses = append(updateClauses, fmt.Sprintf("failure_message = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if charge.CreatedAt != nil {
		args = append(args, *charge.CreatedAt)
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
		return fmt.Errorf("failed to upsert charge: %w", err)
	}

	return nil
}
