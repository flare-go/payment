// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: coupon.sql

package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createCoupon = `-- name: CreateCoupon :one
INSERT INTO coupons (
    id, name, amount_off, percent_off, currency, duration,
    duration_in_months, max_redemptions, times_redeemed, valid, redeem_by
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
         )
RETURNING id, name, amount_off, percent_off, currency, duration, duration_in_months, max_redemptions, times_redeemed, valid, created_at, updated_at, redeem_by
`

type CreateCouponParams struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	AmountOff        int64              `json:"amountOff"`
	PercentOff       float64            `json:"percentOff"`
	Currency         Currency           `json:"currency"`
	Duration         CouponDuration     `json:"duration"`
	DurationInMonths int32              `json:"durationInMonths"`
	MaxRedemptions   int32              `json:"maxRedemptions"`
	TimesRedeemed    int32              `json:"timesRedeemed"`
	Valid            bool               `json:"valid"`
	RedeemBy         pgtype.Timestamptz `json:"redeemBy"`
}

func (q *Queries) CreateCoupon(ctx context.Context, arg CreateCouponParams) (*Coupon, error) {
	row := q.db.QueryRow(ctx, createCoupon,
		arg.ID,
		arg.Name,
		arg.AmountOff,
		arg.PercentOff,
		arg.Currency,
		arg.Duration,
		arg.DurationInMonths,
		arg.MaxRedemptions,
		arg.TimesRedeemed,
		arg.Valid,
		arg.RedeemBy,
	)
	var i Coupon
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.AmountOff,
		&i.PercentOff,
		&i.Currency,
		&i.Duration,
		&i.DurationInMonths,
		&i.MaxRedemptions,
		&i.TimesRedeemed,
		&i.Valid,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.RedeemBy,
	)
	return &i, err
}

const deleteCoupon = `-- name: DeleteCoupon :exec
DELETE FROM coupons WHERE id = $1
`

func (q *Queries) DeleteCoupon(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx, deleteCoupon, id)
	return err
}

const getCouponByID = `-- name: GetCouponByID :one
SELECT id, name, amount_off, percent_off, currency, duration, duration_in_months, max_redemptions, times_redeemed, valid, created_at, updated_at, redeem_by FROM coupons WHERE id = $1 LIMIT 1
`

func (q *Queries) GetCouponByID(ctx context.Context, id string) (*Coupon, error) {
	row := q.db.QueryRow(ctx, getCouponByID, id)
	var i Coupon
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.AmountOff,
		&i.PercentOff,
		&i.Currency,
		&i.Duration,
		&i.DurationInMonths,
		&i.MaxRedemptions,
		&i.TimesRedeemed,
		&i.Valid,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.RedeemBy,
	)
	return &i, err
}

const listCoupons = `-- name: ListCoupons :many
SELECT id, name, amount_off, percent_off, currency, duration, duration_in_months, max_redemptions, times_redeemed, valid, created_at, updated_at, redeem_by FROM coupons
ORDER BY id
LIMIT $1 OFFSET $2
`

type ListCouponsParams struct {
	Limit  int64 `json:"limit"`
	Offset int64 `json:"offset"`
}

