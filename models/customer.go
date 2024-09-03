package models

import (
	"time"
)

// Customer 代表系統中的客戶
// Customer represents a customer in the system
type Customer struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"user_id"`
	Balance   int64     `json:"balance"`
	StripeID  string    `json:"stripe_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
