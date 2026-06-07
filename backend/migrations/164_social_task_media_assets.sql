CREATE TABLE IF NOT EXISTS social_task_media_assets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    storage_provider VARCHAR(50) NOT NULL DEFAULT 'inline',
    storage_key VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    content_type VARCHAR(100) NOT NULL DEFAULT '',
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    sha256 VARCHAR(128) NOT NULL DEFAULT '',
    byte_size BIGINT NOT NULL DEFAULT 0,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_social_task_media_assets_user_storage_key
    ON social_task_media_assets(user_id, storage_key);
CREATE INDEX IF NOT EXISTS idx_social_task_media_assets_user_id
    ON social_task_media_assets(user_id);
CREATE INDEX IF NOT EXISTS idx_social_task_media_assets_sha256
    ON social_task_media_assets(sha256);

COMMENT ON TABLE social_task_media_assets IS 'Stored SocialOps-native task media assets materialized from user task submissions.';
COMMENT ON COLUMN social_task_media_assets.storage_provider IS 'Current task media origin, such as inline.';
COMMENT ON COLUMN social_task_media_assets.storage_key IS 'Internal task media lookup key stored in payload/template snapshots.';
COMMENT ON COLUMN social_task_media_assets.url IS 'Stored task media body reference; currently inline data URL for executor-safe reads.';
