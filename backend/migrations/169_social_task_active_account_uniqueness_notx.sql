-- Enforce one pending/running social task per account online.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_social_task_logs_one_active_per_account
    ON social_task_logs (social_account_id)
    WHERE status IN ('pending', 'running');
