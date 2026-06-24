-- Align the schema comment with the runtime contract: execution_auth is stored
-- as encrypted execution credentials, not as deliverable plaintext JSON.
COMMENT ON COLUMN social_accounts.execution_auth IS 'Encrypted platform execution authentication used for social task execution.';
