-- Agent: add visibility enum and shared_with user list
ALTER TABLE agent
    ADD COLUMN IF NOT EXISTS visibility  TEXT   NOT NULL DEFAULT 'private',
    ADD COLUMN IF NOT EXISTS shared_with TEXT[] NOT NULL DEFAULT '{}';

-- Backfill from legacy private_mode: false -> 'public', true -> 'private'
UPDATE agent SET visibility = CASE WHEN private_mode THEN 'private' ELSE 'public' END;

-- Session: add visibility and shared_with
ALTER TABLE session
    ADD COLUMN IF NOT EXISTS visibility  TEXT   NOT NULL DEFAULT 'private',
    ADD COLUMN IF NOT EXISTS shared_with TEXT[] NOT NULL DEFAULT '{}';
