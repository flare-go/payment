package models

import "time"

type PromotionCode struct {
	ID             string     `json:"id"`
	Code           string     `json:"code"`
	CouponID       string     `json:"coupon_id"`
	CustomerID     *string    `json:"customer_id,omitempty"`
	Active         bool       `json:"active"`
	MaxRedemptions *int       `json:"max_redemptions,omitempty"`
	TimesRedeemed  int        `json:"times_redeemed"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type PartialPromotionCode struct {
	ID             string     `json:"id"`
	Code           *string    `json:"code,omitempty"`
	CouponID       *string    `json:"coupon_id,omitempty"`
	CustomerID     *string    `json:"customer_id,omitempty"`
	Active         *bool      `json:"active,omitempty"`
	MaxRedemptions *int       `json:"max_redemptions,omitempty"`
	TimesRedeemed  *int       `json:"times_redeemed,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}
