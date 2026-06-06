package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

// SocialTaskAction constants.
const (
	SocialTaskActionLoginCheck = "login_check"
	SocialTaskActionFollow     = "follow"
	SocialTaskActionMessage    = "message"
	SocialTaskActionPost       = "post"
	SocialTaskActionLike       = "like"
	SocialTaskActionRetweet    = "retweet"

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

const socialTaskRunningRecoveryTimeout = 2 * time.Minute

// SocialTaskExecutor handles async execution of social tasks.
type SocialTaskExecutor struct {
	entClient   *dbent.Client
	billing     *SocialBillingService
	workerCount int
	taskCh      chan int64 // task log IDs to process
	stopCh      chan struct{}
	wg          sync.WaitGroup
	stopOnce    sync.Once
	stopped     atomic.Bool
	minInterval time.Duration // minimum interval between operations per account
	maxRetries  int
	executors   map[string]SocialPlatformExecutor
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
		executors:   make(map[string]SocialPlatformExecutor),
	}
}

// RegisterPlatformExecutor attaches a real platform adapter to the task worker.
func (e *SocialTaskExecutor) RegisterPlatformExecutor(platform string, executor SocialPlatformExecutor) {
	if e == nil || executor == nil {
		return
	}
	if e.executors == nil {
		e.executors = make(map[string]SocialPlatformExecutor)
	}
	platform = normalizeSocialPlatform(platform)
	if platform == "" {
		return
	}
	e.executors[platform] = executor
	if isTwitterPlatform(platform) {
		e.executors["x_twitter"] = executor
		e.executors["twitter"] = executor
		e.executors["x"] = executor
	}
}

// Start launches the worker goroutines.
func (e *SocialTaskExecutor) Start() {
	for i := 0; i < e.workerCount; i++ {
		e.wg.Add(1)
		go e.worker(i)
	}
	e.wg.Add(1)
	go e.pendingRecoveryLoop()
	slog.Info("social task executor started", "workers", e.workerCount)
}

// Stop gracefully shuts down the executor.
func (e *SocialTaskExecutor) Stop() {
	if e == nil {
		return
	}
	e.stopOnce.Do(func() {
		e.stopped.Store(true)
		close(e.stopCh)
		e.wg.Wait()
		slog.Info("social task executor stopped")
	})
}

// Enqueue adds a task log ID to the processing queue.
// Returns false if the queue is full.
func (e *SocialTaskExecutor) Enqueue(taskLogID int64) bool {
	if e == nil || e.stopped.Load() {
		return false
	}
	select {
	case e.taskCh <- taskLogID:
		return true
	default:
		return false
	}
}

func (e *SocialTaskExecutor) isStopped() bool {
	return e == nil || e.stopped.Load()
}

// EnqueueBatch enqueues multiple task log IDs.
// Returns the number successfully enqueued and the exact IDs that were rejected.
func (e *SocialTaskExecutor) EnqueueBatch(taskLogIDs []int64) (int, []int64) {
	return enqueueSocialTaskBatch(taskLogIDs, e.Enqueue)
}

func enqueueSocialTaskBatch(taskLogIDs []int64, enqueue func(int64) bool) (int, []int64) {
	enqueued := 0
	failed := make([]int64, 0)
	for _, id := range taskLogIDs {
		if enqueue != nil && enqueue(id) {
			enqueued++
			continue
		}
		failed = append(failed, id)
	}
	return enqueued, failed
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

func (e *SocialTaskExecutor) pendingRecoveryLoop() {
	defer e.wg.Done()
	e.recoverPendingTasks()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.recoverPendingTasks()
		}
	}
}

func (e *SocialTaskExecutor) recoverPendingTasks() {
	if e == nil || e.isStopped() {
		return
	}
	if failed, err := e.failStaleRunningTasks(context.Background(), time.Now().Add(-socialTaskRunningRecoveryTimeout)); err != nil {
		slog.Warn("failed to recover stale running social tasks", "error", err)
	} else if failed > 0 {
		slog.Warn("failed stale running social tasks without charge", "failed", failed)
	}
	limit := cap(e.taskCh)
	if limit <= 0 {
		limit = 50
	}
	enqueued, err := e.ProcessPendingTasks(context.Background(), limit)
	if err != nil {
		slog.Warn("failed to recover pending social tasks", "error", err)
		return
	}
	if enqueued > 0 {
		slog.Info("recovered pending social tasks", "enqueued", enqueued)
	}
}

