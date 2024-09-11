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
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PartialCustomer struct {
	ID        string
	Email     *string
	Balance   *int64
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func NewCustomer() *Customer {
	return &Customer{}
}

func (c *Customer) ConvertFromSQLCCustomer(sqlcCustomer any) *Customer {

	var balance int64
	var id, name, email string
	var createdAt, updatedAt time.Time

	switch sp := sqlcCustomer.(type) {
	case *sqlc.Customer:
		id = sp.ID
		balance = sp.Balance
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	case *sqlc.GetCustomerRow:
		id = sp.ID
		balance = sp.Balance
		name = sp.Name
		email = sp.Email
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	case *sqlc.ListCustomersRow:
		id = sp.ID
		balance = sp.Balance
		name = sp.Name
		email = sp.UserEmail
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	c.ID = id
	c.Balance = balance
	c.Name = name
	c.Email = email
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt

	return c
}
