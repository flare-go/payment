package models

import (
	"encoding/json"
	"log"
	"time"

	"goflare.io/payment/sqlc"
)

// Product 代表可訂閱或購買的產品
// Product represents a product that can be subscribed to or purchased
type Product struct {
	ID          uint64            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Active      bool              `json:"active"`
	StripeID    string            `json:"stripe_id"`
	Metadata    map[string]string `json:"metadata"`
	Prices      []*Price          `json:"prices"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func NewProduct() *Product {
	return &Product{}
}

func (p *Product) ConvertFromSQLCProduct(sqlcProduct any) *Product {

	var (
		id                   uint64
		name, desc, stripeID string
		active               bool
		metadata             map[string]string
		createdAt, updatedAt time.Time
	)

	switch sp := sqlcProduct.(type) {
	case *sqlc.Product:
		id = sp.ID
		name = sp.Name
		if sp.Description != nil {
			desc = *sp.Description
		}
		stripeID = sp.StripeID
		active = sp.Active
		if err := json.Unmarshal(sp.Metadata, &metadata); err != nil {
			log.Println("Error unmarshalling metadata:", err)
		}
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	p.ID = id
	p.Name = name
	p.Description = desc
	p.StripeID = stripeID
	p.Active = active
	p.Metadata = metadata
	p.CreatedAt = createdAt
	p.UpdatedAt = updatedAt

	return p
}
