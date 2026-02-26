-- name: CreateTask :one
INSERT INTO tasks (
    project_id,
    title,
    description,
    status,
    priority,
    assignee_id,
    reporter_id,
    due_date,
    completed_at,
    position,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: GetTaskByID :one
SELECT *
FROM tasks
WHERE id = $1
LIMIT 1;

-- name: ListTasksByProjectID :many
SELECT *
FROM tasks
WHERE project_id = $1
ORDER BY position ASC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateTask :one
UPDATE tasks
SET
    title = $2,
    description = $3,
    status = $4,
    priority = $5,
    assignee_id = $6,
    reporter_id = $7,
    due_date = $8,
    completed_at = $9,
    position = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1;

