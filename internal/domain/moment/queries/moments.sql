-- name: GetPostByID :one
SELECT id, created_at, updated_at, deleted_at, user_id, content, status
FROM posts 
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreatePost :one
INSERT INTO posts (
    id, created_at, updated_at, user_id, content, status
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('user_id'), sqlc.narg('content'), sqlc.narg('status')
)
RETURNING id, created_at, updated_at, deleted_at, user_id, content, status;

-- name: GetPosts :many
SELECT id, created_at, updated_at, deleted_at, user_id, content, status
FROM posts 
WHERE deleted_at IS NULL AND status = $1
ORDER BY created_at DESC 
LIMIT $2 OFFSET $3;

-- name: CountPosts :one
SELECT COUNT(*) 
FROM posts 
WHERE deleted_at IS NULL AND status = $1;

-- name: UpdatePostStatus :exec
UPDATE posts 
SET status = $1, updated_at = $2
WHERE id = $3 AND deleted_at IS NULL;

-- name: GetCommentByID :one
SELECT id, created_at, updated_at, deleted_at, user_id, post_id, content
FROM comments 
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateComment :one
INSERT INTO comments (
    id, created_at, updated_at, user_id, post_id, content
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('user_id'), sqlc.narg('post_id'), sqlc.narg('content')
)
RETURNING id, created_at, updated_at, deleted_at, user_id, post_id, content;

-- name: GetCommentsByPostID :many
SELECT id, created_at, updated_at, deleted_at, user_id, post_id, content
FROM comments 
WHERE deleted_at IS NULL AND post_id = $1
ORDER BY created_at ASC 
LIMIT $2 OFFSET $3;

-- name: CountCommentsByPostID :one
SELECT COUNT(*) 
FROM comments 
WHERE deleted_at IS NULL AND post_id = $1;

-- name: CreateLike :one
INSERT INTO likes (
    id, created_at, updated_at, user_id, target_id, target_type
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('user_id'), sqlc.narg('target_id'), sqlc.narg('target_type')
)
RETURNING id, created_at, updated_at, deleted_at, user_id, target_id, target_type;

-- name: DeleteLike :exec
UPDATE likes 
SET deleted_at = $1, updated_at = $2
WHERE user_id = $3 AND target_id = $4 AND target_type = $5 AND deleted_at IS NULL;
