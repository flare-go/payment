// repository/discount/repository.go

package discount

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
)

type Repository interface {
	Upsert(ctx context.Context, tx pgx.Tx, discount *models.PartialDiscount) error
	Delete(ctx context.Context, tx pgx.Tx, id string) error
}

type repository struct {
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}

func (r *repository) Upsert(ctx context.Context, tx pgx.Tx, discount *models.PartialDiscount) error {
	const query = `
    INSERT INTO discounts (id, customer_id, coupon_id, start, "end", created_at, updated_at)
    VALUES (@id, @customer_id, @coupon_id, @start, @"end", COALESCE(@created_at, NOW()), @updated_at)
    ON CONFLICT (id) DO UPDATE SET
        customer_id = COALESCE(@customer_id, discounts.customer_id),
        coupon_id = COALESCE(@coupon_id, discounts.coupon_id),
        start = COALESCE(@start, discounts.start),
        "end" = COALESCE(@"end", discounts."end"),
        updated_at = @updated_at
    WHERE discounts.id = @id
    `

	now := time.Now()
	args := pgx.NamedArgs{
		"id":          discount.ID,
		"customer_id": discount.CustomerID,
		"coupon_id":   discount.CouponID,
		"start":       discount.Start,
		"end":         discount.End,
		"created_at":  discount.CreatedAt,
		"updated_at":  now,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("failed to upsert discount: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	return sqlc.New(r.conn).WithTx(tx).DeleteDiscount(ctx, id)
}
