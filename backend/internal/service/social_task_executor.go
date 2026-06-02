package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
)

// SocialTaskAction constants.
const (
	SocialTaskActionLoginCheck = "login_check"
	SocialTaskActionFollow     = "follow"
	SocialTaskActionMessage    = "message"
	SocialTaskActionPost       = "post"
	SocialTaskActionLike       = "like"

	legacySocialTaskActionDM    = "dm"
	legacySocialTaskActionTweet = "tweet"
)

// SocialTaskLogStatus constants for execution
const (
	SocialTaskLogStatusPending = "pending"
	SocialTaskLogStatusRunning = "running"
	SocialTaskLogStatusSuccess = "success"
	SocialTaskLogStatusFailed  = "failed"
)

// SocialTaskExecutor handles async execution of social tasks.
type SocialTaskExecutor struct {
	entClient   *dbent.Client
	billing     *SocialBillingService
	workerCount int
	taskCh      chan int64 // task log IDs to process
	stopCh      chan struct{}
	wg          sync.WaitGroup
	minInterval time.Duration // minimum interval between operations per account
	maxRetries  int
}

// SocialTaskExecutorConfig holds configuration for the task executor.
type SocialTaskExecutorConfig struct {
	WorkerCount   int
	QueueSize     int
	MinIntervalMs int
	MaxRetries    int
}

// NewSocialTaskExecutor creates a new task executor.
func NewSocialTaskExecutor(entClient *dbent.Client, billing *SocialBillingService, cfg SocialTaskExecutorConfig) *SocialTaskExecutor {
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 3
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}
	if cfg.MinIntervalMs <= 0 {
		cfg.MinIntervalMs = 2000
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 2
	}
	return &SocialTaskExecutor{
		entClient:   entClient,
		billing:     billing,
		workerCount: cfg.WorkerCount,
		taskCh:      make(chan int64, cfg.QueueSize),
		stopCh:      make(chan struct{}),
		minInterval: time.Duration(cfg.MinIntervalMs) * time.Millisecond,
		maxRetries:  cfg.MaxRetries,
	}
}

// Start launches the worker goroutines.
func (e *SocialTaskExecutor) Start() {
	for i := 0; i < e.workerCount; i++ {
		e.wg.Add(1)
		go e.worker(i)
	}
	slog.Info("social task executor started", "workers", e.workerCount)
}

// Stop gracefully shuts down the executor.
func (e *SocialTaskExecutor) Stop() {
	close(e.stopCh)
	e.wg.Wait()
	slog.Info("social task executor stopped")
}

// Enqueue adds a task log ID to the processing queue.
// Returns false if the queue is full.
func (e *SocialTaskExecutor) Enqueue(taskLogID int64) bool {
	select {
	case e.taskCh <- taskLogID:
		return true
	default:
		return false
	}
}

// EnqueueBatch enqueues multiple task log IDs.
// Returns the number successfully enqueued.
func (e *SocialTaskExecutor) EnqueueBatch(taskLogIDs []int64) int {
	enqueued := 0
	for _, id := range taskLogIDs {
		if e.Enqueue(id) {
			enqueued++
		}
	}
	return enqueued
}

func (e *SocialTaskExecutor) worker(id int) {
	defer e.wg.Done()
	slog.Debug("social task worker started", "worker_id", id)

	for {
		select {
		case <-e.stopCh:
			return
		case taskLogID := <-e.taskCh:
			e.processTask(taskLogID)
			// Rate limiting: wait between tasks
			time.Sleep(e.minInterval)
		}
	}
}

