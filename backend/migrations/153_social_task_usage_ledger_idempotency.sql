-- Prepare SocialOps task usage ledger rows for a partial uniqueness guarantee.
--
-- Generic API billing already has a (request_id, api_key_id) unique index, but
-- SocialOps task ledger rows do not use api_key_id. PostgreSQL treats NULLs as
-- distinct in unique indexes, so duplicate social task projections with the same
-- request_id would not be blocked by the generic constraint.

WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (PARTITION BY request_id ORDER BY id) AS rn
    FROM usage_logs
    WHERE request_id IS NOT NULL
      AND api_key_id IS NULL
      AND model = 'social-action'
)
UPDATE usage_logs ul
SET request_id = NULL
FROM ranked r
WHERE ul.id = r.id
  AND r.rn > 1;
