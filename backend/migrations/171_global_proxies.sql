CREATE TABLE IF NOT EXISTS global_proxies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    ip_type VARCHAR(30) NOT NULL DEFAULT 'residential',
    endpoint VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    latency_ms INTEGER,
    last_check_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    remark TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_global_proxies_status ON global_proxies(status);
CREATE INDEX IF NOT EXISTS idx_global_proxies_ip_type ON global_proxies(ip_type);
CREATE INDEX IF NOT EXISTS idx_global_proxies_deleted_at ON global_proxies(deleted_at);
CREATE INDEX IF NOT EXISTS idx_global_proxies_status_last_used_at ON global_proxies(status, last_used_at);
