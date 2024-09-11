package promotion_code

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
	const query = `
    INSERT INTO promotion_codes (id, code, coupon_id, customer_id, active, max_redemptions, times_redeemed, expires_at, created_at, updated_at)
    VALUES (@id, @code, @coupon_id, @customer_id, @active, @max_redemptions, @times_redeemed, @expires_at, COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        code = COALESCE(@code, promotion_codes.code),
        coupon_id = COALESCE(@coupon_id, promotion_codes.coupon_id),
        customer_id = COALESCE(@customer_id, promotion_codes.customer_id),
        active = COALESCE(@active, promotion_codes.active),
        max_redemptions = COALESCE(@max_redemptions, promotion_codes.max_redemptions),
        times_redeemed = COALESCE(@times_redeemed, promotion_codes.times_redeemed),
        expires_at = COALESCE(@expires_at, promotion_codes.expires_at),
        updated_at = @updated_at
    WHERE promotion_codes.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":              promotionCode.ID,
		"code":            promotionCode.Code,
		"coupon_id":       promotionCode.CouponID,
		"customer_id":     promotionCode.CustomerID,
		"active":          promotionCode.Active,
		"max_redemptions": promotionCode.MaxRedemptions,
		"times_redeemed":  promotionCode.TimesRedeemed,
		"expires_at":      promotionCode.ExpiresAt,
		"created_at":      promotionCode.CreatedAt,
		"updated_at":      now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert promotion code: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeletePromotionCodes(ctx, id)
}
