-- name: CreateTaskComment :one
INSERT INTO task_comments (
    task_id,
    user_id,
    content
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetTaskCommentByID :one
SELECT *
FROM task_comments
WHERE id = $1
LIMIT 1;

-- name: ListTaskCommentsByTaskID :many
SELECT *
FROM task_comments
WHERE task_id = $1
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

-- name: UpdateTaskComment :one
UPDATE task_comments
SET
    content = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTaskComment :exec
DELETE FROM task_comments
WHERE id = $1;

