ALTER TABLE subscription_plans
  ADD COLUMN IF NOT EXISTS platform VARCHAR(50) NOT NULL DEFAULT 'social';

ALTER TABLE subscription_plans
  ADD COLUMN IF NOT EXISTS daily_limit_usd DECIMAL(20,8);

ALTER TABLE subscription_plans
  ADD COLUMN IF NOT EXISTS weekly_limit_usd DECIMAL(20,8);

ALTER TABLE subscription_plans
  ADD COLUMN IF NOT EXISTS monthly_limit_usd DECIMAL(20,8);

UPDATE subscription_plans sp
SET platform = COALESCE(NULLIF(g.platform, ''), 'social')
FROM groups g
WHERE g.id = sp.group_id
  AND (sp.platform IS NULL OR sp.platform = '' OR sp.platform = 'social');

ALTER TABLE user_subscriptions
  ADD COLUMN IF NOT EXISTS plan_id BIGINT;

ALTER TABLE user_subscriptions
  ADD COLUMN IF NOT EXISTS plan_name VARCHAR(100) NOT NULL DEFAULT '';

ALTER TABLE user_subscriptions
  ADD COLUMN IF NOT EXISTS plan_platform VARCHAR(50) NOT NULL DEFAULT '';

ALTER TABLE user_subscriptions
  ADD COLUMN IF NOT EXISTS daily_limit_usd DECIMAL(20,8);

ALTER TABLE user_subscriptions
  ADD COLUMN IF NOT EXISTS weekly_limit_usd DECIMAL(20,8);

ALTER TABLE user_subscriptions
  ADD COLUMN IF NOT EXISTS monthly_limit_usd DECIMAL(20,8);

UPDATE user_subscriptions us
SET
  plan_platform = COALESCE(NULLIF(us.plan_platform, ''), NULLIF(g.platform, ''), 'social'),
  daily_limit_usd = COALESCE(us.daily_limit_usd, g.daily_limit_usd),
  weekly_limit_usd = COALESCE(us.weekly_limit_usd, g.weekly_limit_usd),
  monthly_limit_usd = COALESCE(us.monthly_limit_usd, g.monthly_limit_usd)
FROM groups g
WHERE g.id = us.group_id;

CREATE INDEX IF NOT EXISTS idx_subscription_plans_platform
  ON subscription_plans(platform);

CREATE INDEX IF NOT EXISTS idx_user_subscriptions_plan_id
  ON user_subscriptions(plan_id);
