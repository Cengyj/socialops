-- Add SocialOps task billing/proxy/idempotency fields without modifying the
-- already-applied SocialOps table migrations.

ALTER TABLE social_task_logs
    ADD COLUMN IF NOT EXISTS price DOUBLE PRECISION NOT NULL DEFAULT 0.1,
    ADD COLUMN IF NOT EXISTS charged_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS charge_status VARCHAR(30) NOT NULL DEFAULT 'not_charged',
    ADD COLUMN IF NOT EXISTS charge_source VARCHAR(30),
    ADD COLUMN IF NOT EXISTS proxy_id BIGINT REFERENCES social_ips(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS proxy_snapshot TEXT,
    ADD COLUMN IF NOT EXISTS billing_request_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

COMMENT ON COLUMN social_accounts.password IS 'Social account password stored and returned as provided for SocialOps account ownership/export workflows.';
COMMENT ON COLUMN social_accounts.email_password IS 'Social account email password stored and returned as provided for SocialOps account ownership/export workflows.';
COMMENT ON COLUMN social_task_logs.action IS 'Billable social action: login_check / dm / follow / tweet / like.';
COMMENT ON COLUMN social_task_logs.price IS 'Unit price for the submitted social action.';
COMMENT ON COLUMN social_task_logs.charged_amount IS 'Final charged amount; failed or unexecuted tasks remain zero.';
COMMENT ON COLUMN social_task_logs.charge_status IS 'Billing status: not_charged / charged / refunded.';
COMMENT ON COLUMN social_task_logs.charge_source IS 'Final charge source: subscription / wallet.';
COMMENT ON COLUMN social_task_logs.proxy_id IS 'User-owned proxy selected for this task.';
COMMENT ON COLUMN social_task_logs.proxy_snapshot IS 'Proxy details captured at task submission time.';
COMMENT ON COLUMN social_task_logs.billing_request_id IS 'Subscription/wallet charge or reservation request identifier.';
COMMENT ON COLUMN social_task_logs.idempotency_key IS 'Client-provided task idempotency key.';

CREATE INDEX IF NOT EXISTS idx_social_task_logs_charge_status ON social_task_logs(charge_status);
CREATE INDEX IF NOT EXISTS idx_social_task_logs_proxy_id ON social_task_logs(proxy_id);
CREATE INDEX IF NOT EXISTS idx_social_task_logs_user_account_action_idem
    ON social_task_logs(user_id, social_account_id, action, idempotency_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_social_task_logs_user_account_action_idem_unique
    ON social_task_logs(user_id, social_account_id, action, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
