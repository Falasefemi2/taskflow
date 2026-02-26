-- name: CreateTaskLabel :one
INSERT INTO task_labels (
    task_id,
    name,
    color
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetTaskLabelByID :one
SELECT *
FROM task_labels
WHERE id = $1
LIMIT 1;

-- name: ListTaskLabelsByTaskID :many
SELECT *
FROM task_labels
WHERE task_id = $1
ORDER BY name ASC
LIMIT $2 OFFSET $3;

-- name: UpdateTaskLabel :one
UPDATE task_labels
SET
    name = $2,
    color = $3
WHERE id = $1
RETURNING *;

-- name: DeleteTaskLabel :exec
DELETE FROM task_labels
WHERE id = $1;

