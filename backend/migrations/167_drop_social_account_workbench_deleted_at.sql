-- /accounts deletion is a physical account-pool deletion, not a workbench
-- visibility flag or soft-delete marker. Remove the legacy columns after
-- physically deleting any remaining rows that still carry those markers.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'social_accounts'
      AND column_name = 'user_workbench_deleted_at'
  ) THEN
    EXECUTE '
      DELETE FROM social_task_logs
      WHERE social_account_id IN (
        SELECT id
        FROM social_accounts
        WHERE user_workbench_deleted_at IS NOT NULL
      )
    ';

    EXECUTE '
      UPDATE social_ips
      SET bound_social_account_id = NULL
      WHERE bound_social_account_id IN (
        SELECT id
        FROM social_accounts
        WHERE user_workbench_deleted_at IS NOT NULL
      )
    ';

    EXECUTE '
      DELETE FROM social_accounts
      WHERE user_workbench_deleted_at IS NOT NULL
    ';
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'social_accounts'
      AND column_name = 'deleted_at'
  ) THEN
    EXECUTE '
      DELETE FROM social_task_logs
      WHERE social_account_id IN (
        SELECT id
        FROM social_accounts
        WHERE deleted_at IS NOT NULL
      )
    ';

    EXECUTE '
      UPDATE social_ips
      SET bound_social_account_id = NULL
      WHERE bound_social_account_id IN (
        SELECT id
        FROM social_accounts
        WHERE deleted_at IS NOT NULL
      )
    ';

    EXECUTE '
      DELETE FROM social_accounts
      WHERE deleted_at IS NOT NULL
    ';
  END IF;
END $$;

DROP INDEX IF EXISTS idx_social_accounts_assigned_workbench_deleted;
DROP INDEX IF EXISTS socialaccount_assigned_user_id_user_workbench_deleted_at;
DROP INDEX IF EXISTS idx_social_accounts_deleted_at;
DROP INDEX IF EXISTS socialaccount_deleted_at;
DROP INDEX IF EXISTS idx_social_accounts_platform_name_key_unique;
DROP INDEX IF EXISTS socialaccount_platform_key_name_key;
DROP INDEX IF EXISTS idx_social_accounts_platform_identity;
DROP INDEX IF EXISTS idx_social_accounts_platform_user_id;
DROP INDEX IF EXISTS socialaccount_platform_user_id;

ALTER TABLE social_accounts
  DROP COLUMN IF EXISTS user_workbench_deleted_at,
  DROP COLUMN IF EXISTS deleted_at;

CREATE UNIQUE INDEX IF NOT EXISTS idx_social_accounts_platform_name_key_unique
  ON social_accounts(platform_key, name_key);

CREATE INDEX IF NOT EXISTS idx_social_accounts_platform_user_id
  ON social_accounts(platform_user_id);
