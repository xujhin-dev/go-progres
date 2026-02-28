-- name: GetCouponByID :one
SELECT id, created_at, updated_at, deleted_at, name, total, stock, amount, start_time, end_time
FROM coupons 
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateCoupon :one
INSERT INTO coupons (
    id, created_at, updated_at, name, total, stock, amount, start_time, end_time
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('name'), sqlc.narg('total'), sqlc.narg('stock'), 
    sqlc.narg('amount'), sqlc.narg('start_time'), sqlc.narg('end_time')
)
RETURNING id, created_at, updated_at, deleted_at, name, total, stock, amount, start_time, end_time;

-- name: DecreaseCouponStock :exec
UPDATE coupons 
SET stock = stock - 1, updated_at = $1
WHERE id = $2 AND deleted_at IS NULL AND stock > 0;

-- name: GetUserCoupon :one
SELECT id, created_at, updated_at, deleted_at, user_id, coupon_id, status
FROM user_coupons 
WHERE user_id = $1 AND coupon_id = $2 AND deleted_at IS NULL;

-- name: CreateUserCoupon :one
INSERT INTO user_coupons (
    id, created_at, updated_at, user_id, coupon_id, status
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('user_id'), sqlc.narg('coupon_id'), sqlc.narg('status')
)
RETURNING id, created_at, updated_at, deleted_at, user_id, coupon_id, status;

-- name: CountUserCoupons :one
SELECT COUNT(*) 
FROM user_coupons 
WHERE user_id = $1 AND coupon_id = $2 AND deleted_at IS NULL;
