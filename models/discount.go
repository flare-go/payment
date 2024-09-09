package models

import "time"

type Discount struct {
	ID         string     `json:"id"`
	CustomerID string     `json:"customer_id"`
	CouponID   string     `json:"coupon_id"`
	Start      time.Time  `json:"start"`
	End        *time.Time `json:"end"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type PartialDiscount struct {
	ID         string
	CustomerID *string
	CouponID   *string
	Start      *time.Time
	End        *time.Time
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}
