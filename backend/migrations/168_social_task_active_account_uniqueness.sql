-- Prepare social_task_logs for one active task per social account.
--
-- Keep the newest pending/running task for each account and fail-close older
-- duplicates without charging. This makes the following partial unique index
-- safe to build while preserving the most recent in-flight work.
WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY social_account_id
            ORDER BY created_at DESC, id DESC
        ) AS rn
    FROM social_task_logs
    WHERE status IN ('pending', 'running')
)
UPDATE social_task_logs stl
SET status = 'failed',
    result_message = COALESCE(NULLIF(result_message, ''), 'duplicate active task failed closed before active-task uniqueness rollout'),
    charged_amount = 0,
    charge_status = 'not_charged',
    charge_source = NULL,
    billing_request_id = NULL,
    executed_at = COALESCE(executed_at, NOW())
FROM ranked r
WHERE stl.id = r.id
  AND r.rn > 1;
