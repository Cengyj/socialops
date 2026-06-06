ALTER TABLE social_accounts
  ADD COLUMN IF NOT EXISTS user_workbench_deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_social_accounts_assigned_workbench_deleted
  ON social_accounts(assigned_user_id, user_workbench_deleted_at);
