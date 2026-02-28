-- name: GetUserByID :one
SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at
FROM users 
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByMobile :one
SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at
FROM users 
WHERE mobile = $1 AND deleted_at IS NULL;

-- name: GetUserByUsername :one
SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at
FROM users 
WHERE username = $1 AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (
    id, created_at, updated_at, username, password, email, mobile, 
    nickname, avatar_url, role, is_member, member_expire_at, status, 
    banned_until, token, token_expire_at
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('username'), sqlc.narg('password'), sqlc.narg('email'), 
    sqlc.narg('mobile'), sqlc.narg('nickname'), sqlc.narg('avatar_url'), 
    sqlc.narg('role'), sqlc.narg('is_member'), sqlc.narg('member_expire_at'), 
    sqlc.narg('status'), sqlc.narg('banned_until'), sqlc.narg('token'), 
    sqlc.narg('token_expire_at')
)
RETURNING id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at;

-- name: UpdateUser :exec
UPDATE users SET 
    updated_at = $1, username = $2, password = $3, 
    email = $4, mobile = $5, nickname = $6, 
    avatar_url = $7, role = $8, is_member = $9, 
    member_expire_at = $10, status = $11, 
    banned_until = $12, token = $13, token_expire_at = $14
WHERE id = $15 AND deleted_at IS NULL;

-- name: DeleteUser :exec
UPDATE users 
SET deleted_at = $1, updated_at = $2
WHERE id = $3 AND deleted_at IS NULL;

-- name: GetUsersList :many
SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at
FROM users 
WHERE deleted_at IS NULL 
ORDER BY created_at DESC 
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) 
FROM users 
WHERE deleted_at IS NULL;

-- name: UpdateMemberStatus :exec
UPDATE users 
SET is_member = $1, member_expire_at = $2, updated_at = $3
WHERE id = $4 AND deleted_at IS NULL;
