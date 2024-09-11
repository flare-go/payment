package coupon

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
	const query = `
    INSERT INTO coupons (id, name, amount_off, percent_off, currency, duration, duration_in_months, max_redemptions, times_redeemed, valid, created_at, updated_at, redeem_by)
    VALUES (@id, @name, @amount_off, @percent_off, @currency, @duration, @duration_in_months, @max_redemptions, @times_redeemed, @valid, COALESCE(@created_at, NOW()), @updated_at, @redeem_by)
    ON CONFLICT (id) DO UPDATE SET
        name = COALESCE(@name, coupons.name),
        amount_off = COALESCE(@amount_off, coupons.amount_off),
        percent_off = COALESCE(@percent_off, coupons.percent_off),
        currency = COALESCE(@currency, coupons.currency),
        duration = COALESCE(@duration, coupons.duration),
        duration_in_months = COALESCE(@duration_in_months, coupons.duration_in_months),
        max_redemptions = COALESCE(@max_redemptions, coupons.max_redemptions),
        times_redeemed = COALESCE(@times_redeemed, coupons.times_redeemed),
        valid = COALESCE(@valid, coupons.valid),
        updated_at = @updated_at,
        redeem_by = COALESCE(@redeem_by, coupons.redeem_by)
    WHERE coupons.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":                 coupon.ID,
		"name":               coupon.Name,
		"amount_off":         coupon.AmountOff,
		"percent_off":        coupon.PercentOff,
		"currency":           coupon.Currency,
		"duration":           coupon.Duration,
		"duration_in_months": coupon.DurationInMonths,
		"max_redemptions":    coupon.MaxRedemptions,
		"times_redeemed":     coupon.TimesRedeemed,
		"valid":              coupon.Valid,
		"created_at":         coupon.CreatedAt,
		"updated_at":         now,
		"redeem_by":          coupon.RedeemBy,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert coupon: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteCoupon(ctx, id)
}
