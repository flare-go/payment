package tax_rate

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
	const query = `
    INSERT INTO tax_rates (id, display_name, description, jurisdiction, percentage, inclusive, active, created_at, updated_at)
    VALUES (@id, @display_name, @description, @jurisdiction, @percentage, @inclusive, @active, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        display_name = COALESCE(@display_name, tax_rates.display_name),
        description = COALESCE(@description, tax_rates.description),
        jurisdiction = COALESCE(@jurisdiction, tax_rates.jurisdiction),
        percentage = COALESCE(@percentage, tax_rates.percentage),
        inclusive = COALESCE(@inclusive, tax_rates.inclusive),
        active = COALESCE(@active, tax_rates.active),
        updated_at = @updated_at
    WHERE tax_rates.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":           taxRate.ID,
		"display_name": taxRate.DisplayName,
		"description":  taxRate.Description,
		"jurisdiction": taxRate.Jurisdiction,
		"percentage":   taxRate.Percentage,
		"inclusive":    taxRate.Inclusive,
		"active":       taxRate.Active,
		"created_at":   taxRate.CreatedAt,
		"updated_at":   now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert tax rate: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteReviews(ctx, id)
}