func (e *SocialTaskExecutor) processTask(taskLogID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Mark as running only from pending so duplicate enqueues cannot double-charge.
	updated, err := e.entClient.SocialTaskLog.Update().
		Where(
			socialtasklog.IDEQ(taskLogID),
			socialtasklog.StatusEQ(SocialTaskLogStatusPending),
		).
		SetStatus(SocialTaskLogStatusRunning).
		Save(ctx)
	if err != nil {
		slog.Error("failed to mark task as running", "task_log_id", taskLogID, "error", err)
		return
	}
	if updated == 0 {
		slog.Debug("social task was already claimed", "task_log_id", taskLogID)
		return
	}
	taskLog, err := e.entClient.SocialTaskLog.Get(ctx, taskLogID)
	if err != nil {
		slog.Error("failed to load claimed task", "task_log_id", taskLogID, "error", err)
		return
	}

	// Execute the action
	result, execErr := e.executeAction(ctx, taskLog)

	// Update status based on result
	now := time.Now()
	update := e.entClient.SocialTaskLog.UpdateOneID(int64(taskLogID)).
		SetExecutedAt(now)

	if execErr != nil {
		errMsg := execErr.Error()
		update.SetStatus(SocialTaskLogStatusFailed).
			SetResultMessage(errMsg).
			SetChargedAmount(0).
			SetChargeStatus(SocialTaskChargeStatusNotCharged).
			ClearChargeSource()
		slog.Warn("social task failed", "task_log_id", taskLogID, "action", taskLog.Action, "error", errMsg)
	} else {
		charge, chargeErr := e.finalizeSuccessfulTask(ctx, taskLogID, taskLog.UserID, taskLog.Price, result)
		if chargeErr != nil {
			errMsg := fmt.Sprintf("execution succeeded but billing failed: %v", chargeErr)
			update.SetStatus(SocialTaskLogStatusFailed).
				SetResultMessage(errMsg).
				SetChargedAmount(0).
				SetChargeStatus(SocialTaskChargeStatusNotCharged).
				ClearChargeSource().
				ClearBillingRequestID()
			slog.Error("social task billing failed after execution", "task_log_id", taskLogID, "action", taskLog.Action, "error", chargeErr)
		} else {
			slog.Info("social task completed", "task_log_id", taskLogID, "action", taskLog.Action, "charged_amount", charge.Amount, "charge_source", charge.Source)
			return
		}
	}

	if _, err := update.Save(ctx); err != nil {
		slog.Error("failed to update task result", "task_log_id", taskLogID, "error", err)
	}
}

func (e *SocialTaskExecutor) executeAction(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	switch taskLog.Action {
	case SocialTaskActionLoginCheck:
		return e.doLoginCheck(ctx, taskLog)
	case SocialTaskActionFollow:
		return e.doFollow(ctx, taskLog)
	case SocialTaskActionMessage:
		return e.doMessage(ctx, taskLog)
	case SocialTaskActionPost:
		return e.doPost(ctx, taskLog)
	case SocialTaskActionLike:
		return e.doLike(ctx, taskLog)
	default:
		return "", fmt.Errorf("unsupported action: %s", taskLog.Action)
	}
}

// --- Action implementations ---

func unsupportedSocialAction(action string) (string, error) {
	return "", fmt.Errorf("%s is not configured: social platform executor is not available", action)
}

func (e *SocialTaskExecutor) doLoginCheck(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	return unsupportedSocialAction(taskLog.Action)
}

func (e *SocialTaskExecutor) doMessage(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog.Target == nil || *taskLog.Target == "" {
		return "", fmt.Errorf("message target is required")
	}
	if taskLog.Content == nil || *taskLog.Content == "" {
		return "", fmt.Errorf("message content is required")
	}
	return unsupportedSocialAction(taskLog.Action)
}

func (e *SocialTaskExecutor) doFollow(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog.Target == nil || *taskLog.Target == "" {
		return "", fmt.Errorf("follow target is required")
	}
	return unsupportedSocialAction(taskLog.Action)
}

func (e *SocialTaskExecutor) doPost(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog.Content == nil || *taskLog.Content == "" {
		return "", fmt.Errorf("post content is required")
	}
	return unsupportedSocialAction(taskLog.Action)
}

func (e *SocialTaskExecutor) doLike(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog.Target == nil || *taskLog.Target == "" {
		return "", fmt.Errorf("like target (post URL/ID) is required")
	}
	return unsupportedSocialAction(taskLog.Action)
}

// ProcessPendingTasks scans for pending tasks and enqueues them.
// This can be called periodically or on-demand.
func (e *SocialTaskExecutor) ProcessPendingTasks(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	tasks, err := e.entClient.SocialTaskLog.Query().
		Where(socialtasklog.StatusEQ(SocialTaskLogStatusPending)).
		Order(dbent.Asc(socialtasklog.FieldCreatedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return 0, err
	}

	ids := make([]int64, len(tasks))
	for i, t := range tasks {
		ids[i] = int64(t.ID)
	}
	return e.EnqueueBatch(ids), nil
}

func (e *SocialTaskExecutor) finalizeSuccessfulTask(ctx context.Context, taskLogID, userID int64, amount float64, result string) (*SocialBillingChargeResult, error) {
	if e == nil || e.billing == nil || e.entClient == nil {
		return nil, fmt.Errorf("social billing finalizer is unavailable")
	}
	return e.billing.FinalizeSuccessfulTask(ctx, e.entClient, taskLogID, userID, amount, result)
}
