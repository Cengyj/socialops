package service

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type ConcurrencyCache interface {
	AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error)
	ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error
	GetAccountConcurrency(ctx context.Context, accountID int64) (int, error)
	GetAccountConcurrencyBatch(ctx context.Context, accountIDs []int64) (map[int64]int, error)
	IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error)
	DecrementAccountWaitCount(ctx context.Context, accountID int64) error
	GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error)
	AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error)
	ReleaseUserSlot(ctx context.Context, userID int64, requestID string) error
	GetUserConcurrency(ctx context.Context, userID int64) (int, error)
	IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error)
	DecrementWaitCount(ctx context.Context, userID int64) error
	GetAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error)
	GetUsersLoadBatch(ctx context.Context, users []UserWithConcurrency) (map[int64]*UserLoadInfo, error)
	CleanupExpiredAccountSlots(ctx context.Context, accountID int64) error
	CleanupStaleProcessSlots(ctx context.Context, requestIDPrefix string) error
}

type ConcurrencyService struct {
	cache                    ConcurrencyCache
	accountLoadBatchCacheTTL time.Duration
	accountLoadBatchCache    map[string]accountLoadBatchCacheEntry
	accountLoadBatchMu       sync.Mutex
}

type AccountWithConcurrency struct {
	ID             int64
	MaxConcurrency int
}

type UserWithConcurrency struct {
	ID             int64
	MaxConcurrency int
}

type AccountLoadInfo struct {
	AccountID          int64   `json:"account_id"`
	CurrentConcurrency int     `json:"current_concurrency"`
	WaitingCount       int     `json:"waiting_count"`
	LoadRate           float64 `json:"load_rate"`
}

type UserLoadInfo struct {
	UserID             int64   `json:"user_id"`
	CurrentConcurrency int     `json:"current_concurrency"`
	WaitingCount       int     `json:"waiting_count"`
	LoadRate           float64 `json:"load_rate"`
}

type ConcurrencyAcquireResult struct {
	Acquired    bool
	RequestID   string
	ReleaseFunc func()
}

type accountLoadBatchCacheEntry struct {
	expiresAt time.Time
	value     map[int64]*AccountLoadInfo
}

var requestIDCounter atomic.Uint64

func NewConcurrencyService(cache ConcurrencyCache) *ConcurrencyService {
	return &ConcurrencyService{
		cache:                    cache,
		accountLoadBatchCacheTTL: 0,
		accountLoadBatchCache:    make(map[string]accountLoadBatchCacheEntry),
	}
}

func RequestIDPrefix() string {
	return "socialops"
}

func generateRequestID() string {
	next := requestIDCounter.Add(1)
	return RequestIDPrefix() + "-" + strconv.FormatUint(next, 36)
}

func CalculateMaxWait(concurrency int) int {
	if concurrency < 1 {
		concurrency = 1
	}
	return concurrency + 20
}

func (s *ConcurrencyService) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int) (*ConcurrencyAcquireResult, error) {
	requestID := generateRequestID()
	if maxConcurrency <= 0 || s == nil || s.cache == nil {
		return &ConcurrencyAcquireResult{Acquired: true, RequestID: requestID, ReleaseFunc: func() {}}, nil
	}
	acquired, err := s.cache.AcquireAccountSlot(ctx, accountID, maxConcurrency, requestID)
	if err != nil {
		return nil, err
	}
	result := &ConcurrencyAcquireResult{Acquired: acquired, RequestID: requestID}
	if acquired {
		result.ReleaseFunc = func() {
			_ = s.cache.ReleaseAccountSlot(context.Background(), accountID, requestID)
		}
	}
	return result, nil
}

func (s *ConcurrencyService) AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int) (*ConcurrencyAcquireResult, error) {
	requestID := generateRequestID()
	if maxConcurrency <= 0 || s == nil || s.cache == nil {
		return &ConcurrencyAcquireResult{Acquired: true, RequestID: requestID, ReleaseFunc: func() {}}, nil
	}
	acquired, err := s.cache.AcquireUserSlot(ctx, userID, maxConcurrency, requestID)
	if err != nil {
		return nil, err
	}
	result := &ConcurrencyAcquireResult{Acquired: acquired, RequestID: requestID}
	if acquired {
		result.ReleaseFunc = func() {
			_ = s.cache.ReleaseUserSlot(context.Background(), userID, requestID)
		}
	}
	return result, nil
}

