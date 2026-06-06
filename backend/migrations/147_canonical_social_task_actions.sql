UPDATE social_task_logs
SET action = 'message'
WHERE action = 'dm';

UPDATE social_task_logs
SET action = 'post'
WHERE action = 'tweet';

COMMENT ON COLUMN social_task_logs.action IS 'Billable social action: login_check / follow / message / post / like.';
