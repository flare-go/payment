package disputes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
	"strings"
	"time"
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
		EvidenceDueBy: pgtype.Timestamptz{Time: dispute.EvidenceDueBy},
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
		EvidenceDueBy: pgtype.Timestamptz{Time: dispute.EvidenceDueBy},
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
	query := `
    INSERT INTO disputes (id, charge_id, amount, status, reason, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{dispute.ID}
	updateClauses := []string{}
	argIndex := 2

	if dispute.ChargeID != nil {
		args = append(args, *dispute.ChargeID)
		updateClauses = append(updateClauses, fmt.Sprintf("charge_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if dispute.Amount != nil {
		args = append(args, *dispute.Amount)
		updateClauses = append(updateClauses, fmt.Sprintf("amount = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if dispute.Currency != nil {
		args = append(args, *dispute.Currency)
		updateClauses = append(updateClauses, fmt.Sprintf("currency = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if dispute.Status != nil {
		args = append(args, *dispute.Status)
		updateClauses = append(updateClauses, fmt.Sprintf("status = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if dispute.Reason != nil {
		args = append(args, *dispute.Reason)
		updateClauses = append(updateClauses, fmt.Sprintf("reason = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if dispute.CreatedAt != nil {
		args = append(args, *dispute.CreatedAt)
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
		return fmt.Errorf("failed to upsert dispute: %w", err)
	}

	return nil
}
