ALTER TABLE redeem_codes
  ADD COLUMN IF NOT EXISTS plan_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_redeem_codes_plan_id
  ON redeem_codes(plan_id);
