package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

const (
	IdempotencyStatusProcessing      = "processing"
	IdempotencyStatusSucceeded       = "succeeded"
	IdempotencyStatusFailedRetryable = "failed_retryable"
)

var (
	ErrIdempotencyKeyRequired    = infraerrors.BadRequest("IDEMPOTENCY_KEY_REQUIRED", "idempotency key is required")
	ErrIdempotencyKeyInvalid     = infraerrors.BadRequest("IDEMPOTENCY_KEY_INVALID", "idempotency key is invalid")
	ErrIdempotencyKeyConflict    = infraerrors.Conflict("IDEMPOTENCY_KEY_CONFLICT", "idempotency key reused with different payload")
	ErrIdempotencyInProgress     = infraerrors.Conflict("IDEMPOTENCY_IN_PROGRESS", "idempotent request is still processing")
	ErrIdempotencyRetryBackoff   = infraerrors.Conflict("IDEMPOTENCY_RETRY_BACKOFF", "idempotent request is in retry backoff window")
	ErrIdempotencyStoreUnavail   = infraerrors.ServiceUnavailable("IDEMPOTENCY_STORE_UNAVAILABLE", "idempotency store unavailable")
	ErrIdempotencyInvalidPayload = infraerrors.BadRequest("IDEMPOTENCY_PAYLOAD_INVALID", "failed to normalize request payload")
)

// IdempotencyRepository is the storage interface for idempotency records.
// Implementation removed in Phase 2D; kept as interface for SystemOperationLockService.
type IdempotencyRepository interface {
	CreateProcessing(ctx context.Context, record *IdempotencyRecord) (owner bool, err error)
	GetByScopeAndKeyHash(ctx context.Context, scope, keyHash string) (*IdempotencyRecord, error)
	TryReclaim(ctx context.Context, id int64, currentStatus string, now, lockedUntil, expiresAt time.Time) (bool, error)
	ExtendProcessingLock(ctx context.Context, id int64, operationID string, lockedUntil, expiresAt time.Time) (bool, error)
	MarkSucceeded(ctx context.Context, id int64, responseStatus int, responseBody string, expiresAt time.Time) error
	MarkFailedRetryable(ctx context.Context, id int64, reason string, failedAt, expiresAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time, limit int) (int64, error)
}

// IdempotencyConfig holds configuration for the idempotency coordinator.
type IdempotencyConfig struct {
	DefaultTTL           time.Duration
	SystemOperationTTL   time.Duration
	ProcessingTimeout    time.Duration
	FailedRetryBackoff   time.Duration
	MaxStoredResponseLen int
	ObserveOnly          bool
}

// DefaultIdempotencyConfig returns safe defaults.
func DefaultIdempotencyConfig() IdempotencyConfig {
	return IdempotencyConfig{
		DefaultTTL:           24 * time.Hour,
		SystemOperationTTL:   time.Hour,
		ProcessingTimeout:    30 * time.Second,
		FailedRetryBackoff:   5 * time.Second,
		MaxStoredResponseLen: 64 * 1024,
		ObserveOnly:          true,
	}
}

// DefaultWriteIdempotencyTTL returns the default TTL for write operations.
func DefaultWriteIdempotencyTTL() time.Duration {
	if c := DefaultIdempotencyCoordinator(); c != nil {
		return c.cfg.DefaultTTL
	}
	return 24 * time.Hour
}

// DefaultSystemOperationIdempotencyTTL returns the TTL for system operations.
func DefaultSystemOperationIdempotencyTTL() time.Duration {
	if c := DefaultIdempotencyCoordinator(); c != nil {
		return c.cfg.SystemOperationTTL
	}
	return time.Hour
}

// IdempotencyExecuteOptions holds options for an idempotent execution.
type IdempotencyExecuteOptions struct {
	Scope          string
	ActorScope     string
	Method         string
	Route          string
	IdempotencyKey string
	Payload        any
	TTL            time.Duration
	RequireKey     bool
}

