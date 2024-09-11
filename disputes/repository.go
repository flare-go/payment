package disputes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
)

type Repository interface {
	Create(ctx context.Context, dispute *models.Dispute) error
	GetByID(ctx context.Context, id string) (*models.Dispute, error)
	Update(ctx context.Context, dispute *models.Dispute) error
	Close(ctx context.Context, id string) error
	Upsert(ctx context.Context, tx pgx.Tx, dispute *models.PartialDispute) error
}

type repository struct {
	conn   driver.PostgresPool
	logger *zap.Logger
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger) Repository {
	return &repository{
		conn:   conn,
		logger: logger,
	}
}

func (r *repository) Create(ctx context.Context, dispute *models.Dispute) error {

	if err := sqlc.New(r.conn).CreateDispute(ctx, sqlc.CreateDisputeParams{
		ID:            dispute.ID,
		ChargeID:      dispute.ChargeID,
		Amount:        float64(dispute.Amount),
		Currency:      sqlc.Currency(dispute.Currency),
		Status:        sqlc.DisputeStatus(dispute.Status),
		Reason:        sqlc.DisputeReason(dispute.Reason),
		EvidenceDueBy: pgtype.Timestamptz{Time: dispute.EvidenceDueBy, Valid: true},
	}); err != nil {
		r.logger.Error("Failed to create dispute", zap.Error(err))
		return fmt.Errorf("failed to create dispute: %w", err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id string) (*models.Dispute, error) {

	dispute, err := sqlc.New(r.conn).GetDisputeByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("dispute not found: %w", err)
		}
		r.logger.Error("Failed to get dispute by ID", zap.Error(err), zap.String("id", id))
		return nil, fmt.Errorf("failed to get dispute: %w", err)
	}

	return models.NewDispute().ConvertFromSQLCDispute(dispute), nil
}

func (r *repository) Update(ctx context.Context, dispute *models.Dispute) error {
	q := sqlc.New(r.conn)

	err := q.UpdateDispute(ctx, sqlc.UpdateDisputeParams{
		ID:            dispute.ID,
		ChargeID:      dispute.ChargeID,
		Amount:        float64(dispute.Amount),
		Currency:      sqlc.Currency(dispute.Currency),
		Status:        sqlc.DisputeStatus(dispute.Status),
		Reason:        sqlc.DisputeReason(dispute.Reason),
		EvidenceDueBy: pgtype.Timestamptz{Time: dispute.EvidenceDueBy, Valid: true},
	})

	if err != nil {
		r.logger.Error("Failed to update dispute", zap.Error(err), zap.String("id", dispute.ID))
		return fmt.Errorf("failed to update dispute: %w", err)
	}

	return nil
}

func (r *repository) Close(ctx context.Context, id string) error {

	return sqlc.New(r.conn).CloseDispute(ctx, sqlc.CloseDisputeParams{
		ID: id,
	})
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, dispute *models.PartialDispute) error {
	const query = `
    INSERT INTO disputes (id, charge_id, amount, status, reason, currency, evidence_due_by, created_at, updated_at)
    VALUES (@id, @charge_id, @amount, @status, @reason, @currency, @evidence_due_by, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        charge_id = COALESCE(@charge_id, disputes.charge_id),
        amount = COALESCE(@amount, disputes.amount),
        status = COALESCE(@status, disputes.status),
        reason = COALESCE(@reason, disputes.reason),
        currency = COALESCE(@currency, disputes.currency),
        evidence_due_by = COALESCE(@evidence_due_by, @evidence_due_by),
        updated_at = @updated_at
    WHERE disputes.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":              dispute.ID,
		"charge_id":       dispute.ChargeID,
		"amount":          dispute.Amount,
		"status":          dispute.Status,
		"reason":          dispute.Reason,
		"currency":        dispute.Currency,
		"evidence_due_by": dispute.EvidenceDueBy,
		"created_at":      dispute.CreatedAt,
		"updated_at":      now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert dispute: %w", err)
	}

	return nil
}
