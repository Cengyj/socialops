-- Store platform execution credentials on the total-pool social account.
ALTER TABLE social_accounts
    ADD COLUMN IF NOT EXISTS auth_cookie TEXT;

COMMENT ON COLUMN social_accounts.auth_cookie IS 'Platform auth cookie captured for account operations.';
