ALTER TABLE social_task_logs
    ADD COLUMN IF NOT EXISTS payload JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE social_task_logs
    ADD COLUMN IF NOT EXISTS template_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN social_task_logs.payload IS 'Structured execution payload snapshot for social task execution.';
COMMENT ON COLUMN social_task_logs.template_snapshot IS 'Saved task template snapshot resolved at submission time.';
