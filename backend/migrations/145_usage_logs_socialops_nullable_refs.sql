-- SocialOps usage projection rows are not AI gateway requests and may not have
-- an API key or legacy AI account. Keep generic usage_logs usable for social
-- task/dashboard projections without reintroducing AI account dependencies.
ALTER TABLE usage_logs ALTER COLUMN api_key_id DROP NOT NULL;
ALTER TABLE usage_logs ALTER COLUMN account_id DROP NOT NULL;
ALTER TABLE usage_logs DROP CONSTRAINT IF EXISTS usage_logs_account_id_fkey;