func (q *Queries) ListCoupons(ctx context.Context, arg ListCouponsParams) ([]*Coupon, error) {
	rows, err := q.db.Query(ctx, listCoupons, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*Coupon{}
	for rows.Next() {
		var i Coupon
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.AmountOff,
			&i.PercentOff,
			&i.Currency,
			&i.Duration,
			&i.DurationInMonths,
			&i.MaxRedemptions,
			&i.TimesRedeemed,
			&i.Valid,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.RedeemBy,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateCoupon = `-- name: UpdateCoupon :one
UPDATE coupons
SET name = $2,
    amount_off = $3,
    percent_off = $4,
    currency = $5,
    duration = $6,
    duration_in_months = $7,
    max_redemptions = $8,
    times_redeemed = $9,
    valid = $10,
    redeem_by = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, amount_off, percent_off, currency, duration, duration_in_months, max_redemptions, times_redeemed, valid, created_at, updated_at, redeem_by
`

type UpdateCouponParams struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	AmountOff        int64              `json:"amountOff"`
	PercentOff       float64            `json:"percentOff"`
	Currency         Currency           `json:"currency"`
	Duration         CouponDuration     `json:"duration"`
	DurationInMonths int32              `json:"durationInMonths"`
	MaxRedemptions   int32              `json:"maxRedemptions"`
	TimesRedeemed    int32              `json:"timesRedeemed"`
	Valid            bool               `json:"valid"`
	RedeemBy         pgtype.Timestamptz `json:"redeemBy"`
}

func (q *Queries) UpdateCoupon(ctx context.Context, arg UpdateCouponParams) (*Coupon, error) {
	row := q.db.QueryRow(ctx, updateCoupon,
		arg.ID,
		arg.Name,
		arg.AmountOff,
		arg.PercentOff,
		arg.Currency,
		arg.Duration,
		arg.DurationInMonths,
		arg.MaxRedemptions,
		arg.TimesRedeemed,
		arg.Valid,
		arg.RedeemBy,
	)
	var i Coupon
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.AmountOff,
		&i.PercentOff,
		&i.Currency,
		&i.Duration,
		&i.DurationInMonths,
		&i.MaxRedemptions,
		&i.TimesRedeemed,
		&i.Valid,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.RedeemBy,
	)
	return &i, err
}

const upsertCoupon = `-- name: UpsertCoupon :exec
INSERT INTO coupons (
    id,
    name,
    currency,
    duration,
    amount_off,
    percent_off,
    duration_in_months,
    max_redemptions,
    times_redeemed,
    valid,
    redeem_by,
    created_at,
    updated_at
) VALUES (
             $1,
             $7,
             $8,
             $9,
             $10,
             $11,
             $12,
             $13,
             $2,
             $3,
             $4,
             $5,
             $6
         )
ON CONFLICT (id) DO UPDATE SET
                               name = COALESCE($7, coupons.name),
                               currency = COALESCE($8, coupons.currency),
                               duration = COALESCE($9, coupons.duration),
                               amount_off = COALESCE($10, coupons.amount_off),
                               percent_off = COALESCE($11, coupons.percent_off),
                               duration_in_months = COALESCE($12, coupons.duration_in_months),
                               max_redemptions = COALESCE($13, coupons.max_redemptions),
                               times_redeemed = $2,
                               valid = $3,
                               redeem_by = $4,
                               updated_at = $6
`

type UpsertCouponParams struct {
	ID               string             `json:"id"`
	TimesRedeemed    int32              `json:"timesRedeemed"`
	Valid            bool               `json:"valid"`
	RedeemBy         pgtype.Timestamptz `json:"redeemBy"`
	CreatedAt        pgtype.Timestamptz `json:"createdAt"`
	UpdatedAt        pgtype.Timestamptz `json:"updatedAt"`
	Name             *string            `json:"name"`
	Currency         NullCurrency       `json:"currency"`
	Duration         NullCouponDuration `json:"duration"`
	AmountOff        *int64             `json:"amountOff"`
	PercentOff       float64            `json:"percentOff"`
	DurationInMonths *int32             `json:"durationInMonths"`
	MaxRedemptions   *int32             `json:"maxRedemptions"`
}

func (q *Queries) UpsertCoupon(ctx context.Context, arg UpsertCouponParams) error {
	_, err := q.db.Exec(ctx, upsertCoupon,
		arg.ID,
		arg.TimesRedeemed,
		arg.Valid,
		arg.RedeemBy,
		arg.CreatedAt,
		arg.UpdatedAt,
		arg.Name,
		arg.Currency,
		arg.Duration,
		arg.AmountOff,
		arg.PercentOff,
		arg.DurationInMonths,
		arg.MaxRedemptions,
	)
	return err
}
