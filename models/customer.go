package models

import (
	"time"

	"goflare.io/payment/sqlc"
)

// Customer 代表系統中的客戶
// Customer represents a customer in the system
type Customer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	UserID    uint64    `json:"user_id"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PartialCustomer struct {
	ID        string
	UserID    *uint64
	Email     *string
	Name      *string
	Phone     *string
	Balance   *int64
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func NewCustomer() *Customer {
	return &Customer{}
}

func (c *Customer) ConvertFromSQLCCustomer(sqlcCustomer any) *Customer {

	var userID uint64
	var balance int64
	var id, name, email string
	var createdAt, updatedAt time.Time

	switch sp := sqlcCustomer.(type) {
	case *sqlc.Customer:
		id = sp.ID
		userID = sp.UserID
		balance = sp.Balance
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	case *sqlc.GetCustomerRow:
		id = sp.ID
		userID = sp.UserID
		balance = sp.Balance
		name = sp.Name
		email = sp.Email
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	case *sqlc.ListCustomersRow:
		id = sp.ID
		userID = sp.UserID
		balance = sp.Balance
		name = sp.Name
		email = sp.Email
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
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt

	return c
}
