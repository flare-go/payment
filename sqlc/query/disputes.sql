-- name: CreateDispute :exec
INSERT INTO disputes (
    id, charge_id, amount, currency, status, reason, evidence_due_by, created_at, updated_at
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9
         );

-- name: GetDisputeByID :one
SELECT id, charge_id, amount, currency, status, reason, evidence_due_by, created_at, updated_at
FROM disputes
WHERE id = $1;

-- name: UpdateDispute :exec
UPDATE disputes
SET charge_id = $2, amount = $3, currency = $4, status = $5, reason = $6, evidence_due_by = $7, updated_at = $8
WHERE id = $1;

-- name: CloseDispute :exec
UPDATE disputes
SET status = 'closed', updated_at = $2
WHERE id = $1;

-- name: UpsertDispute :exec
INSERT INTO disputes (
    id,
    charge_id,
    amount,
    currency,
    status,
    reason,
    evidence_due_by
) VALUES (
             $1, $2, $3, $4, $5, $6, $7
         )
ON CONFLICT (id)
    DO UPDATE SET
                  charge_id = EXCLUDED.charge_id,
                  amount = EXCLUDED.amount,
                  currency = EXCLUDED.currency,
                  status = EXCLUDED.status,
                  reason = EXCLUDED.reason,
                  evidence_due_by = EXCLUDED.evidence_due_by,
                  updated_at = NOW();

-- name: DeleteDispute :exec
DELETE FROM disputes WHERE id = $1;