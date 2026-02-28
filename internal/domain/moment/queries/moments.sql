-- name: GetPostByID :one
SELECT id, created_at, updated_at, deleted_at, user_id, content, media_urls, type, status
FROM posts 
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreatePost :one
INSERT INTO posts (
    id, created_at, updated_at, user_id, content, media_urls, type, status
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('user_id'), sqlc.narg('content'), sqlc.narg('media_urls'), 
    sqlc.narg('type'), sqlc.narg('status')
)
RETURNING id, created_at, updated_at, deleted_at, user_id, content, media_urls, type, status;

-- name: GetPosts :many
SELECT id, created_at, updated_at, deleted_at, user_id, content, media_urls, type, status
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
SELECT id, created_at, updated_at, deleted_at, user_id, post_id, content, parent_id, root_id, level
FROM comments 
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateComment :one
INSERT INTO comments (
    id, created_at, updated_at, user_id, post_id, content, parent_id, root_id, level
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('user_id'), sqlc.narg('post_id'), sqlc.narg('content'), 
    sqlc.narg('parent_id'), sqlc.narg('root_id'), sqlc.narg('level')
)
RETURNING id, created_at, updated_at, deleted_at, user_id, post_id, content, parent_id, root_id, level;

-- name: GetCommentsByPostID :many
SELECT id, created_at, updated_at, deleted_at, user_id, post_id, content, parent_id, root_id, level
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

-- name: GetLikeByUserAndTarget :one
SELECT id, created_at, updated_at, deleted_at, user_id, target_id, target_type
FROM likes 
WHERE user_id = $1 AND target_id = $2 AND target_type = $3 AND deleted_at IS NULL;

-- name: CreateTopic :one
INSERT INTO topics (
    id, created_at, updated_at, name
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('name')
)
RETURNING id, created_at, updated_at, deleted_at, name;

-- name: GetTopicByName :one
SELECT id, created_at, updated_at, deleted_at, name
FROM topics 
WHERE name = $1 AND deleted_at IS NULL;

-- name: GetAllTopics :many
SELECT id, created_at, updated_at, deleted_at, name
FROM topics 
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CreatePostTopic :exec
INSERT INTO post_topics (post_id, topic_id) VALUES ($1, $2);

-- name: DeletePostTopic :exec
DELETE FROM post_topics WHERE post_id = $1 AND topic_id = $2;
