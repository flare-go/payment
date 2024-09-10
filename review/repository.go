// repository/review/repository.go
package review

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
	Upsert(ctx context.Context, tx pgx.Tx, review *models.PartialReview) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, review *models.PartialReview) error {
	query := `
    INSERT INTO reviews (id, payment_intent_id, reason, status, opened_at, closed_at, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{review.ID}
	var updateClauses []string
	argIndex := 2

	if review.PaymentIntentID != nil {
		args = append(args, *review.PaymentIntentID)
		updateClauses = append(updateClauses, fmt.Sprintf("payment_intent_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if review.Reason != nil {
		args = append(args, *review.Reason)
		updateClauses = append(updateClauses, fmt.Sprintf("reason = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if review.ClosedReason != nil {
		args = append(args, *review.ClosedReason)
		updateClauses = append(updateClauses, fmt.Sprintf("closed_reason = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if review.Status != nil {
		args = append(args, *review.Status)
		updateClauses = append(updateClauses, fmt.Sprintf("status = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if review.OpenedAt != nil {
		args = append(args, *review.OpenedAt)
		updateClauses = append(updateClauses, fmt.Sprintf("opened_at = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if review.ClosedAt != nil {
		args = append(args, *review.ClosedAt)
		updateClauses = append(updateClauses, fmt.Sprintf("closed_at = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if review.CreatedAt != nil {
		args = append(args, *review.CreatedAt)
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
		return fmt.Errorf("failed to upsert review: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteReviews(ctx, id)
}
