// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: promotion_code.sql

package sqlc

import (
	"context"
)

const deletePromotionCodes = `-- name: DeletePromotionCodes :exec
DELETE FROM promotion_codes WHERE id = $1
`

func (q *Queries) DeletePromotionCodes(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx, deletePromotionCodes, id)
	return err
}