func (e *SocialTaskExecutor) failStaleRunningTasks(ctx context.Context, staleBefore time.Time) (int, error) {
	if e == nil || e.entClient == nil {
		return 0, nil
	}
	now := time.Now()
	message := "任务执行超时，本次未扣费"
	return e.entClient.SocialTaskLog.Update().
		Where(
			socialtasklog.StatusEQ(SocialTaskLogStatusRunning),
			socialtasklog.ChargeStatusEQ(SocialTaskChargeStatusNotCharged),
			socialtasklog.UpdatedAtLTE(staleBefore),
		).
		SetStatus(SocialTaskLogStatusFailed).
		SetResultMessage(message).
		SetExecutedAt(now).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		ClearChargeSource().
		ClearBillingRequestID().
		Save(ctx)
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

	result, execErr := e.executeActionSafely(ctx, taskLog)

	// Update status based on result
	now := time.Now()
	update := e.entClient.SocialTaskLog.UpdateOneID(int64(taskLogID)).
		SetExecutedAt(now)

	if execErr != nil {
		e.recordAccountExecutionOutcome(ctx, taskLog, result, execErr)
		errMsg := safeSocialTaskFailureMessage(execErr)
		update.SetStatus(SocialTaskLogStatusFailed).
			SetResultMessage(errMsg).
			SetChargedAmount(0).
			SetChargeStatus(SocialTaskChargeStatusNotCharged).
			ClearChargeSource().
			ClearBillingRequestID()
		slog.Warn("social task failed", "task_log_id", taskLogID, "action", taskLog.Action, "error", errMsg)
	} else {
		charge, chargeErr := e.finalizeSuccessfulTask(ctx, taskLogID, taskLog.UserID, taskLog.Price, result)
		if chargeErr != nil {
			errMsg := safeSocialTaskFailureMessage(fmt.Errorf("execution succeeded but billing failed: %w", chargeErr))
			update.SetStatus(SocialTaskLogStatusFailed).
				SetResultMessage(errMsg).
				SetChargedAmount(0).
				SetChargeStatus(SocialTaskChargeStatusNotCharged).
				ClearChargeSource().
				ClearBillingRequestID()
			slog.Error("social task billing failed after execution", "task_log_id", taskLogID, "action", taskLog.Action, "error", chargeErr)
		} else {
			e.recordAccountExecutionOutcome(ctx, taskLog, result, nil)
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
	case SocialTaskActionRetweet:
		return e.doRetweet(ctx, taskLog)
	default:
		return "", fmt.Errorf("unsupported action: %s", taskLog.Action)
	}
}

func (e *SocialTaskExecutor) executeActionSafely(ctx context.Context, taskLog *dbent.SocialTaskLog) (result string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			taskLogID := int64(0)
			if taskLog != nil {
				taskLogID = taskLog.ID
			}
			slog.Error("social platform executor panicked", "task_log_id", taskLogID, "panic", fmt.Sprint(recovered))
			result = ""
			err = newSocialExecutionError(SocialExecutionFailurePlatform, "social platform executor failed unexpectedly", nil)
		}
	}()
	return e.executeAction(ctx, taskLog)
}

// --- Action implementations ---

func unsupportedSocialAction(action string) (string, error) {
	return "", fmt.Errorf("%s is not configured: social platform executor is not available", action)
}

func (e *SocialTaskExecutor) executePlatformAction(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if e == nil || e.entClient == nil || taskLog == nil {
		return unsupportedSocialAction("")
	}
	account, err := e.entClient.SocialAccount.Get(ctx, taskLog.SocialAccountID)
	if err != nil {
		slog.Warn("social task account load failed", "task_log_id", taskLog.ID, "account_id", taskLog.SocialAccountID, "error", err)
		return "", fmt.Errorf("social account is unavailable")
	}
	if err := validateTaskExecutionAccountScope(taskLog, account); err != nil {
		return "", err
	}
	platform := normalizeSocialPlatform(account.PlatformKey)
	if platform == "" {
		platform = normalizeSocialPlatform(account.Platform)
	}
	executor := e.executors[platform]
	if executor == nil {
		return unsupportedSocialAction(taskLog.Action)
	}
	return executor.Execute(ctx, taskLog, account)
}

