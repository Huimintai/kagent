-- name: GetAgentSessionStats :many
-- Returns top agents ranked by distinct user count with session/message counts
SELECT
    s.agent_id,
    COUNT(DISTINCT s.user_id) as user_count,
    COUNT(DISTINCT s.id) as session_count,
    COALESCE(SUM(ec.event_count), 0)::bigint as message_count,
    MAX(s.created_at) as last_active_at
FROM session s
INNER JOIN agent a ON a.id = s.agent_id AND a.deleted_at IS NULL
LEFT JOIN (
    SELECT session_id, COUNT(*) as event_count
    FROM event
    WHERE deleted_at IS NULL
    GROUP BY session_id
) ec ON s.id = ec.session_id
WHERE s.deleted_at IS NULL
  AND s.agent_id IS NOT NULL
  AND (s.source IS NULL OR s.source != 'agent')
GROUP BY s.agent_id
ORDER BY user_count DESC, session_count DESC
LIMIT $1;

-- name: GetToolServerStats :many
-- Returns tool servers ranked by number of agents that reference their tools
SELECT
    ts.name,
    ts.group_kind,
    COUNT(DISTINCT a.id) as agent_count,
    ts.last_connected
FROM toolserver ts
INNER JOIN tool t ON ts.name = t.server_name AND ts.group_kind = t.group_kind AND t.deleted_at IS NULL
INNER JOIN agent a ON a.deleted_at IS NULL
    AND (
        EXISTS (
            SELECT 1 FROM jsonb_array_elements(a.config::jsonb->'http_tools') AS ht
            WHERE ht->'tools' ? t.id
        )
        OR EXISTS (
            SELECT 1 FROM jsonb_array_elements(a.config::jsonb->'sse_tools') AS st
            WHERE st->'tools' ? t.id
        )
    )
WHERE ts.deleted_at IS NULL
GROUP BY ts.name, ts.group_kind, ts.last_connected
HAVING COUNT(DISTINCT a.id) > 0
ORDER BY agent_count DESC, ts.last_connected DESC NULLS LAST
LIMIT $1;

-- name: GetPlatformSummary :one
-- Returns total counts for the platform overview
SELECT
    (SELECT COUNT(*) FROM agent WHERE deleted_at IS NULL) as total_agents,
    (SELECT COUNT(*) FROM session WHERE deleted_at IS NULL AND (source IS NULL OR source != 'agent')) as total_sessions,
    (SELECT COUNT(*) FROM toolserver WHERE deleted_at IS NULL) as total_tool_servers,
    (SELECT COUNT(*) FROM session WHERE deleted_at IS NULL AND created_at >= NOW() - INTERVAL '24 hours' AND (source IS NULL OR source != 'agent')) as sessions_today;
