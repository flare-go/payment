// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: quote.sql

package sqlc

import (
	"context"
)

const deleteQuote = `-- name: DeleteQuote :exec
DELETE FROM quotes WHERE id = $1
`

func (q *Queries) DeleteQuote(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx, deleteQuote, id)
	return err
}
