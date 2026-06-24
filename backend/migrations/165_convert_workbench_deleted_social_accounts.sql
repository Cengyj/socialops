-- Accounts removed from /accounts used to be hidden only from the user's
-- workbench. SocialOps now treats that action as an account-pool deletion, so
-- historical rows must leave the total account pool as well.
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
