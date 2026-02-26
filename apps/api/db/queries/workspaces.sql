-- name: CreateWorkspace :one
INSERT INTO workspaces (
    name,
    slug,
    description,
    owner_id,
    status
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetWorkspaceByID :one
SELECT *
FROM workspaces
WHERE id = $1
LIMIT 1;

-- name: GetWorkspaceBySlug :one
SELECT *
FROM workspaces
WHERE slug = $1
LIMIT 1;

-- name: ListWorkspacesByOwnerID :many
SELECT *
FROM workspaces
WHERE owner_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET
    name = $2,
    slug = $3,
    description = $4,
    owner_id = $5,
    status = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;

