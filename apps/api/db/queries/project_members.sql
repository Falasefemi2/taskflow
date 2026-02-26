-- name: AddProjectMember :one
INSERT INTO project_members (
    project_id,
    user_id,
    role
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetProjectMember :one
SELECT *
FROM project_members
WHERE project_id = $1
  AND user_id = $2
LIMIT 1;

-- name: ListProjectMembers :many
SELECT *
FROM project_members
WHERE project_id = $1
ORDER BY joined_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateProjectMemberRole :one
UPDATE project_members
SET role = $3
WHERE project_id = $1
  AND user_id = $2
RETURNING *;

-- name: RemoveProjectMember :exec
DELETE FROM project_members
WHERE project_id = $1
  AND user_id = $2;

