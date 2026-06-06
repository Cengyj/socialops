package service

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/socialops/internal/config"
)

// DashboardAggregationRepository is the SocialOps usage aggregation port.
type DashboardAggregationRepository interface {
	RecomputeRange(ctx context.Context, start, end time.Time) error
}

// DashboardAggregationService keeps social usage aggregates consistent after
// cleanup tasks. Full periodic aggregation can be expanded with the social task
// log model; this recovery phase only needs a safe recompute hook.
type DashboardAggregationService struct {
	repo    DashboardAggregationRepository
	enabled bool
	running int32
}

func NewDashboardAggregationService(repo DashboardAggregationRepository, _ any, cfg *config.Config) *DashboardAggregationService {
	enabled := true
	if cfg != nil {
		enabled = cfg.DashboardAgg.Enabled
	}
	return &DashboardAggregationService{repo: repo, enabled: enabled}
}

func (s *DashboardAggregationService) TriggerRecomputeRange(start, end time.Time) error {
	if s == nil || s.repo == nil {
		return errors.New("dashboard aggregation service not ready")
	}
	if !s.enabled {
		return errors.New("dashboard aggregation is disabled")
	}
	if !end.After(start) {
		return errors.New("invalid recompute range")
	}
	go func() {
		if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
			return
		}
		defer atomic.StoreInt32(&s.running, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		_ = s.repo.RecomputeRange(ctx, start, end)
	}()
	return nil
}
