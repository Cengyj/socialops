-- Clarify SocialOps account field model without dropping credential data.
-- account_id/bound_ip remain as migration sources until the identity migration
-- removes them. auth_cookie remains an explicit account credential field.

ALTER TABLE social_accounts
    ADD COLUMN IF NOT EXISTS platform_user_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS two_factor VARCHAR(1024),
    ADD COLUMN IF NOT EXISTS backup_code VARCHAR(1024),
    ADD COLUMN IF NOT EXISTS email_client_id VARCHAR(1024),
    ADD COLUMN IF NOT EXISTS email_token TEXT,
    ADD COLUMN IF NOT EXISTS registration_ip VARCHAR(255),
    ADD COLUMN IF NOT EXISTS execution_auth TEXT,
    ADD COLUMN IF NOT EXISTS default_proxy_snapshot TEXT;

UPDATE social_accounts
SET platform_user_id = account_id
WHERE platform_user_id IS NULL
  AND account_id IS NOT NULL
  AND BTRIM(account_id) <> '';

UPDATE social_accounts
SET default_proxy_snapshot = bound_ip
WHERE default_proxy_snapshot IS NULL
  AND bound_ip IS NOT NULL
  AND BTRIM(bound_ip) <> '';

COMMENT ON COLUMN social_accounts.platform_user_id IS 'Platform user ID / rest_id, e.g. Twitter/X numeric user ID.';
COMMENT ON COLUMN social_accounts.account_id IS 'Legacy platform account ID; use platform_user_id.';
COMMENT ON COLUMN social_accounts.two_factor IS 'Account two-factor secret or code.';
COMMENT ON COLUMN social_accounts.backup_code IS 'Account backup/recovery code.';
COMMENT ON COLUMN social_accounts.email_client_id IS 'Email OAuth/client identifier used for account support workflows.';
COMMENT ON COLUMN social_accounts.email_token IS 'Email token used for account support workflows.';
COMMENT ON COLUMN social_accounts.registration_ip IS 'IP used when the social account was registered.';
COMMENT ON COLUMN social_accounts.execution_auth IS 'Platform execution authentication JSON/base64 JSON.';
COMMENT ON COLUMN social_accounts.auth_cookie IS 'Platform auth cookie captured for account operations.';
COMMENT ON COLUMN social_accounts.default_proxy_snapshot IS 'Default execution proxy snapshot JSON.';
COMMENT ON COLUMN social_accounts.bound_ip IS 'Legacy default execution proxy snapshot; use default_proxy_snapshot.';

CREATE INDEX IF NOT EXISTS idx_social_accounts_platform_user_id
    ON social_accounts(platform_user_id)
    WHERE deleted_at IS NULL;
