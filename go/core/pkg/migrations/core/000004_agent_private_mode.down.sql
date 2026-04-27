ALTER TABLE agent
    DROP COLUMN IF EXISTS user_id,
    DROP COLUMN IF EXISTS private_mode;
