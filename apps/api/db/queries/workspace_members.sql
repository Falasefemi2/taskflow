-- name: AddWorkspaceMember :one
INSERT INTO workspace_members (
    workspace_id,
    user_id,
    role
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetWorkspaceMember :one
SELECT *
FROM workspace_members
WHERE workspace_id = $1
  AND user_id = $2
LIMIT 1;

-- name: ListWorkspaceMembers :many
SELECT *
FROM workspace_members
WHERE workspace_id = $1
ORDER BY joined_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWorkspaceMemberRole :one
UPDATE workspace_members
SET role = $3
WHERE workspace_id = $1
  AND user_id = $2
RETURNING *;

-- name: RemoveWorkspaceMember :exec
DELETE FROM workspace_members
WHERE workspace_id = $1
  AND user_id = $2;

