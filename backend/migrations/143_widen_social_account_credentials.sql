-- Widen social account delivery credential columns for longer business values.
-- Social account credentials are stored and returned as account-pool delivery
-- data; this migration only adjusts column capacity.
ALTER TABLE social_accounts
    ALTER COLUMN password TYPE VARCHAR(1024),
    ALTER COLUMN email_password TYPE VARCHAR(1024);

COMMENT ON COLUMN social_accounts.password IS 'Social account password delivery field managed by the application service layer.';
COMMENT ON COLUMN social_accounts.email_password IS 'Social account email password delivery field managed by the application service layer.';
