-- name: CreateAgentComment :one
INSERT INTO agent_comment (id, agent_id, user_id, content, created_at)
VALUES ($1, $2, $3, $4, NOW())
RETURNING *;

-- name: ListAgentComments :many
SELECT * FROM agent_comment
WHERE agent_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2;

-- name: DeleteAgentComment :exec
UPDATE agent_comment SET deleted_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: CountAgentComments :one
SELECT COUNT(*) FROM agent_comment
WHERE agent_id = $1 AND deleted_at IS NULL;
