-- Widen social account credential columns for application-level encrypted values.
-- Existing plaintext rows are encrypted by SocialAccountService during startup
-- only when stable credential encryption is explicitly configured.
ALTER TABLE social_accounts
    ALTER COLUMN password TYPE VARCHAR(1024),
    ALTER COLUMN email_password TYPE VARCHAR(1024);

COMMENT ON COLUMN social_accounts.password IS 'Encrypted social account password managed by the application service layer.';
COMMENT ON COLUMN social_accounts.email_password IS 'Encrypted social account email password managed by the application service layer.';
