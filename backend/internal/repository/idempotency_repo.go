package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/socialops/internal/service"
)

type idempotencyRepository struct {
	sql idempotencySQL
}

func NewIdempotencyRepository(db *sql.DB) service.IdempotencyRepository {
	return &idempotencyRepository{sql: db}
}

type idempotencySQL interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *idempotencyRepository) CreateProcessing(ctx context.Context, record *service.IdempotencyRecord) (bool, error) {
	if r == nil || r.sql == nil {
		return false, sql.ErrConnDone
	}
	row := r.sql.QueryRowContext(ctx, `
INSERT INTO idempotency_records (
	scope, idempotency_key_hash, request_fingerprint, status,
	locked_until, expires_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
ON CONFLICT (scope, idempotency_key_hash) DO NOTHING
RETURNING id, created_at, updated_at`,
		record.Scope,
		record.IdempotencyKeyHash,
		record.RequestFingerprint,
		record.Status,
		record.LockedUntil,
		record.ExpiresAt,
	)

	if err := row.Scan(&record.ID, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *idempotencyRepository) GetByScopeAndKeyHash(ctx context.Context, scope, keyHash string) (*service.IdempotencyRecord, error) {
	if r == nil || r.sql == nil {
		return nil, sql.ErrConnDone
	}
	row := r.sql.QueryRowContext(ctx, `
SELECT id, scope, idempotency_key_hash, request_fingerprint, status,
       response_status, response_body, error_reason, locked_until, expires_at, created_at, updated_at
FROM idempotency_records
WHERE scope = $1 AND idempotency_key_hash = $2`,
		scope,
		keyHash,
	)

	record, err := scanIdempotencyRecord(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return record, nil
}

func (r *idempotencyRepository) TryReclaim(ctx context.Context, id int64, currentStatus string, now, lockedUntil, expiresAt time.Time) (bool, error) {
	if r == nil || r.sql == nil {
		return false, sql.ErrConnDone
	}
	res, err := r.sql.ExecContext(ctx, `
UPDATE idempotency_records
SET status = $1,
    locked_until = $2,
    expires_at = $3,
    error_reason = NULL,
    response_status = NULL,
    response_body = NULL,
    updated_at = NOW()
WHERE id = $4
  AND status = $5
  AND (locked_until IS NULL OR locked_until <= $6 OR expires_at <= $6)`,
		service.IdempotencyStatusProcessing,
		lockedUntil,
		expiresAt,
		id,
		currentStatus,
		now,
	)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	return rows > 0, err
}

func (r *idempotencyRepository) ExtendProcessingLock(ctx context.Context, id int64, requestFingerprint string, lockedUntil, expiresAt time.Time) (bool, error) {
	if r == nil || r.sql == nil {
		return false, sql.ErrConnDone
	}
	res, err := r.sql.ExecContext(ctx, `
UPDATE idempotency_records
SET locked_until = $1,
    expires_at = $2,
    updated_at = NOW()
WHERE id = $3
  AND request_fingerprint = $4
  AND status = $5`,
		lockedUntil,
		expiresAt,
		id,
		requestFingerprint,
		service.IdempotencyStatusProcessing,
	)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	return rows > 0, err
}

func (r *idempotencyRepository) MarkSucceeded(ctx context.Context, id int64, responseStatus int, responseBody string, expiresAt time.Time) error {
	if r == nil || r.sql == nil {
		return sql.ErrConnDone
	}
	_, err := r.sql.ExecContext(ctx, `
UPDATE idempotency_records
SET status = $1,
    response_status = $2,
    response_body = $3,
    error_reason = NULL,
    locked_until = NULL,
    expires_at = $4,
    updated_at = NOW()
WHERE id = $5`,
		service.IdempotencyStatusSucceeded,
		responseStatus,
		responseBody,
		expiresAt,
		id,
	)
	return err
}

func (r *idempotencyRepository) MarkFailedRetryable(ctx context.Context, id int64, reason string, failedAt, expiresAt time.Time) error {
	if r == nil || r.sql == nil {
		return sql.ErrConnDone
	}
	_, err := r.sql.ExecContext(ctx, `
UPDATE idempotency_records
SET status = $1,
    error_reason = $2,
    locked_until = $3,
    expires_at = $4,
    updated_at = NOW()
WHERE id = $5`,
		service.IdempotencyStatusFailedRetryable,
		reason,
		failedAt,
		expiresAt,
		id,
	)
	return err
}

func (r *idempotencyRepository) DeleteExpired(ctx context.Context, now time.Time, limit int) (int64, error) {
	if r == nil || r.sql == nil {
		return 0, sql.ErrConnDone
	}
	if limit <= 0 {
		limit = 1000
	}
	res, err := r.sql.ExecContext(ctx, `
DELETE FROM idempotency_records
WHERE id IN (
	SELECT id
	FROM idempotency_records
	WHERE expires_at <= $1
	ORDER BY expires_at ASC
	LIMIT $2
)`,
		now,
		limit,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type idempotencyRecordScanner interface {
	Scan(dest ...any) error
}

func scanIdempotencyRecord(row idempotencyRecordScanner) (*service.IdempotencyRecord, error) {
	var record service.IdempotencyRecord
	var responseStatus sql.NullInt64
	var responseBody sql.NullString
	var errorReason sql.NullString
	var lockedUntil sql.NullTime

	if err := row.Scan(
		&record.ID,
		&record.Scope,
		&record.IdempotencyKeyHash,
		&record.RequestFingerprint,
		&record.Status,
		&responseStatus,
		&responseBody,
		&errorReason,
		&lockedUntil,
		&record.ExpiresAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if responseStatus.Valid {
		v := int(responseStatus.Int64)
		record.ResponseStatus = &v
	}
	if responseBody.Valid {
		record.ResponseBody = &responseBody.String
	}
	if errorReason.Valid {
		record.ErrorReason = &errorReason.String
	}
	if lockedUntil.Valid {
		record.LockedUntil = &lockedUntil.Time
	}

	return &record, nil
}