// IdempotencyExecuteResult holds the result of an idempotent execution.
type IdempotencyExecuteResult struct {
	Data     any
	Replayed bool
}

// IdempotencyCoordinator coordinates idempotent operations.
// When repo is nil, all operations execute directly without deduplication.
type IdempotencyCoordinator struct {
	repo IdempotencyRepository
	cfg  IdempotencyConfig
}

var (
	defaultIdempotencyMu          sync.RWMutex
	defaultIdempotencyCoordinator *IdempotencyCoordinator
)

// SetDefaultIdempotencyCoordinator sets the global coordinator.
func SetDefaultIdempotencyCoordinator(c *IdempotencyCoordinator) {
	defaultIdempotencyMu.Lock()
	defer defaultIdempotencyMu.Unlock()
	defaultIdempotencyCoordinator = c
}

// DefaultIdempotencyCoordinator returns the global coordinator (may be nil).
func DefaultIdempotencyCoordinator() *IdempotencyCoordinator {
	defaultIdempotencyMu.RLock()
	defer defaultIdempotencyMu.RUnlock()
	return defaultIdempotencyCoordinator
}

// NewIdempotencyCoordinator creates a new coordinator.
func NewIdempotencyCoordinator(repo IdempotencyRepository, cfg IdempotencyConfig) *IdempotencyCoordinator {
	return &IdempotencyCoordinator{repo: repo, cfg: cfg}
}

