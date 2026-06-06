-- Enforce one total-pool account per normalized platform username.
-- Forward-only: adds canonical keys instead of rewriting historical name/platform.

ALTER TABLE social_accounts
    ADD COLUMN IF NOT EXISTS platform_key VARCHAR(50),
    ADD COLUMN IF NOT EXISTS name_key VARCHAR(100);

UPDATE social_accounts
SET
    platform_key = LOWER(BTRIM(platform)),
    name_key = REGEXP_REPLACE(LOWER(BTRIM(name)), '^@+', '')
WHERE platform_key IS NULL OR name_key IS NULL;

ALTER TABLE social_accounts
    ALTER COLUMN platform_key SET NOT NULL,
    ALTER COLUMN name_key SET NOT NULL;

COMMENT ON COLUMN social_accounts.platform_key IS 'Normalized platform key for SocialOps total-pool uniqueness.';
COMMENT ON COLUMN social_accounts.name_key IS 'Normalized username key for SocialOps total-pool uniqueness.';
COMMENT ON COLUMN social_task_logs.charge_source IS 'Final charge source: subscription / wallet / mixed.';

CREATE UNIQUE INDEX IF NOT EXISTS idx_social_accounts_platform_name_key_unique
    ON social_accounts(platform_key, name_key)
    WHERE deleted_at IS NULL;
