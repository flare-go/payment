package models

import "time"

// Product 代表可訂閱或購買的產品
// Product represents a product that can be subscribed to or purchased
type Product struct {
	ID          uint64            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Active      bool              `json:"active"`
	StripeID    string            `json:"stripe_id"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