// Execute runs fn, returning its result. When repo is nil, executes directly.
func (c *IdempotencyCoordinator) Execute(
	ctx context.Context,
	opts IdempotencyExecuteOptions,
	execute func(context.Context) (any, error),
) (*IdempotencyExecuteResult, error) {
	if execute == nil {
		return nil, infraerrors.InternalServer("IDEMPOTENCY_EXECUTOR_NIL", "idempotency executor is nil")
	}

	key, err := NormalizeIdempotencyKey(opts.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if key == "" {
		if opts.RequireKey && !c.cfg.ObserveOnly {
			return nil, ErrIdempotencyKeyRequired
		}
		data, err := execute(ctx)
		if err != nil {
			return nil, err
		}
		return &IdempotencyExecuteResult{Data: data}, nil
	}
	if c == nil || c.repo == nil {
		RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "repo_nil")
		return nil, ErrIdempotencyStoreUnavail
	}
	if strings.TrimSpace(opts.Scope) == "" {
		return nil, infraerrors.BadRequest("IDEMPOTENCY_SCOPE_REQUIRED", "idempotency scope is required")
	}

	fingerprint, err := BuildIdempotencyFingerprint(opts.Method, opts.Route, opts.ActorScope, opts.Payload)
	if err != nil {
		return nil, err
	}

	ttl := opts.TTL
	if ttl <= 0 {
		ttl = c.cfg.DefaultTTL
	}
	if ttl <= 0 {
		ttl = DefaultIdempotencyConfig().DefaultTTL
	}
	processingTimeout := c.cfg.ProcessingTimeout
	if processingTimeout <= 0 {
		processingTimeout = DefaultIdempotencyConfig().ProcessingTimeout
	}
	now := time.Now()
	expiresAt := now.Add(ttl)
	lockedUntil := now.Add(processingTimeout)
	keyHash := HashIdempotencyKey(key)

	record := &IdempotencyRecord{
		Scope:              opts.Scope,
		IdempotencyKeyHash: keyHash,
		RequestFingerprint: fingerprint,
		Status:             IdempotencyStatusProcessing,
		LockedUntil:        &lockedUntil,
		ExpiresAt:          expiresAt,
	}

	owner, err := c.repo.CreateProcessing(ctx, record)
	if err != nil {
		RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "create_processing_error")
		return nil, ErrIdempotencyStoreUnavail.WithCause(err)
	}
	if owner {
		recordIdempotencyClaim(opts.Route, opts.Scope, map[string]string{"mode": "new_claim"})
	}
	if !owner {
		existing, getErr := c.repo.GetByScopeAndKeyHash(ctx, opts.Scope, keyHash)
		if getErr != nil {
			RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "get_existing_error")
			return nil, ErrIdempotencyStoreUnavail.WithCause(getErr)
		}
		if existing == nil {
			RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "missing_existing")
			return nil, ErrIdempotencyStoreUnavail
		}
		if existing.RequestFingerprint != fingerprint {
			recordIdempotencyConflict(opts.Route, opts.Scope, map[string]string{"reason": "fingerprint_mismatch"})
			return nil, ErrIdempotencyKeyConflict
		}

		reclaimedByExpired := false
		if !existing.ExpiresAt.After(now) {
			taken, reclaimErr := c.repo.TryReclaim(ctx, existing.ID, existing.Status, now, lockedUntil, expiresAt)
			if reclaimErr != nil {
				RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "try_reclaim_expired_error")
				return nil, ErrIdempotencyStoreUnavail.WithCause(reclaimErr)
			}
			if taken {
				reclaimedByExpired = true
				record.ID = existing.ID
				recordIdempotencyClaim(opts.Route, opts.Scope, map[string]string{"mode": "expired_reclaim"})
			} else {
				latest, latestErr := c.repo.GetByScopeAndKeyHash(ctx, opts.Scope, keyHash)
				if latestErr != nil {
					RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "get_existing_after_expired_reclaim_error")
					return nil, ErrIdempotencyStoreUnavail.WithCause(latestErr)
				}
				if latest == nil {
					RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "missing_existing_after_expired_reclaim")
					return nil, ErrIdempotencyStoreUnavail
				}
				if latest.RequestFingerprint != fingerprint {
					recordIdempotencyConflict(opts.Route, opts.Scope, map[string]string{"reason": "fingerprint_mismatch"})
					return nil, ErrIdempotencyKeyConflict
				}
				existing = latest
			}
		}

		if !reclaimedByExpired {
			switch existing.Status {
			case IdempotencyStatusSucceeded:
				data, parseErr := c.decodeStoredResponse(existing.ResponseBody)
				if parseErr != nil {
					RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "decode_stored_response_error")
					return nil, ErrIdempotencyStoreUnavail.WithCause(parseErr)
				}
				recordIdempotencyReplay(opts.Route, opts.Scope, nil)
				return &IdempotencyExecuteResult{Data: data, Replayed: true}, nil
			case IdempotencyStatusProcessing:
				recordIdempotencyConflict(opts.Route, opts.Scope, map[string]string{"reason": "in_progress"})
				return nil, c.conflictWithRetryAfter(ErrIdempotencyInProgress, existing.LockedUntil, now)
			case IdempotencyStatusFailedRetryable:
				if existing.LockedUntil != nil && existing.LockedUntil.After(now) {
					recordIdempotencyConflict(opts.Route, opts.Scope, map[string]string{"reason": "retry_backoff"})
					recordIdempotencyRetryBackoff(opts.Route, opts.Scope, nil)
					return nil, c.conflictWithRetryAfter(ErrIdempotencyRetryBackoff, existing.LockedUntil, now)
				}
				taken, reclaimErr := c.repo.TryReclaim(ctx, existing.ID, IdempotencyStatusFailedRetryable, now, lockedUntil, expiresAt)
				if reclaimErr != nil {
					RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "try_reclaim_error")
					return nil, ErrIdempotencyStoreUnavail.WithCause(reclaimErr)
				}
				if !taken {
					recordIdempotencyConflict(opts.Route, opts.Scope, map[string]string{"reason": "reclaim_race"})
					return nil, c.conflictWithRetryAfter(ErrIdempotencyInProgress, existing.LockedUntil, now)
				}
				record.ID = existing.ID
				recordIdempotencyClaim(opts.Route, opts.Scope, map[string]string{"mode": "reclaim"})
			default:
				recordIdempotencyConflict(opts.Route, opts.Scope, map[string]string{"reason": "unexpected_status"})
				return nil, ErrIdempotencyKeyConflict
			}
		}
	}

	if record.ID == 0 {
		RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "record_id_missing")
		return nil, ErrIdempotencyStoreUnavail
	}

	start := time.Now()
	defer func() {
		recordIdempotencyProcessingDuration(opts.Route, opts.Scope, time.Since(start), nil)
	}()

	data, err := execute(ctx)
	if err != nil {
		backoff := c.cfg.FailedRetryBackoff
		if backoff <= 0 {
			backoff = DefaultIdempotencyConfig().FailedRetryBackoff
		}
		backoffUntil := time.Now().Add(backoff)
		reason := infraerrors.Reason(err)
		if reason == "" {
			reason = "EXECUTION_FAILED"
		}
		recordIdempotencyRetryBackoff(opts.Route, opts.Scope, nil)
		if markErr := c.repo.MarkFailedRetryable(ctx, record.ID, reason, backoffUntil, expiresAt); markErr != nil {
			RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "mark_failed_retryable_error")
		}
		return nil, err
	}

	body, err := c.marshalStoredResponse(data)
	if err != nil {
		RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "marshal_response_error")
		return nil, ErrIdempotencyStoreUnavail.WithCause(err)
	}
	if err := c.repo.MarkSucceeded(ctx, record.ID, 200, body, expiresAt); err != nil {
		RecordIdempotencyStoreUnavailable(opts.Route, opts.Scope, "mark_succeeded_error")
		return nil, ErrIdempotencyStoreUnavail.WithCause(err)
	}
	return &IdempotencyExecuteResult{Data: data}, nil
}

