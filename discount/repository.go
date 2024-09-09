// repository/discount/repository.go

package discount

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
	Upsert(ctx context.Context, tx pgx.Tx, discount *models.PartialDiscount) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, discount *models.PartialDiscount) error {
	query := `
    INSERT INTO discounts (id, customer_id, coupon_id, start, "end", created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{discount.ID}
	updateClauses := []string{}
	argIndex := 2

	if discount.CustomerID != nil {
		args = append(args, *discount.CustomerID)
		updateClauses = append(updateClauses, fmt.Sprintf("customer_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if discount.CouponID != nil {
		args = append(args, *discount.CouponID)
		updateClauses = append(updateClauses, fmt.Sprintf("coupon_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if discount.Start != nil {
		args = append(args, *discount.Start)
		updateClauses = append(updateClauses, fmt.Sprintf("start = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if discount.End != nil {
		args = append(args, *discount.End)
		updateClauses = append(updateClauses, fmt.Sprintf("\"end\" = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if discount.CreatedAt != nil {
		args = append(args, *discount.CreatedAt)
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

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteDiscount(ctx, id)
}
