DO $$
BEGIN
    EXECUTE 'DROP INDEX IF EXISTS idx_social_accounts_'
        || 'source';
END $$;

ALTER TABLE social_accounts
    DROP COLUMN IF EXISTS source;
