// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: customer.sql

package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createCustomer = `-- name: CreateCustomer :one
INSERT INTO customers (
    id,
    user_id,
    balance
) VALUES (
             $1, $2, $3
         )
RETURNING id, created_at, updated_at
`

type CreateCustomerParams struct {
	ID      string `json:"id"`
	UserID  int32  `json:"userId"`
	Balance int64  `json:"balance"`
}

type CreateCustomerRow struct {
	ID        string             `json:"id"`
	CreatedAt pgtype.Timestamptz `json:"createdAt"`
	UpdatedAt pgtype.Timestamptz `json:"updatedAt"`
}

func (q *Queries) CreateCustomer(ctx context.Context, arg CreateCustomerParams) (*CreateCustomerRow, error) {
	row := q.db.QueryRow(ctx, createCustomer, arg.ID, arg.UserID, arg.Balance)
	var i CreateCustomerRow
	err := row.Scan(&i.ID, &i.CreatedAt, &i.UpdatedAt)
	return &i, err
}

const deleteCustomer = `-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = $1
`

func (q *Queries) DeleteCustomer(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx, deleteCustomer, id)
	return err
}

const getCustomer = `-- name: GetCustomer :one
SELECT c.id, c.user_id, c.balance, c.created_at, c.updated_at,
       u.email, u.username as name
FROM customers c
         JOIN users u ON c.user_id = u.id
WHERE c.id = $1
`

type GetCustomerRow struct {
	ID        string             `json:"id"`
	UserID    int32              `json:"userId"`
	Balance   int64              `json:"balance"`
	CreatedAt pgtype.Timestamptz `json:"createdAt"`
	UpdatedAt pgtype.Timestamptz `json:"updatedAt"`
	Email     string             `json:"email"`
	Name      string             `json:"name"`
}

func (q *Queries) GetCustomer(ctx context.Context, dollar_1 *string) (*GetCustomerRow, error) {
	row := q.db.QueryRow(ctx, getCustomer, dollar_1)
	var i GetCustomerRow
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Balance,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.Name,
	)
	return &i, err
}

const listCustomers = `-- name: ListCustomers :many
SELECT c.id, c.user_id, c.balance, c.created_at, c.updated_at,
       u.email, u.username as name
FROM customers c
         JOIN users u ON c.user_id = u.id
ORDER BY c.created_at DESC
LIMIT $1 OFFSET $2
`

type ListCustomersParams struct {
	Column1 *int64 `json:"column1"`
	Column2 *int64 `json:"column2"`
}

type ListCustomersRow struct {
	ID        string             `json:"id"`
	UserID    int32              `json:"userId"`
	Balance   int64              `json:"balance"`
	CreatedAt pgtype.Timestamptz `json:"createdAt"`
	UpdatedAt pgtype.Timestamptz `json:"updatedAt"`
	Email     string             `json:"email"`
	Name      string             `json:"name"`
}

func (q *Queries) ListCustomers(ctx context.Context, arg ListCustomersParams) ([]*ListCustomersRow, error) {
	rows, err := q.db.Query(ctx, listCustomers, arg.Column1, arg.Column2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*ListCustomersRow{}
	for rows.Next() {
		var i ListCustomersRow
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.Balance,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Email,
			&i.Name,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateCustomer = `-- name: UpdateCustomer :exec
UPDATE customers
SET balance = $2,
    updated_at = NOW()
WHERE id = $1
`

type UpdateCustomerParams struct {
	ID      string `json:"id"`
	Balance int64  `json:"balance"`
}

func (q *Queries) UpdateCustomer(ctx context.Context, arg UpdateCustomerParams) error {
	_, err := q.db.Exec(ctx, updateCustomer, arg.ID, arg.Balance)
	return err
}

const updateCustomerBalance = `-- name: UpdateCustomerBalance :exec
UPDATE customers
SET balance = balance + $2,
    updated_at = NOW()
WHERE id = $1
`

type UpdateCustomerBalanceParams struct {
	ID      string `json:"id"`
	Balance int64  `json:"balance"`
}

func (q *Queries) UpdateCustomerBalance(ctx context.Context, arg UpdateCustomerBalanceParams) error {
	_, err := q.db.Exec(ctx, updateCustomerBalance, arg.ID, arg.Balance)
	return err
}

const upsertCustomer = `-- name: UpsertCustomer :exec
INSERT INTO customers (id, user_id, balance, updated_at)
VALUES ($1, $2, $4, $3)
ON CONFLICT (id) DO UPDATE SET
                               user_id = COALESCE($2, customers.user_id),
                               balance = COALESCE($4, customers.balance),
                               updated_at = $3
`

type UpsertCustomerParams struct {
	ID        string             `json:"id"`
	UserID    *int32             `json:"userId"`
	UpdatedAt pgtype.Timestamptz `json:"updatedAt"`
	Balance   *int64             `json:"balance"`
}

func (q *Queries) UpsertCustomer(ctx context.Context, arg UpsertCustomerParams) error {
	_, err := q.db.Exec(ctx, upsertCustomer,
		arg.ID,
		arg.UserID,
		arg.UpdatedAt,
		arg.Balance,
	)
	return err
}
