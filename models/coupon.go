package models

import "time"

type Coupon struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	AmountOff        int64      `json:"amount_off,omitempty"`
	PercentOff       float64    `json:"percent_off,omitempty"`
	Currency         string     `json:"currency,omitempty"`
	Duration         string     `json:"duration"`
	DurationInMonths int        `json:"duration_in_months,omitempty"`
	MaxRedemptions   int        `json:"max_redemptions,omitempty"`
	TimesRedeemed    int        `json:"times_redeemed"`
	Valid            bool       `json:"valid"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	RedeemBy         *time.Time `json:"redeem_by,omitempty"`
}

type PartialCoupon struct {
	ID               string
	Name             *string
	AmountOff        *int64
	PercentOff       *float64
	Currency         *string
	Duration         *string
	DurationInMonths *int
	MaxRedemptions   *int
	TimesRedeemed    *int32
	Valid            *bool
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
	RedeemBy         *time.Time
}
