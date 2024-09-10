package tax_rate

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
	Upsert(ctx context.Context, tx pgx.Tx, taxRate *models.PartialTaxRate) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, taxRate *models.PartialTaxRate) error {
	query := `
    INSERT INTO tax_rates (id, display_name, description, jurisdiction, percentage, inclusive, active, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{taxRate.ID}
	var updateClauses []string
	argIndex := 2

	if taxRate.DisplayName != nil {
		args = append(args, *taxRate.DisplayName)
		updateClauses = append(updateClauses, fmt.Sprintf("display_name = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if taxRate.Description != nil {
		args = append(args, *taxRate.Description)
		updateClauses = append(updateClauses, fmt.Sprintf("description = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if taxRate.Jurisdiction != nil {
		args = append(args, *taxRate.Jurisdiction)
		updateClauses = append(updateClauses, fmt.Sprintf("jurisdiction = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if taxRate.Percentage != nil {
		args = append(args, *taxRate.Percentage)
		updateClauses = append(updateClauses, fmt.Sprintf("percentage = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if taxRate.Inclusive != nil {
		args = append(args, *taxRate.Inclusive)
		updateClauses = append(updateClauses, fmt.Sprintf("inclusive = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if taxRate.Active != nil {
		args = append(args, *taxRate.Active)
		updateClauses = append(updateClauses, fmt.Sprintf("active = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if taxRate.CreatedAt != nil {
		args = append(args, *taxRate.CreatedAt)
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
		return fmt.Errorf("failed to upsert tax rate: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteReviews(ctx, id)
}
