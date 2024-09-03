package models

import (
	"goflare.io/payment/sqlc"
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

func NewCustomer() *Customer {
	return &Customer{}
}

func (c *Customer) ConvertFromSQLCCustomer(sqlcCustomer any) *Customer {

	var id, userID uint64
	var balance int64
	var stripeID string
	var createdAt, updatedAt time.Time

	switch sp := sqlcCustomer.(type) {
	case *sqlc.Customer:
		id = sp.ID
		userID = sp.UserID
		balance = sp.Balance
		stripeID = sp.StripeID
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	c.ID = id
	c.UserID = userID
	c.Balance = balance
	c.StripeID = stripeID
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt

	return c
}
