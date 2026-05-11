CREATE TABLE IF NOT EXISTS agent_comment (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_agent_comment_agent_id ON agent_comment(agent_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_agent_comment_created ON agent_comment(agent_id, created_at DESC) WHERE deleted_at IS NULL;