func (s *ConcurrencyService) SetAccountLoadBatchCacheTTL(ttl time.Duration) {
	if s == nil {
		return
	}
	s.accountLoadBatchCacheTTL = ttl
	if s.accountLoadBatchCache == nil {
		s.accountLoadBatchCache = make(map[string]accountLoadBatchCacheEntry)
	}
}

func (s *ConcurrencyService) GetAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	if s == nil || s.cache == nil || len(accounts) == 0 {
		return map[int64]*AccountLoadInfo{}, nil
	}
	key := accountLoadBatchKey(accounts)
	if s.accountLoadBatchCacheTTL > 0 {
		s.accountLoadBatchMu.Lock()
		if entry, ok := s.accountLoadBatchCache[key]; ok && time.Now().Before(entry.expiresAt) {
			out := cloneAccountLoadBatch(entry.value)
			s.accountLoadBatchMu.Unlock()
			return out, nil
		}
		s.accountLoadBatchMu.Unlock()
	}
	result, err := s.cache.GetAccountsLoadBatch(ctx, accounts)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = map[int64]*AccountLoadInfo{}
	}
	if s.accountLoadBatchCacheTTL > 0 {
		s.accountLoadBatchMu.Lock()
		s.accountLoadBatchCache[key] = accountLoadBatchCacheEntry{
			expiresAt: time.Now().Add(s.accountLoadBatchCacheTTL),
			value:     cloneAccountLoadBatch(result),
		}
		s.accountLoadBatchMu.Unlock()
	}
	return result, nil
}

func (s *ConcurrencyService) GetAccountsLoadBatchFresh(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	if s == nil || s.cache == nil || len(accounts) == 0 {
		return map[int64]*AccountLoadInfo{}, nil
	}
	result, err := s.cache.GetAccountsLoadBatch(ctx, accounts)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = map[int64]*AccountLoadInfo{}
	}
	return result, nil
}

func (s *ConcurrencyService) GetUsersLoadBatch(ctx context.Context, users []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	if s == nil || s.cache == nil || len(users) == 0 {
		return map[int64]*UserLoadInfo{}, nil
	}
	result, err := s.cache.GetUsersLoadBatch(ctx, users)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = map[int64]*UserLoadInfo{}
	}
	return result, nil
}

func (s *ConcurrencyService) IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error) {
	if s == nil || s.cache == nil {
		return true, nil
	}
	allowed, err := s.cache.IncrementWaitCount(ctx, userID, maxWait)
	if err != nil {
		return true, nil
	}
	return allowed, nil
}

func (s *ConcurrencyService) IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error) {
	if s == nil || s.cache == nil {
		return true, nil
	}
	allowed, err := s.cache.IncrementAccountWaitCount(ctx, accountID, maxWait)
	if err != nil {
		return true, nil
	}
	return allowed, nil
}

func (s *ConcurrencyService) GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error) {
	if s == nil || s.cache == nil {
		return 0, nil
	}
	return s.cache.GetAccountWaitingCount(ctx, accountID)
}

func (s *ConcurrencyService) GetAccountConcurrencyBatch(ctx context.Context, accountIDs []int64) (map[int64]int, error) {
	if s == nil || s.cache == nil || len(accountIDs) == 0 {
		return map[int64]int{}, nil
	}
	return s.cache.GetAccountConcurrencyBatch(ctx, accountIDs)
}

func (s *ConcurrencyService) CleanupStaleProcessSlots(ctx context.Context) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.CleanupStaleProcessSlots(ctx, RequestIDPrefix())
}

func accountLoadBatchKey(accounts []AccountWithConcurrency) string {
	key := ""
	for _, account := range accounts {
		key += strconv.FormatInt(account.ID, 10) + ":" + strconv.Itoa(account.MaxConcurrency) + ";"
	}
	return key
}

func cloneAccountLoadBatch(in map[int64]*AccountLoadInfo) map[int64]*AccountLoadInfo {
	out := make(map[int64]*AccountLoadInfo, len(in))
	for id, info := range in {
		if info == nil {
			out[id] = nil
			continue
		}
		clone := *info
		out[id] = &clone
	}
	return out
}
