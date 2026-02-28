-- name: GetOrderByNo :one
SELECT id, created_at, updated_at, deleted_at, order_no, user_id, amount, status, channel, subject, extra_params, paid_at
FROM orders 
WHERE order_no = $1 AND deleted_at IS NULL;

-- name: CreateOrder :one
INSERT INTO orders (
    id, created_at, updated_at, order_no, user_id, amount, status, channel, subject, extra_params, paid_at
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('order_no'), sqlc.narg('user_id'), sqlc.narg('amount'), 
    sqlc.narg('status'), sqlc.narg('channel'), sqlc.narg('subject'), 
    sqlc.narg('extra_params'), sqlc.narg('paid_at')
)
RETURNING id, created_at, updated_at, deleted_at, order_no, user_id, amount, status, channel, subject, extra_params, paid_at;

-- name: UpdateOrderStatus :exec
UPDATE orders 
SET status = $1, paid_at = $2, updated_at = $3, extra_params = $4
WHERE order_no = $5 AND deleted_at IS NULL;
