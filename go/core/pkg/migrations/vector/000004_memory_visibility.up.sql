-- Memory: add visibility and shared_with
ALTER TABLE memory
    ADD COLUMN IF NOT EXISTS visibility  TEXT   NOT NULL DEFAULT 'private',
    ADD COLUMN IF NOT EXISTS shared_with TEXT[] NOT NULL DEFAULT '{}';
