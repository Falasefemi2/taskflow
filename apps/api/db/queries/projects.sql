-- name: CreateProject :one
INSERT INTO projects (
    workspace_id,
    name,
    description,
    status,
    color,
    owner_id,
    start_date,
    due_date,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetProjectByID :one
SELECT *
FROM projects
WHERE id = $1
LIMIT 1;

-- name: ListProjectsByWorkspaceID :many
SELECT *
FROM projects
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateProject :one
UPDATE projects
SET
    name = $2,
    description = $3,
    status = $4,
    color = $5,
    owner_id = $6,
    start_date = $7,
    due_date = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;

