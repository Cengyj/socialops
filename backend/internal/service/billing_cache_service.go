package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/pkg/logger"
	"golang.org/x/sync/singleflight"
)

type cacheWriteKind int

const (
	cacheWriteSetBalance cacheWriteKind = iota
	cacheWriteSetSubscription
	cacheWriteUpdateSubscriptionUsage
	cacheWriteDeductBalance
	cacheWriteUpdateRateLimitUsage
)

const (
	cacheWriteWorkerCount = 4
	cacheWriteBufferSize  = 1000
	cacheWriteTimeout     = 2 * time.Second
	balanceLoadTimeout    = 3 * time.Second
)

type cacheWriteTask struct {
	kind             cacheWriteKind
	userID           int64
	groupID          int64
	apiKeyID         int64
	balance          float64
	amount           float64
	subscriptionData *SubscriptionCacheData
}

type BillingCacheService struct {
	cache    BillingCache
	userRepo UserRepository

	cacheWriteChan     chan cacheWriteTask
	cacheWriteWg       sync.WaitGroup
	cacheWriteStopOnce sync.Once
	cacheWriteMu       sync.RWMutex
	stopped            atomic.Bool
	balanceLoadSF      singleflight.Group
}

func NewBillingCacheService(
	cache BillingCache,
	userRepo UserRepository,
	_ UserSubscriptionRepository,
	_ APIKeyRepository,
	_ any,
	_ UserGroupRateRepository,
	_ *config.Config,
) *BillingCacheService {
	svc := &BillingCacheService{
		cache:    cache,
		userRepo: userRepo,
	}
	svc.startCacheWriteWorkers()
	return svc
}

func (s *BillingCacheService) Stop() {
	if s == nil {
		return
	}
	s.cacheWriteStopOnce.Do(func() {
		s.stopped.Store(true)

		s.cacheWriteMu.Lock()
		ch := s.cacheWriteChan
		if ch != nil {
			close(ch)
		}
		s.cacheWriteMu.Unlock()

		if ch == nil {
			return
		}
		s.cacheWriteWg.Wait()

		s.cacheWriteMu.Lock()
		if s.cacheWriteChan == ch {
			s.cacheWriteChan = nil
		}
		s.cacheWriteMu.Unlock()
	})
}

func (s *BillingCacheService) startCacheWriteWorkers() {
	ch := make(chan cacheWriteTask, cacheWriteBufferSize)
	s.cacheWriteChan = ch
	for i := 0; i < cacheWriteWorkerCount; i++ {
		s.cacheWriteWg.Add(1)
		go s.cacheWriteWorker(ch)
	}
}

func (s *BillingCacheService) enqueueCacheWrite(task cacheWriteTask) bool {
	if s == nil || s.stopped.Load() {
		return false
	}

	s.cacheWriteMu.RLock()
	defer s.cacheWriteMu.RUnlock()
	if s.cacheWriteChan == nil {
		return false
	}

	select {
	case s.cacheWriteChan <- task:
		return true
	default:
		return false
	}
}

func (s *BillingCacheService) cacheWriteWorker(ch <-chan cacheWriteTask) {
	defer s.cacheWriteWg.Done()
	for task := range ch {
		ctx, cancel := context.WithTimeout(context.Background(), cacheWriteTimeout)
		s.runCacheWrite(ctx, task)
		cancel()
	}
}

func (s *BillingCacheService) runCacheWrite(ctx context.Context, task cacheWriteTask) {
	if s == nil || s.cache == nil {
		return
	}
	switch task.kind {
	case cacheWriteSetBalance:
		if err := s.cache.SetUserBalance(ctx, task.userID, task.balance); err != nil {
			logger.LegacyPrintf("service.billing_cache", "set balance cache failed for user %d: %v", task.userID, err)
		}
	case cacheWriteSetSubscription:
		if task.subscriptionData == nil {
			return
		}
		if err := s.cache.SetSubscriptionCache(ctx, task.userID, task.groupID, task.subscriptionData); err != nil {
			logger.LegacyPrintf("service.billing_cache", "set subscription cache failed for user %d group %d: %v", task.userID, task.groupID, err)
		}
	case cacheWriteUpdateSubscriptionUsage:
		if err := s.cache.UpdateSubscriptionUsage(ctx, task.userID, task.groupID, task.amount); err != nil {
			logger.LegacyPrintf("service.billing_cache", "update subscription cache failed for user %d group %d: %v", task.userID, task.groupID, err)
		}
	case cacheWriteDeductBalance:
		if err := s.cache.DeductUserBalance(ctx, task.userID, task.amount); err != nil {
			logger.LegacyPrintf("service.billing_cache", "deduct balance cache failed for user %d: %v", task.userID, err)
		}
	case cacheWriteUpdateRateLimitUsage:
		if err := s.cache.UpdateAPIKeyRateLimitUsage(ctx, task.apiKeyID, task.amount); err != nil {
			logger.LegacyPrintf("service.billing_cache", "update api key rate limit cache failed for key %d: %v", task.apiKeyID, err)
		}
	}
}

func (s *BillingCacheService) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	if s == nil {
		return 0, errors.New("billing cache service is nil")
	}
	if s.cache != nil {
		if balance, err := s.cache.GetUserBalance(ctx, userID); err == nil {
			return balance, nil
		}
	}

	value, err, _ := s.balanceLoadSF.Do(strconv.FormatInt(userID, 10), func() (any, error) {
		loadCtx, cancel := context.WithTimeout(context.Background(), balanceLoadTimeout)
		defer cancel()

		if s.userRepo == nil {
			return nil, errors.New("user repository is unavailable")
		}
		user, err := s.userRepo.GetByID(loadCtx, userID)
		if err != nil {
			return nil, fmt.Errorf("get user balance: %w", err)
		}
		_ = s.enqueueCacheWrite(cacheWriteTask{
			kind:    cacheWriteSetBalance,
			userID:  userID,
			balance: user.Balance,
		})
		return user.Balance, nil
	})
	if err != nil {
		return 0, err
	}
	balance, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected balance type: %T", value)
	}
	return balance, nil
}

func (s *BillingCacheService) QueueDeductBalance(userID int64, amount float64) {
	if s == nil || s.cache == nil {
		return
	}
	if s.enqueueCacheWrite(cacheWriteTask{kind: cacheWriteDeductBalance, userID: userID, amount: amount}) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), cacheWriteTimeout)
	defer cancel()
	_ = s.cache.DeductUserBalance(ctx, userID, amount)
}

func (s *BillingCacheService) QueueUpdateSubscriptionUsage(userID, groupID int64, amount float64) {
	if s == nil || s.cache == nil {
		return
	}
	if s.enqueueCacheWrite(cacheWriteTask{kind: cacheWriteUpdateSubscriptionUsage, userID: userID, groupID: groupID, amount: amount}) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), cacheWriteTimeout)
	defer cancel()
	_ = s.cache.UpdateSubscriptionUsage(ctx, userID, groupID, amount)
}

func (s *BillingCacheService) InvalidateUserBalance(ctx context.Context, userID int64) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.InvalidateUserBalance(ctx, userID)
}

func (s *BillingCacheService) InvalidateSubscription(ctx context.Context, userID, groupID int64) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.InvalidateSubscriptionCache(ctx, userID, groupID)
}

func (s *BillingCacheService) InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.InvalidateAPIKeyRateLimit(ctx, keyID)
}