// IdempotencyCleanupService is a no-op placeholder.
type IdempotencyCleanupService struct{}

// NewIdempotencyCleanupService creates a no-op cleanup service.
func NewIdempotencyCleanupService(_ IdempotencyRepository, _ *any) *IdempotencyCleanupService {
	return &IdempotencyCleanupService{}
}

// Start is a no-op.
func (s *IdempotencyCleanupService) Start() {}

// Stop is a no-op.
func (s *IdempotencyCleanupService) Stop() {}

// NormalizeIdempotencyKey validates and normalizes an idempotency key.
func NormalizeIdempotencyKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil
	}
	if len(key) > 128 {
		return "", ErrIdempotencyKeyInvalid
	}
	for _, r := range key {
		if r < 33 || r > 126 {
			return "", ErrIdempotencyKeyInvalid
		}
	}
	return key, nil
}

func HashIdempotencyKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func BuildIdempotencyFingerprint(method, route, actorScope string, payload any) (string, error) {
	if method == "" {
		method = "POST"
	}
	if route == "" {
		route = "/"
	}
	if actorScope == "" {
		actorScope = "anonymous"
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", ErrIdempotencyInvalidPayload.WithCause(err)
	}
	sum := sha256.Sum256([]byte(strings.ToUpper(method) + "\n" + route + "\n" + actorScope + "\n" + string(raw)))
	return hex.EncodeToString(sum[:]), nil
}

func RetryAfterSecondsFromError(err error) int {
	appErr := new(infraerrors.ApplicationError)
	if !errors.As(err, &appErr) || appErr == nil || appErr.Metadata == nil {
		return 0
	}
	seconds, convErr := strconv.Atoi(strings.TrimSpace(appErr.Metadata["retry_after"]))
	if convErr != nil || seconds <= 0 {
		return 0
	}
	return seconds
}

