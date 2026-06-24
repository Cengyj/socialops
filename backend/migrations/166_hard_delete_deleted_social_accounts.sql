-- Ensure every historical account-pool removal is a physical deletion. This
-- follows up on earlier workbench-removal and soft-delete semantics so already
-- applied databases do not keep hidden social account rows in the total pool.
DELETE FROM social_task_logs
WHERE social_account_id IN (
  SELECT id
  FROM social_accounts
  WHERE user_workbench_deleted_at IS NOT NULL
     OR deleted_at IS NOT NULL
);

UPDATE social_ips
SET bound_social_account_id = NULL
WHERE bound_social_account_id IN (
  SELECT id
  FROM social_accounts
  WHERE user_workbench_deleted_at IS NOT NULL
     OR deleted_at IS NOT NULL
);

DELETE FROM social_accounts
WHERE user_workbench_deleted_at IS NOT NULL
   OR deleted_at IS NOT NULL;
