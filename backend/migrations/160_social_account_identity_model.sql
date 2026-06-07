-- Rebuild SocialOps account identity around one explicit business key.
-- This is intentionally contractive for identity fields: the product no
-- longer keeps legacy account_id/bound_ip aliases. auth_cookie is retained as
-- an explicit account credential field and does not participate in uniqueness.

ALTER TABLE social_accounts
    ADD COLUMN IF NOT EXISTS identity_kind VARCHAR(30),
    ADD COLUMN IF NOT EXISTS identity_key VARCHAR(100);

UPDATE social_accounts
SET
    platform_key = CASE
        WHEN BTRIM(REGEXP_REPLACE(LOWER(BTRIM(platform)), '[-/_[:space:]]+', '_', 'g'), '_') IN ('x', 'twitter', 'x_twitter', 'twitter_x') THEN 'x_twitter'
        ELSE BTRIM(REGEXP_REPLACE(LOWER(BTRIM(platform)), '[-/_[:space:]]+', '_', 'g'), '_')
    END,
    name_key = REGEXP_REPLACE(LOWER(BTRIM(name)), '^@+', ''),
    identity_kind = 'username',
    identity_key = REGEXP_REPLACE(LOWER(BTRIM(name)), '^@+', '')
WHERE identity_kind IS NULL
   OR identity_key IS NULL
   OR platform_key IS NULL
   OR name_key IS NULL;

ALTER TABLE social_accounts
    ALTER COLUMN identity_kind SET NOT NULL,
    ALTER COLUMN identity_kind SET DEFAULT 'username',
    ALTER COLUMN identity_key SET NOT NULL;

DROP INDEX IF EXISTS idx_social_accounts_platform_name_key_unique;
DROP INDEX IF EXISTS socialaccount_platform_key_name_key;
DROP INDEX IF EXISTS idx_social_accounts_business_identity_unique;
DROP INDEX IF EXISTS idx_social_accounts_platform_identity;

CREATE UNIQUE INDEX IF NOT EXISTS idx_social_accounts_platform_name_key_unique
    ON social_accounts(platform_key, name_key)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_social_accounts_platform_identity
    ON social_accounts(platform_key, name_key)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN social_accounts.identity_kind IS 'Business identity type. SocialOps account identity is username-based.';
COMMENT ON COLUMN social_accounts.identity_key IS 'Normalized username identity key mirrored from name_key.';
COMMENT ON COLUMN social_accounts.platform_key IS 'Normalized platform key for SocialOps account uniqueness.';
COMMENT ON COLUMN social_accounts.name_key IS 'Normalized username key for SocialOps account uniqueness.';
COMMENT ON COLUMN social_accounts.platform_user_id IS 'Platform user ID / rest_id metadata. It is never used for SocialOps identity or duplicate checks.';
COMMENT ON COLUMN social_accounts.auth_cookie IS 'Platform auth cookie captured for account operations.';
COMMENT ON COLUMN social_accounts.execution_auth IS 'Platform execution authentication JSON/base64 JSON used for social task execution.';
COMMENT ON COLUMN social_accounts.default_proxy_snapshot IS 'Default execution proxy snapshot JSON.';

ALTER TABLE social_accounts
    DROP COLUMN IF EXISTS account_id,
    DROP COLUMN IF EXISTS bound_ip;
