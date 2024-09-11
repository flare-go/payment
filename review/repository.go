package review

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
	const query = `
    INSERT INTO reviews (id, payment_intent_id, reason, closed_reason, status, opened_at, closed_at, created_at, updated_at)
    VALUES (@id, @payment_intent_id, @reason, @closed_reason, @status, @opened_at, @closed_at, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        payment_intent_id = COALESCE(@payment_intent_id, reviews.payment_intent_id),
        reason = COALESCE(@reason, reviews.reason),
        closed_reason = COALESCE(@closed_reason, reviews.closed_reason),
        status = COALESCE(@status, reviews.status),
        opened_at = COALESCE(@opened_at, reviews.opened_at),
        closed_at = COALESCE(@closed_at, reviews.closed_at),
        updated_at = @updated_at
    WHERE reviews.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":                review.ID,
		"payment_intent_id": review.PaymentIntentID,
		"reason":            review.Reason,
		"closed_reason":     review.ClosedReason,
		"status":            review.Status,
		"opened_at":         review.OpenedAt,
		"closed_at":         review.ClosedAt,
		"created_at":        review.CreatedAt,
		"updated_at":        now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert review: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteReviews(ctx, id)
}
