package promotion_code

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
	Upsert(ctx context.Context, tx pgx.Tx, promotionCode *models.PartialPromotionCode) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, promotionCode *models.PartialPromotionCode) error {
	query := `
    INSERT INTO promotion_codes (id, code, coupon_id, customer_id, active, max_redemptions, times_redeemed, expires_at, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    ON CONFLICT (id) DO UPDATE SET
    `
	args := []interface{}{promotionCode.ID}
	var updateClauses []string
	argIndex := 2

	if promotionCode.Code != nil {
		args = append(args, *promotionCode.Code)
		updateClauses = append(updateClauses, fmt.Sprintf("code = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if promotionCode.CouponID != nil {
		args = append(args, *promotionCode.CouponID)
		updateClauses = append(updateClauses, fmt.Sprintf("coupon_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if promotionCode.CustomerID != nil {
		args = append(args, *promotionCode.CustomerID)
		updateClauses = append(updateClauses, fmt.Sprintf("customer_id = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if promotionCode.Active != nil {
		args = append(args, *promotionCode.Active)
		updateClauses = append(updateClauses, fmt.Sprintf("active = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if promotionCode.MaxRedemptions != nil {
		args = append(args, *promotionCode.MaxRedemptions)
		updateClauses = append(updateClauses, fmt.Sprintf("max_redemptions = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if promotionCode.TimesRedeemed != nil {
		args = append(args, *promotionCode.TimesRedeemed)
		updateClauses = append(updateClauses, fmt.Sprintf("times_redeemed = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if promotionCode.ExpiresAt != nil {
		args = append(args, *promotionCode.ExpiresAt)
		updateClauses = append(updateClauses, fmt.Sprintf("expires_at = $%d", argIndex))
		argIndex++
	} else {
		args = append(args, nil)
	}

	if promotionCode.CreatedAt != nil {
		args = append(args, *promotionCode.CreatedAt)
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
		return fmt.Errorf("failed to upsert promotion code: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeletePromotionCodes(ctx, id)
}
