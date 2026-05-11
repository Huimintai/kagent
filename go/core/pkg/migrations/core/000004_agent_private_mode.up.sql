ALTER TABLE agent
    ADD COLUMN IF NOT EXISTS user_id      TEXT    NOT NULL DEFAULT 'admin@kagent.dev',
    ADD COLUMN IF NOT EXISTS private_mode BOOLEAN NOT NULL DEFAULT false;
