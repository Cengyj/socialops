-- Build the SocialOps task usage ledger idempotency guarantee online.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_social_task_request_unique
    ON usage_logs (request_id)
    WHERE request_id IS NOT NULL
      AND api_key_id IS NULL
      AND model = 'social-action';
