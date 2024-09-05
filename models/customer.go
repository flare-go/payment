package models

import (
	"time"

	"goflare.io/payment/sqlc"
)

// Customer 代表系統中的客戶
// Customer represents a customer in the system
type Customer struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
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
	var name, email, stripeID string
	var createdAt, updatedAt time.Time

	switch sp := sqlcCustomer.(type) {
	case *sqlc.Customer:
		id = sp.ID
		userID = sp.UserID
		balance = sp.Balance
		stripeID = sp.StripeID
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	case *sqlc.GetCustomerRow:
		id = sp.ID
		userID = sp.UserID
		balance = sp.Balance
		name = sp.Name
		email = sp.Email
		stripeID = sp.StripeID
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	case *sqlc.ListCustomersRow:
		id = sp.ID
		userID = sp.UserID
		balance = sp.Balance
		name = sp.Name
		email = sp.Email
		stripeID = sp.StripeID
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	c.ID = id
	c.UserID = userID
	c.Balance = balance
	c.Name = name
	c.Email = email
	c.StripeID = stripeID
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt

	return c
}