func (c *IdempotencyCoordinator) conflictWithRetryAfter(base *infraerrors.ApplicationError, lockedUntil *time.Time, now time.Time) error {
	if lockedUntil == nil {
		return base
	}
	seconds := int(lockedUntil.Sub(now).Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return base.WithMetadata(map[string]string{"retry_after": strconv.Itoa(seconds)})
}

func (c *IdempotencyCoordinator) marshalStoredResponse(data any) (string, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	body := string(raw)
	if c != nil && c.cfg.MaxStoredResponseLen > 0 && len(body) > c.cfg.MaxStoredResponseLen {
		body = body[:c.cfg.MaxStoredResponseLen] + "...(truncated)"
	}
	return body, nil
}

func (c *IdempotencyCoordinator) decodeStoredResponse(stored *string) (any, error) {
	if stored == nil || strings.TrimSpace(*stored) == "" {
		return map[string]any{}, nil
	}
	var out any
	if err := json.Unmarshal([]byte(*stored), &out); err != nil {
		return nil, fmt.Errorf("decode stored response: %w", err)
	}
	return out, nil
}

type IdempotencyMetricsSnapshot struct {
	ClaimTotal                uint64  `json:"claim_total"`
	ReplayTotal               uint64  `json:"replay_total"`
	ConflictTotal             uint64  `json:"conflict_total"`
	RetryBackoffTotal         uint64  `json:"retry_backoff_total"`
	ProcessingDurationCount   uint64  `json:"processing_duration_count"`
	ProcessingDurationTotalMs float64 `json:"processing_duration_total_ms"`
	StoreUnavailableTotal     uint64  `json:"store_unavailable_total"`
}

type idempotencyMetrics struct {
	claimTotal               atomic.Uint64
	replayTotal              atomic.Uint64
	conflictTotal            atomic.Uint64
	retryBackoffTotal        atomic.Uint64
	processingDurationCount  atomic.Uint64
	processingDurationMicros atomic.Uint64
	storeUnavailableTotal    atomic.Uint64
}

var defaultIdempotencyMetrics idempotencyMetrics

func GetIdempotencyMetricsSnapshot() IdempotencyMetricsSnapshot {
	totalMicros := defaultIdempotencyMetrics.processingDurationMicros.Load()
	return IdempotencyMetricsSnapshot{
		ClaimTotal:                defaultIdempotencyMetrics.claimTotal.Load(),
		ReplayTotal:               defaultIdempotencyMetrics.replayTotal.Load(),
		ConflictTotal:             defaultIdempotencyMetrics.conflictTotal.Load(),
		RetryBackoffTotal:         defaultIdempotencyMetrics.retryBackoffTotal.Load(),
		ProcessingDurationCount:   defaultIdempotencyMetrics.processingDurationCount.Load(),
		ProcessingDurationTotalMs: float64(totalMicros) / 1000.0,
		StoreUnavailableTotal:     defaultIdempotencyMetrics.storeUnavailableTotal.Load(),
	}
}

func recordIdempotencyClaim(_ string, _ string, _ map[string]string) {
	defaultIdempotencyMetrics.claimTotal.Add(1)
}

func recordIdempotencyReplay(_ string, _ string, _ map[string]string) {
	defaultIdempotencyMetrics.replayTotal.Add(1)
}

func recordIdempotencyConflict(_ string, _ string, _ map[string]string) {
	defaultIdempotencyMetrics.conflictTotal.Add(1)
}

func recordIdempotencyRetryBackoff(_ string, _ string, _ map[string]string) {
	defaultIdempotencyMetrics.retryBackoffTotal.Add(1)
}

func recordIdempotencyProcessingDuration(_ string, _ string, duration time.Duration, _ map[string]string) {
	if duration < 0 {
		duration = 0
	}
	defaultIdempotencyMetrics.processingDurationCount.Add(1)
	defaultIdempotencyMetrics.processingDurationMicros.Add(uint64(duration.Microseconds()))
}

func RecordIdempotencyStoreUnavailable(_, _, _ string) {
	defaultIdempotencyMetrics.storeUnavailableTotal.Add(1)
}

func resetIdempotencyMetricsForTest() {
	defaultIdempotencyMetrics.claimTotal.Store(0)
	defaultIdempotencyMetrics.replayTotal.Store(0)
	defaultIdempotencyMetrics.conflictTotal.Store(0)
	defaultIdempotencyMetrics.retryBackoffTotal.Store(0)
	defaultIdempotencyMetrics.processingDurationCount.Store(0)
	defaultIdempotencyMetrics.processingDurationMicros.Store(0)
	defaultIdempotencyMetrics.storeUnavailableTotal.Store(0)
}
