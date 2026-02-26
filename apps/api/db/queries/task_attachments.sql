-- name: CreateTaskAttachment :one
INSERT INTO task_attachments (
    task_id,
    uploaded_by,
    file_name,
    file_url,
    file_size,
    file_type
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetTaskAttachmentByID :one
SELECT *
FROM task_attachments
WHERE id = $1
LIMIT 1;

-- name: ListTaskAttachmentsByTaskID :many
SELECT *
FROM task_attachments
WHERE task_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteTaskAttachment :exec
DELETE FROM task_attachments
WHERE id = $1;

