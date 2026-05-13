-- name: GetAgent :one
SELECT * FROM agent
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListAgents :many
SELECT * FROM agent
WHERE deleted_at IS NULL
ORDER BY created_at ASC;

-- name: ListAgentsVisible :many
SELECT * FROM agent
WHERE deleted_at IS NULL AND (
    visibility = 'public'
    OR user_id = @user_id
    OR (visibility = 'shared' AND @user_id = ANY(shared_with))
)
ORDER BY created_at ASC;

-- name: UpsertAgent :exec
INSERT INTO agent (id, type, workload_type, config, user_id, private_mode, visibility, shared_with, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    type          = EXCLUDED.type,
    workload_type = EXCLUDED.workload_type,
    config        = EXCLUDED.config,
    user_id       = EXCLUDED.user_id,
    private_mode  = EXCLUDED.private_mode,
    visibility    = EXCLUDED.visibility,
    shared_with   = EXCLUDED.shared_with,
    updated_at    = NOW(),
    deleted_at    = NULL;

-- name: UpdateAgentVisibility :exec
UPDATE agent SET visibility = $2, shared_with = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $4 AND deleted_at IS NULL;

-- name: SoftDeleteAgent :exec
UPDATE agent SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL;
