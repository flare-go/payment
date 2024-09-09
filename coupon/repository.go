// repository/coupon/repository.go

package coupon

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
	Upsert(ctx context.Context, tx pgx.Tx, coupon *models.PartialCoupon) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, coupon *models.PartialCoupon) error {
	query := `
    INSERT INTO coupons (id, name, amount_off, percent_off, currency, duration, duration_in_months, max_redemptions, times_redeemed, valid, created_at, updated_at, redeem_by)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{coupon.ID}
	var updateClauses []string
	argIndex := 2

	if coupon.Name != nil {
		args = append(args, *coupon.Name)
		updateClauses = append(updateClauses, fmt.Sprintf("name = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if coupon.AmountOff != nil {
		args = append(args, *coupon.AmountOff)
		updateClauses = append(updateClauses, fmt.Sprintf("amount_off = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if coupon.PercentOff != nil {
		args = append(args, *coupon.PercentOff)
		updateClauses = append(updateClauses, fmt.Sprintf("percent_off = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if coupon.Currency != nil {
		args = append(args, *coupon.Currency)
		updateClauses = append(updateClauses, fmt.Sprintf("currency = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if coupon.Duration != nil {
		args = append(args, *coupon.Duration)
		updateClauses = append(updateClauses, fmt.Sprintf("duration = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if coupon.DurationInMonths != nil {
		args = append(args, *coupon.DurationInMonths)
		updateClauses = append(updateClauses, fmt.Sprintf("duration_in_months = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if coupon.MaxRedemptions != nil {
		args = append(args, *coupon.MaxRedemptions)
		updateClauses = append(updateClauses, fmt.Sprintf("max_redemptions = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if coupon.TimesRedeemed != nil {
		args = append(args, *coupon.TimesRedeemed)
		updateClauses = append(updateClauses, fmt.Sprintf("times_redeemed = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if coupon.Valid != nil {
		args = append(args, *coupon.Valid)
		updateClauses = append(updateClauses, fmt.Sprintf("valid = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if coupon.CreatedAt != nil {
		args = append(args, *coupon.CreatedAt)
		updateClauses = append(updateClauses, fmt.Sprintf("created_at = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	args = append(args, time.Now())
	updateClauses = append(updateClauses, fmt.Sprintf("updated_at = $%d", argIndex))
	argIndex++

	if coupon.RedeemBy != nil {
		args = append(args, *coupon.RedeemBy)
		updateClauses = append(updateClauses, fmt.Sprintf("redeem_by = $%d", argIndex))
	} else {
		args = append(args, nil)
	}

	if len(updateClauses) > 0 {
		query += strings.Join(updateClauses, ", ")
	}
	query += " WHERE id = $1"

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to upsert coupon: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteCoupon(ctx, id)
}
