package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/socialops/internal/config"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/logger"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
)

const usageCleanupWorkerName = "usage_cleanup_worker"

const (
	UsageCleanupStatusPending   = "pending"
	UsageCleanupStatusRunning   = "running"
	UsageCleanupStatusSucceeded = "succeeded"
	UsageCleanupStatusFailed    = "failed"
	UsageCleanupStatusCanceled  = "canceled"
)

type UsageCleanupFilters struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	UserID    *int64    `json:"user_id,omitempty"`
	APIKeyID  *int64    `json:"api_key_id,omitempty"`
	AccountID *int64    `json:"account_id,omitempty"`
	GroupID   *int64    `json:"group_id,omitempty"`
	Operation *string   `json:"operation,omitempty"`
	Status    *string   `json:"status,omitempty"`
}

type UsageCleanupTask struct {
	ID          int64               `json:"id"`
	Status      string              `json:"status"`
	Filters     UsageCleanupFilters `json:"filters"`
	CreatedBy   int64               `json:"created_by"`
	DeletedRows int64               `json:"deleted_rows"`
	ErrorMsg    *string             `json:"error_message,omitempty"`
	CanceledBy  *int64              `json:"canceled_by,omitempty"`
	CanceledAt  *time.Time          `json:"canceled_at,omitempty"`
	StartedAt   *time.Time          `json:"started_at,omitempty"`
	FinishedAt  *time.Time          `json:"finished_at,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type UsageCleanupRepository interface {
	CreateTask(ctx context.Context, task *UsageCleanupTask) error
	ListTasks(ctx context.Context, params pagination.PaginationParams) ([]UsageCleanupTask, *pagination.PaginationResult, error)
	ClaimNextPendingTask(ctx context.Context, staleRunningAfterSeconds int64) (*UsageCleanupTask, error)
	GetTaskStatus(ctx context.Context, taskID int64) (string, error)
	UpdateTaskProgress(ctx context.Context, taskID int64, deletedRows int64) error
	CancelTask(ctx context.Context, taskID int64, canceledBy int64) (bool, error)
	MarkTaskSucceeded(ctx context.Context, taskID int64, deletedRows int64) error
	MarkTaskFailed(ctx context.Context, taskID int64, deletedRows int64, errorMsg string) error
	DeleteUsageLogsBatch(ctx context.Context, filters UsageCleanupFilters, limit int) (int64, error)
}

// UsageCleanupService creates and executes cleanup tasks for SocialOps usage logs.
type UsageCleanupService struct {
	repo      UsageCleanupRepository
	dashboard *DashboardAggregationService
	cfg       *config.Config

	running   int32
	startOnce sync.Once
	stopOnce  sync.Once

	workerCtx    context.Context
	workerCancel context.CancelFunc
}

func NewUsageCleanupService(repo UsageCleanupRepository, _ any, dashboard *DashboardAggregationService, cfg *config.Config) *UsageCleanupService {
	workerCtx, workerCancel := context.WithCancel(context.Background())
	return &UsageCleanupService{
		repo:         repo,
		dashboard:    dashboard,
		cfg:          cfg,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}
}

func describeUsageCleanupFilters(filters UsageCleanupFilters) string {
	var parts []string
	parts = append(parts, "start="+filters.StartTime.UTC().Format(time.RFC3339))
	parts = append(parts, "end="+filters.EndTime.UTC().Format(time.RFC3339))
	if filters.UserID != nil {
		parts = append(parts, fmt.Sprintf("user_id=%d", *filters.UserID))
	}
	if filters.APIKeyID != nil {
		parts = append(parts, fmt.Sprintf("api_key_id=%d", *filters.APIKeyID))
	}
	if filters.AccountID != nil {
		parts = append(parts, fmt.Sprintf("account_id=%d", *filters.AccountID))
	}
	if filters.GroupID != nil {
		parts = append(parts, fmt.Sprintf("group_id=%d", *filters.GroupID))
	}
	if filters.Operation != nil {
		parts = append(parts, "operation="+strings.TrimSpace(*filters.Operation))
	}
	if filters.Status != nil {
		parts = append(parts, "status="+strings.TrimSpace(*filters.Status))
	}
	return strings.Join(parts, " ")
}

func (s *UsageCleanupService) Start() {
	if s == nil {
		return
	}
	if s.cfg != nil && !s.cfg.UsageCleanup.Enabled {
		logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] not started (disabled)")
		return
	}
	if s.repo == nil {
		logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] not started (missing repo)")
		return
	}
	interval := s.workerInterval()
	s.startOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-s.workerCtx.Done():
					return
				case <-ticker.C:
					s.runOnce()
				}
			}
		}()
		logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] started (interval=%s max_range_days=%d batch_size=%d task_timeout=%s)", interval, s.maxRangeDays(), s.batchSize(), s.taskTimeout())
	})
}

func (s *UsageCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.workerCancel != nil {
			s.workerCancel()
		}
		logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] stopped (%s)", usageCleanupWorkerName)
	})
}

func (s *UsageCleanupService) ListTasks(ctx context.Context, params pagination.PaginationParams) ([]UsageCleanupTask, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("cleanup service not ready")
	}
	return s.repo.ListTasks(ctx, params)
}

func (s *UsageCleanupService) CreateTask(ctx context.Context, filters UsageCleanupFilters, createdBy int64) (*UsageCleanupTask, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("cleanup service not ready")
	}
	if s.cfg != nil && !s.cfg.UsageCleanup.Enabled {
		return nil, infraerrors.New(http.StatusServiceUnavailable, "USAGE_CLEANUP_DISABLED", "usage cleanup is disabled")
	}
	if createdBy <= 0 {
		return nil, infraerrors.BadRequest("USAGE_CLEANUP_INVALID_CREATOR", "invalid creator")
	}

	logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] create_task requested: operator=%d %s", createdBy, describeUsageCleanupFilters(filters))
	sanitizeUsageCleanupFilters(&filters)
	if err := s.validateFilters(filters); err != nil {
		return nil, err
	}

	task := &UsageCleanupTask{
		Status:    UsageCleanupStatusPending,
		Filters:   filters,
		CreatedBy: createdBy,
	}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create cleanup task: %w", err)
	}
	go s.runOnce()
	return task, nil
}

func (s *UsageCleanupService) runOnce() {
	svc := s
	if svc == nil || svc.repo == nil {
		return
	}
	if !atomic.CompareAndSwapInt32(&svc.running, 0, 1) {
		logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] run_once skipped: already_running=true")
		return
	}
	defer atomic.StoreInt32(&svc.running, 0)

	parent := context.Background()
	if svc.workerCtx != nil {
		parent = svc.workerCtx
	}
	ctx, cancel := context.WithTimeout(parent, svc.taskTimeout())
	defer cancel()

	task, err := svc.repo.ClaimNextPendingTask(ctx, int64(svc.taskTimeout().Seconds()))
	if err != nil {
		logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] claim pending task failed: %v", err)
		return
	}
	if task == nil {
		slog.Debug("[UsageCleanup] run_once done: no_task=true")
		return
	}

	svc.executeTask(ctx, task)
}

func (s *UsageCleanupService) executeTask(ctx context.Context, task *UsageCleanupTask) {
	if s == nil || s.repo == nil || task == nil {
		return
	}

	batchSize := s.batchSize()
	deletedTotal := task.DeletedRows
	start := time.Now()
	var batchNum int

	for {
		if ctx != nil && ctx.Err() != nil {
			return
		}
		canceled, err := s.isTaskCanceled(ctx, task.ID)
		if err != nil {
			s.markTaskFailed(task.ID, deletedTotal, err)
			return
		}
		if canceled {
			return
		}

		batchNum++
		deleted, err := s.repo.DeleteUsageLogsBatch(ctx, task.Filters, batchSize)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			s.markTaskFailed(task.ID, deletedTotal, err)
			return
		}
		deletedTotal += deleted
		if deleted > 0 {
			updateCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = s.repo.UpdateTaskProgress(updateCtx, task.ID, deletedTotal)
			cancel()
		}
		if deleted == 0 || deleted < int64(batchSize) {
			break
		}
	}

	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.repo.MarkTaskSucceeded(updateCtx, task.ID, deletedTotal); err != nil {
		logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] update task succeeded failed: task=%d err=%v", task.ID, err)
		return
	}
	logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] task succeeded: task=%d deleted_rows=%d duration=%s batches=%d", task.ID, deletedTotal, time.Since(start), batchNum)

	if s.dashboard != nil {
		_ = s.dashboard.TriggerRecomputeRange(task.Filters.StartTime, task.Filters.EndTime)
	}
}

func (s *UsageCleanupService) markTaskFailed(taskID int64, deletedRows int64, err error) {
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 500 {
		msg = msg[:500]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if updateErr := s.repo.MarkTaskFailed(ctx, taskID, deletedRows, msg); updateErr != nil {
		logger.LegacyPrintf("service.usage_cleanup", "[UsageCleanup] update task failed failed: task=%d err=%v", taskID, updateErr)
	}
}

func (s *UsageCleanupService) isTaskCanceled(ctx context.Context, taskID int64) (bool, error) {
	if s == nil || s.repo == nil {
		return false, fmt.Errorf("cleanup service not ready")
	}
	checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	status, err := s.repo.GetTaskStatus(checkCtx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return status == UsageCleanupStatusCanceled, nil
}

func (s *UsageCleanupService) validateFilters(filters UsageCleanupFilters) error {
	if filters.StartTime.IsZero() || filters.EndTime.IsZero() {
		return infraerrors.BadRequest("USAGE_CLEANUP_MISSING_RANGE", "start_date and end_date are required")
	}
	if filters.EndTime.Before(filters.StartTime) {
		return infraerrors.BadRequest("USAGE_CLEANUP_INVALID_RANGE", "end_date must be after start_date")
	}
	maxDays := s.maxRangeDays()
	if maxDays > 0 {
		delta := filters.EndTime.Sub(filters.StartTime)
		if delta > time.Duration(maxDays)*24*time.Hour {
			return infraerrors.BadRequest("USAGE_CLEANUP_RANGE_TOO_LARGE", fmt.Sprintf("date range exceeds %d days", maxDays))
		}
	}
	return nil
}

func (s *UsageCleanupService) CancelTask(ctx context.Context, taskID int64, canceledBy int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("cleanup service not ready")
	}
	if s.cfg != nil && !s.cfg.UsageCleanup.Enabled {
		return infraerrors.New(http.StatusServiceUnavailable, "USAGE_CLEANUP_DISABLED", "usage cleanup is disabled")
	}
	if canceledBy <= 0 {
		return infraerrors.BadRequest("USAGE_CLEANUP_INVALID_CANCELLER", "invalid canceller")
	}
	status, err := s.repo.GetTaskStatus(ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return infraerrors.New(http.StatusNotFound, "USAGE_CLEANUP_TASK_NOT_FOUND", "cleanup task not found")
		}
		return err
	}
	if status == UsageCleanupStatusCanceled {
		return nil
	}
	if status != UsageCleanupStatusPending && status != UsageCleanupStatusRunning {
		return infraerrors.New(http.StatusConflict, "USAGE_CLEANUP_CANCEL_CONFLICT", "cleanup task cannot be canceled in current status")
	}
	ok, err := s.repo.CancelTask(ctx, taskID, canceledBy)
	if err != nil {
		return err
	}
	if !ok {
		currentStatus, getErr := s.repo.GetTaskStatus(ctx, taskID)
		if getErr == nil && currentStatus == UsageCleanupStatusCanceled {
			return nil
		}
		return infraerrors.New(http.StatusConflict, "USAGE_CLEANUP_CANCEL_CONFLICT", "cleanup task cannot be canceled in current status")
	}
	return nil
}

func sanitizeUsageCleanupFilters(filters *UsageCleanupFilters) {
	if filters == nil {
		return
	}
	if filters.UserID != nil && *filters.UserID <= 0 {
		filters.UserID = nil
	}
	if filters.APIKeyID != nil && *filters.APIKeyID <= 0 {
		filters.APIKeyID = nil
	}
	if filters.AccountID != nil && *filters.AccountID <= 0 {
		filters.AccountID = nil
	}
	if filters.GroupID != nil && *filters.GroupID <= 0 {
		filters.GroupID = nil
	}
	if filters.Operation != nil {
		operation := strings.TrimSpace(*filters.Operation)
		if operation == "" {
			filters.Operation = nil
		} else {
			filters.Operation = &operation
		}
	}
	if filters.Status != nil {
		status := strings.TrimSpace(*filters.Status)
		if status == "" {
			filters.Status = nil
		} else {
			filters.Status = &status
		}
	}
}

func (s *UsageCleanupService) maxRangeDays() int {
	if s == nil || s.cfg == nil {
		return 31
	}
	if s.cfg.UsageCleanup.MaxRangeDays > 0 {
		return s.cfg.UsageCleanup.MaxRangeDays
	}
	return 31
}

func (s *UsageCleanupService) batchSize() int {
	if s == nil || s.cfg == nil {
		return 5000
	}
	if s.cfg.UsageCleanup.BatchSize > 0 {
		return s.cfg.UsageCleanup.BatchSize
	}
	return 5000
}

func (s *UsageCleanupService) workerInterval() time.Duration {
	if s == nil || s.cfg == nil {
		return 10 * time.Second
	}
	if s.cfg.UsageCleanup.WorkerIntervalSeconds > 0 {
		return time.Duration(s.cfg.UsageCleanup.WorkerIntervalSeconds) * time.Second
	}
	return 10 * time.Second
}

func (s *UsageCleanupService) taskTimeout() time.Duration {
	if s == nil || s.cfg == nil {
		return 30 * time.Minute
	}
	if s.cfg.UsageCleanup.TaskTimeoutSeconds > 0 {
		return time.Duration(s.cfg.UsageCleanup.TaskTimeoutSeconds) * time.Second
	}
	return 30 * time.Minute
}