func validateTaskExecutionAccountScope(taskLog *dbent.SocialTaskLog, account *dbent.SocialAccount) error {
	if taskLog == nil || account == nil {
		return fmt.Errorf("social account is unavailable")
	}
	if account.AssignedUserID == nil || int64(*account.AssignedUserID) != taskLog.UserID {
		return fmt.Errorf("social account is unavailable")
	}
	if account.UserWorkbenchDeletedAt != nil {
		return fmt.Errorf("social account is unavailable")
	}
	if account.AccountStatus != SocialAccountStatusAvailable {
		return fmt.Errorf("social account is unavailable")
	}
	return nil
}

func (e *SocialTaskExecutor) doLoginCheck(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doMessage(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog.Target == nil || *taskLog.Target == "" {
		return "", fmt.Errorf("message target is required")
	}
	if taskLog.Content == nil || *taskLog.Content == "" {
		return "", fmt.Errorf("message content is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doFollow(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog.Target == nil || strings.TrimSpace(*taskLog.Target) == "" {
		return "", fmt.Errorf("follow target is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doPost(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog.Content == nil || strings.TrimSpace(*taskLog.Content) == "" {
		return "", fmt.Errorf("post content is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doLike(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog.Target == nil || strings.TrimSpace(*taskLog.Target) == "" {
		return "", fmt.Errorf("like target (post URL/ID) is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doRetweet(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog.Target == nil || strings.TrimSpace(*taskLog.Target) == "" {
		return "", fmt.Errorf("retweet target (post URL/ID) is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

// ProcessPendingTasks scans for pending tasks and enqueues them.
// This can be called periodically or on-demand.
func (e *SocialTaskExecutor) ProcessPendingTasks(ctx context.Context, limit int) (int, error) {
	if e == nil || e.entClient == nil {
		return 0, infraerrors.ServiceUnavailable("SOCIAL_TASK_EXECUTOR_UNAVAILABLE", "social task executor is unavailable")
	}
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
	if len(ids) == 0 {
		return 0, nil
	}
	if e.taskCh == nil || e.isStopped() {
		message := "social platform executor queue is not configured; task was not charged"
		if err := e.markPendingTasksFailedNotCharged(ctx, ids, message); err != nil {
			return 0, err
		}
		return 0, nil
	}
	enqueued, failedIDs := e.EnqueueBatch(ids)
	if len(failedIDs) > 0 {
		message := "social platform executor queue is full; task was not charged"
		if err := e.markPendingTasksFailedNotCharged(ctx, failedIDs, message); err != nil {
			return enqueued, err
		}
	}
	return enqueued, nil
}

func (e *SocialTaskExecutor) markPendingTasksFailedNotCharged(ctx context.Context, taskLogIDs []int64, message string) error {
	now := time.Now()
	for _, taskLogID := range taskLogIDs {
		if _, err := e.entClient.SocialTaskLog.Update().
			Where(
				socialtasklog.IDEQ(taskLogID),
				socialtasklog.StatusEQ(SocialTaskLogStatusPending),
			).
			SetStatus(SocialTaskLogStatusFailed).
			SetResultMessage(message).
			SetExecutedAt(now).
			SetChargedAmount(0).
			SetChargeStatus(SocialTaskChargeStatusNotCharged).
			ClearChargeSource().
			ClearBillingRequestID().
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (e *SocialTaskExecutor) finalizeSuccessfulTask(ctx context.Context, taskLogID, userID int64, amount float64, result string) (*SocialBillingChargeResult, error) {
	if e == nil || e.billing == nil || e.entClient == nil {
		return nil, fmt.Errorf("social billing finalizer is unavailable")
	}
	return e.billing.FinalizeSuccessfulTask(ctx, e.entClient, taskLogID, userID, amount, result)
}

func (e *SocialTaskExecutor) recordAccountExecutionOutcome(ctx context.Context, taskLog *dbent.SocialTaskLog, result string, execErr error) {
	if e == nil || e.entClient == nil || taskLog == nil || taskLog.SocialAccountID <= 0 {
		return
	}
	if execErr == nil {
		update := e.entClient.SocialAccount.UpdateOneID(taskLog.SocialAccountID).
			SetAccountStatus(SocialAccountStatusAvailable).
			SetTaskStatus(SocialTaskStatusStored)
		if strings.TrimSpace(result) != "" {
			update.SetTaskMessage(result)
		} else {
			update.ClearTaskMessage()
		}
		if _, err := update.Save(ctx); err != nil {
			slog.Warn("failed to update social account success state", "task_log_id", taskLog.ID, "account_id", taskLog.SocialAccountID, "error", err)
		}
		return
	}

	kind, ok := socialExecutionFailureKind(execErr)
	if !ok {
		return
	}
	accountStatus, taskStatus, shouldUpdate := socialAccountStateForExecutionFailure(kind)
	if !shouldUpdate {
		return
	}
	update := e.entClient.SocialAccount.UpdateOneID(taskLog.SocialAccountID).
		SetTaskMessage(safeSocialTaskFailureMessage(execErr))
	if accountStatus != "" {
		update.SetAccountStatus(accountStatus)
	}
	if taskStatus != "" {
		update.SetTaskStatus(taskStatus)
	}
	if _, err := update.Save(ctx); err != nil {
		slog.Warn(
			"failed to update social account failure state",
			"task_log_id", taskLog.ID,
			"account_id", taskLog.SocialAccountID,
			"failure_kind", kind,
			"error", err,
		)
	}
}

func safeSocialTaskFailureMessage(err error) string {
	kind, ok := socialExecutionFailureKind(err)
	if ok {
		switch kind {
		case SocialExecutionFailureAuthMissing, SocialExecutionFailureAuthInvalid:
			return "账号认证信息不可用，本次未扣费"
		case SocialExecutionFailureProxyMissing, SocialExecutionFailureProxyInvalid, SocialExecutionFailureProxyUnavailable, SocialExecutionFailureNetwork:
			return "执行代理不可用，本次未扣费"
		case SocialExecutionFailureUnsupported:
			return "该动作暂不支持，本次未扣费"
		case SocialExecutionFailureActionInput:
			return "任务参数不完整，本次未扣费"
		case SocialExecutionFailureAccountLimited, SocialExecutionFailureChallengeRequired:
			return "账号状态或频率受限，本次未扣费"
		case SocialExecutionFailurePlatform:
			return "该平台动作暂不可用，本次未扣费"
		}
	}
	message := strings.ToLower(strings.TrimSpace(fmt.Sprint(err)))
	switch {
	case strings.Contains(message, "queue is full"):
		return "任务队列繁忙，本次未扣费"
	case strings.Contains(message, "billing failed"):
		return "执行已完成，但扣费确认异常，请联系管理员处理"
	case strings.Contains(message, "not configured") || strings.Contains(message, "executor is not available") || strings.Contains(message, "executor queue"):
		return "该平台动作暂不可用，本次未扣费"
	case strings.Contains(message, "auth cookie") || strings.Contains(message, "authentication failed") || strings.Contains(message, "oauth") || strings.Contains(message, "token"):
		return "账号认证信息不可用，本次未扣费"
	case strings.Contains(message, "proxy"):
		return "执行代理不可用，本次未扣费"
	case strings.Contains(message, "unsupported action"):
		return "该动作暂不支持，本次未扣费"
	case strings.Contains(message, "target is required") || strings.Contains(message, "content is required"):
		return "任务参数不完整，本次未扣费"
	case strings.Contains(message, "target not found"):
		return "执行目标不存在，本次未扣费"
	case strings.Contains(message, "rate limit") || strings.Contains(message, "too frequent") || strings.Contains(message, "follow limit") || strings.Contains(message, "suspended") || strings.Contains(message, "locked"):
		return "账号状态或频率受限，本次未扣费"
	case strings.Contains(message, "content is too long") || strings.Contains(message, "duplicate") || strings.Contains(message, "already") || strings.Contains(message, "restricted"):
		return "内容或目标状态不符合平台要求，本次未扣费"
	case strings.Contains(message, "access denied"):
		return "平台拒绝执行，本次未扣费"
	default:
		return "任务执行失败，本次未扣费"
	}
}

func socialAccountStateForExecutionFailure(kind SocialExecutionFailureKind) (accountStatus, taskStatus string, shouldUpdate bool) {
	switch kind {
	case SocialExecutionFailureAuthMissing:
		return SocialAccountStatusNotStored, SocialTaskStatusManualReview, true
	case SocialExecutionFailureAuthInvalid:
		return SocialAccountStatusInvalid, SocialTaskStatusManualReview, true
	case SocialExecutionFailureAccountLimited:
		return SocialAccountStatusLimited, SocialTaskStatusManualReview, true
	case SocialExecutionFailureChallengeRequired:
		return SocialAccountStatusPendingCheck, SocialTaskStatusManualReview, true
	case SocialExecutionFailureProxyMissing, SocialExecutionFailureProxyInvalid, SocialExecutionFailureProxyUnavailable, SocialExecutionFailureNetwork:
		return "", SocialTaskStatusIPUnavailable, true
	default:
		return "", "", false
	}
}
