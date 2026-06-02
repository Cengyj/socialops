-- Add SocialOps social account/task foundations.
-- Forward-only and additive: these tables are new SocialOps business tables and
-- do not restore the legacy AI account credential pool.

CREATE TABLE IF NOT EXISTS social_accounts (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    platform VARCHAR(50) NOT NULL,
    account_id VARCHAR(100),
    password VARCHAR(255),
    phone VARCHAR(50),
    email VARCHAR(255),
    email_password VARCHAR(255),
    account_status VARCHAR(30) NOT NULL DEFAULT 'pending_check',
    task_status VARCHAR(30) NOT NULL DEFAULT 'pending',
    task_message TEXT,
    source VARCHAR(30) NOT NULL DEFAULT 'manual_import',
    bound_ip VARCHAR(255),
    assigned_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    remark TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_social_accounts_platform ON social_accounts(platform);
CREATE INDEX IF NOT EXISTS idx_social_accounts_account_status ON social_accounts(account_status);
CREATE INDEX IF NOT EXISTS idx_social_accounts_task_status ON social_accounts(task_status);
CREATE INDEX IF NOT EXISTS idx_social_accounts_assigned_user_id ON social_accounts(assigned_user_id);
CREATE INDEX IF NOT EXISTS idx_social_accounts_source ON social_accounts(source);
CREATE INDEX IF NOT EXISTS idx_social_accounts_deleted_at ON social_accounts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_social_accounts_platform_account_status ON social_accounts(platform, account_status);

CREATE TABLE IF NOT EXISTS social_ips (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    ip_type VARCHAR(30) NOT NULL DEFAULT 'residential',
    endpoint VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    latency_ms INTEGER,
    last_check_at TIMESTAMPTZ,
    bound_social_account_id BIGINT REFERENCES social_accounts(id) ON DELETE SET NULL,
    remark TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_social_ips_user_id ON social_ips(user_id);
CREATE INDEX IF NOT EXISTS idx_social_ips_status ON social_ips(status);
CREATE INDEX IF NOT EXISTS idx_social_ips_ip_type ON social_ips(ip_type);
CREATE INDEX IF NOT EXISTS idx_social_ips_deleted_at ON social_ips(deleted_at);
CREATE INDEX IF NOT EXISTS idx_social_ips_user_id_status ON social_ips(user_id, status);
CREATE INDEX IF NOT EXISTS idx_social_ips_bound_social_account_id ON social_ips(bound_social_account_id);

CREATE TABLE IF NOT EXISTS social_task_logs (
    id BIGSERIAL PRIMARY KEY,
    social_account_id BIGINT NOT NULL REFERENCES social_accounts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    target TEXT,
    content TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    result_message TEXT,
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_social_task_logs_social_account_id ON social_task_logs(social_account_id);
CREATE INDEX IF NOT EXISTS idx_social_task_logs_user_id ON social_task_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_social_task_logs_action ON social_task_logs(action);
CREATE INDEX IF NOT EXISTS idx_social_task_logs_status ON social_task_logs(status);
CREATE INDEX IF NOT EXISTS idx_social_task_logs_user_id_action ON social_task_logs(user_id, action);
CREATE INDEX IF NOT EXISTS idx_social_task_logs_social_account_id_status ON social_task_logs(social_account_id, status);
