-- name: CreateRefund :exec
INSERT INTO refunds (
    id,
    charge_id,
    amount,
    status,
    reason
) VALUES (
             $1, $2, $3, $4, $5
         );

-- name: GetRefund :one
SELECT
    id,
    charge_id,
    amount,
    status,
    reason,
    created_at,
    updated_at
FROM refunds
WHERE id = $1;

-- name: UpdateRefund :exec
UPDATE refunds
SET
    status = $2,
    reason = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: ListRefunds :many
SELECT
    id,
    charge_id,
    amount,
    status,
    reason,
    created_at,
    updated_at
FROM refunds
WHERE charge_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListByChargeID :many
SELECT
    id,
    charge_id,
    amount,
    status,
    reason,
    created_at,
    updated_at
FROM refunds
WHERE charge_id = $1
ORDER BY created_at DESC;

-- name: DeleteRefund :exec
DELETE FROM refunds
WHERE id = $1;


-- name: UpsertRefund :exec
INSERT INTO refunds (
    id, charge_id, amount, status, reason
) VALUES (
             $1, $2, $3, $4, $5
         )
ON CONFLICT (id) DO UPDATE SET
                                      charge_id = EXCLUDED.charge_id,
                                      amount = EXCLUDED.amount,
                                      status = EXCLUDED.status,
                                      reason = EXCLUDED.reason,
                                      updated_at = NOW();


