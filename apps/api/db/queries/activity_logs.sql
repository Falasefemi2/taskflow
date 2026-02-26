-- name: CreateActivityLog :one
INSERT INTO activity_logs (
    workspace_id,
    project_id,
    task_id,
    user_id,
    action,
    entity_type,
    entity_id,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetActivityLogByID :one
SELECT *
FROM activity_logs
WHERE id = $1
LIMIT 1;

-- name: ListActivityLogsByWorkspaceID :many
SELECT *
FROM activity_logs
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListActivityLogsByProjectID :many
SELECT *
FROM activity_logs
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListActivityLogsByTaskID :many
SELECT *
FROM activity_logs
WHERE task_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

