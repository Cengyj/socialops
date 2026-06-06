-- Restore auth_cookie as an explicit account credential field.
-- It is intentionally not part of the social account business identity key.

ALTER TABLE social_accounts
    ADD COLUMN IF NOT EXISTS auth_cookie TEXT;

COMMENT ON COLUMN social_accounts.auth_cookie IS 'Platform auth cookie captured for account operations.';
