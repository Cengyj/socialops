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
	SocialTaskActionLogin         = "login"
	SocialTaskActionLoginCheck    = "login_check"
	SocialTaskActionFollow        = "follow"
	SocialTaskActionPost          = "post"
	SocialTaskActionLike          = "like"
	SocialTaskActionRetweet       = "retweet"
	SocialTaskActionUpdateProfile = "update_profile"
	SocialTaskActionUpdateAvatar  = "update_avatar"
	SocialTaskActionUpdateBanner  = "update_banner"
)

// SocialTaskLogStatus constants for execution
const (
	SocialTaskLogStatusPending = "pending"
	SocialTaskLogStatusRunning = "running"
	SocialTaskLogStatusSuccess = "success"
	SocialTaskLogStatusFailed  = "failed"
)

const (
	socialTaskExecutionTimeout       = 60 * time.Second
	socialTaskRunningRecoveryTimeout = 2 * time.Minute
)

// SocialTaskExecutor handles async execution of social tasks.
type SocialTaskExecutor struct {
	entClient           *dbent.Client
	billing             *SocialBillingService
	proxyChecker        *SocialIPChecker
	workerCount         int
	taskCh              chan int64 // task log IDs to process
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	stopOnce            sync.Once
	stopped             atomic.Bool
	minInterval         time.Duration // minimum interval between operations per account
	maxRetries          int
	executors           map[string]SocialPlatformExecutor
	credentialEncryptor ExecutionAuthEncryptor
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
		entClient:    entClient,
		billing:      billing,
		proxyChecker: NewSocialIPChecker(entClient),
		workerCount:  cfg.WorkerCount,
		taskCh:       make(chan int64, cfg.QueueSize),
		stopCh:       make(chan struct{}),
		minInterval:  time.Duration(cfg.MinIntervalMs) * time.Millisecond,
		maxRetries:   cfg.MaxRetries,
		executors:    make(map[string]SocialPlatformExecutor),
	}
}

func (e *SocialTaskExecutor) WithCredentialEncryptor(encryptor ExecutionAuthEncryptor) *SocialTaskExecutor {
	if e == nil {
		return nil
	}
	e.credentialEncryptor = encryptor
	return e
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
	message := "任务执行超时，本次未扣费"
	tasks, err := e.entClient.SocialTaskLog.Query().
		Where(
			socialtasklog.StatusEQ(SocialTaskLogStatusRunning),
			socialtasklog.ChargeStatusEQ(SocialTaskChargeStatusNotCharged),
			socialtasklog.UpdatedAtLTE(staleBefore),
		).
		Order(dbent.Asc(socialtasklog.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		return 0, err
	}
	accounts := NewSocialAccountServiceWithCredentialEncryptor(e.entClient, e.credentialEncryptor)
	failed := 0
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if _, err := accounts.MarkStaleRunningTaskLogFailedNotCharged(ctx, int64(task.ID), staleBefore, message); err != nil {
			if infraerrors.Reason(err) == "SOCIAL_TASK_LOG_FINALIZED" || infraerrors.Reason(err) == "SOCIAL_TASK_LOG_NOT_FOUND" {
				continue
			}
			return failed, err
		}
		failed++
	}
	return failed, nil
}

func (e *SocialTaskExecutor) processTask(taskLogID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), socialTaskExecutionTimeout)
	defer cancel()

	// Mark as running only from pending so duplicate enqueues cannot double-charge.
	updated, err := e.entClient.SocialTaskLog.Update().
		Where(
			socialtasklog.IDEQ(taskLogID),
			socialtasklog.StatusEQ(SocialTaskLogStatusPending),
			socialtasklog.ChargeStatusEQ(SocialTaskChargeStatusNotCharged),
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

	now := time.Now()
	if execErr != nil {
		e.refreshProxyStatusAfterNetworkFailure(taskLog, execErr)
		errMsg := safeSocialTaskFailureMessage(execErr)
		if err := e.markRunningTaskFailedNotCharged(ctx, taskLogID, errMsg, now); err != nil {
			slog.Error("failed to update failed task result", "task_log_id", taskLogID, "error", err)
			return
		}
		e.recordAccountExecutionOutcome(ctx, taskLog, result, execErr)
		slog.Warn("social task failed", "task_log_id", taskLogID, "action", taskLog.Action, "error", errMsg)
		return
	} else {
		charge, chargeErr := e.finalizeSuccessfulTask(ctx, taskLogID, taskLog.UserID, taskLog.Price, result)
		if chargeErr != nil {
			errMsg := safeSocialTaskFailureMessage(fmt.Errorf("execution succeeded but billing failed: %w", chargeErr))
			if err := e.markRunningTaskFailedNotCharged(ctx, taskLogID, errMsg, now); err != nil {
				slog.Error("failed to update billing-failed task result", "task_log_id", taskLogID, "error", err)
				return
			}
			e.recordAccountBillingFailure(ctx, taskLog, errMsg)
			slog.Error("social task billing failed after execution", "task_log_id", taskLogID, "action", taskLog.Action, "error", chargeErr)
		} else {
			e.recordAccountExecutionOutcome(ctx, taskLog, result, nil)
			slog.Info("social task completed", "task_log_id", taskLogID, "action", taskLog.Action, "charged_amount", charge.Amount, "charge_source", charge.Source)
			return
		}
	}
}

func (e *SocialTaskExecutor) markRunningTaskFailedNotCharged(ctx context.Context, taskLogID int64, message string, executedAt time.Time) error {
	if e == nil || e.entClient == nil {
		return fmt.Errorf("social task executor is unavailable")
	}
	updated, err := e.entClient.SocialTaskLog.Update().
		Where(
			socialtasklog.IDEQ(taskLogID),
			socialtasklog.StatusEQ(SocialTaskLogStatusRunning),
			socialtasklog.ChargeStatusEQ(SocialTaskChargeStatusNotCharged),
		).
		SetStatus(SocialTaskLogStatusFailed).
		SetResultMessage(message).
		SetExecutedAt(executedAt).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		ClearChargeSource().
		ClearBillingRequestID().
		Save(ctx)
	if err != nil {
		return err
	}
	if updated == 0 {
		return fmt.Errorf("social task log is unavailable")
	}
	return nil
}

func (e *SocialTaskExecutor) refreshProxyStatusAfterNetworkFailure(taskLog *dbent.SocialTaskLog, execErr error) {
	if e == nil || e.entClient == nil || taskLog == nil || taskLog.ProxyID == nil {
		return
	}
	kind, ok := socialExecutionFailureKind(execErr)
	if !ok || kind != SocialExecutionFailureNetwork {
		return
	}
	checker := e.proxyChecker
	if checker == nil {
		checker = NewSocialIPChecker(e.entClient)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := checker.TestIP(ctx, int64(*taskLog.ProxyID)); err != nil {
		slog.Warn(
			"failed to refresh proxy status after social task network failure",
			"task_log_id", taskLog.ID,
			"proxy_id", *taskLog.ProxyID,
			"error", err,
		)
	}
}

func (e *SocialTaskExecutor) executeAction(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog == nil {
		return "", fmt.Errorf("social task log is unavailable")
	}
	switch taskLog.Action {
	case SocialTaskActionLogin:
		return e.doLogin(ctx, taskLog)
	case SocialTaskActionLoginCheck:
		return e.doLoginCheck(ctx, taskLog)
	case SocialTaskActionFollow:
		return e.doFollow(ctx, taskLog)
	case SocialTaskActionPost:
		return e.doPost(ctx, taskLog)
	case SocialTaskActionLike:
		return e.doLike(ctx, taskLog)
	case SocialTaskActionRetweet:
		return e.doRetweet(ctx, taskLog)
	case SocialTaskActionUpdateProfile:
		return e.doUpdateProfile(ctx, taskLog)
	case SocialTaskActionUpdateAvatar:
		return e.doUpdateAvatar(ctx, taskLog)
	case SocialTaskActionUpdateBanner:
		return e.doUpdateBanner(ctx, taskLog)
	default:
		return "", newSocialExecutionError(SocialExecutionFailureUnsupported, fmt.Sprintf("unsupported action: %s", taskLog.Action), nil)
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
	return "", newSocialExecutionError(SocialExecutionFailurePlatform, fmt.Sprintf("%s is not configured: social platform executor is not available", action), nil)
}

func socialTaskActionInputError(message string) error {
	return newSocialExecutionError(SocialExecutionFailureActionInput, message, nil)
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
	if account.AccountStatus != SocialAccountStatusAvailable {
		return fmt.Errorf("social account is unavailable")
	}
	return nil
}

func (e *SocialTaskExecutor) doLoginCheck(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	return e.executePlatformAction(ctx, taskLog)
}

// validateLoginAccountScope mirrors validateTaskExecutionAccountScope but omits
// the "available" requirement: login is the action that acquires credentials and
// makes a freshly imported account available, so it must run on not-yet-available
// accounts. Ownership is still enforced.
func validateLoginAccountScope(taskLog *dbent.SocialTaskLog, account *dbent.SocialAccount) error {
	if taskLog == nil || account == nil {
		return fmt.Errorf("social account is unavailable")
	}
	if account.AssignedUserID == nil || int64(*account.AssignedUserID) != taskLog.UserID {
		return fmt.Errorf("social account is unavailable")
	}
	return nil
}

// doLogin performs a password login through the platform adapter and writes the
// acquired credentials back to the account before billing. Any failure — login
// error or credential write-back error — is returned as a fail-closed error so
// the task is marked failed and never charged.
func (e *SocialTaskExecutor) doLogin(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if e == nil || e.entClient == nil || taskLog == nil {
		return unsupportedSocialAction(SocialTaskActionLogin)
	}
	account, err := e.entClient.SocialAccount.Get(ctx, taskLog.SocialAccountID)
	if err != nil {
		slog.Warn("social login account load failed", "task_log_id", taskLog.ID, "account_id", taskLog.SocialAccountID, "error", err)
		return "", fmt.Errorf("social account is unavailable")
	}
	if err := validateLoginAccountScope(taskLog, account); err != nil {
		return "", err
	}
	platform := normalizeSocialPlatform(account.PlatformKey)
	if platform == "" {
		platform = normalizeSocialPlatform(account.Platform)
	}
	executor, ok := e.executors[platform].(SocialAccountLoginExecutor)
	if !ok || executor == nil {
		return unsupportedSocialAction(SocialTaskActionLogin)
	}
	result, err := executor.Login(ctx, taskLog, account)
	if err != nil {
		return "", err
	}
	if result == nil || strings.TrimSpace(result.ExecutionAuth) == "" {
		return "", newSocialExecutionError(SocialExecutionFailureAuthInvalid, "login succeeded without usable auth credentials", nil)
	}
	if err := e.persistLoginCredentials(ctx, account.ID, account.Name, result); err != nil {
		slog.Error("social login credential write-back failed", "task_log_id", taskLog.ID, "account_id", account.ID, "error", err)
		return "", newSocialExecutionError(SocialExecutionFailurePlatform, "failed to store login credentials", err)
	}
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = "login succeeded"
	}
	return message, nil
}

// persistLoginCredentials writes acquired execution credentials back to the
// account. It runs before the billing finalizer so a write failure fails the
// task closed without charge.
func (e *SocialTaskExecutor) persistLoginCredentials(ctx context.Context, accountID int64, accountName string, result *SocialAccountCredentialResult) error {
	executionAuth, err := normalizeTwitterExecutionAuthForEncryptedStorage(result.ExecutionAuth, accountName, e.credentialEncryptor)
	if err != nil {
		return err
	}
	update := e.entClient.SocialAccount.UpdateOneID(accountID).
		SetExecutionAuth(executionAuth).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored)
	if authCookie := strings.TrimSpace(result.AuthCookie); authCookie != "" {
		update.SetAuthCookie(authCookie)
	}
	if platformUserID := trimPtr(result.PlatformUserID); platformUserID != "" {
		update.SetPlatformUserID(platformUserID)
	}
	_, err = update.Save(ctx)
	return err
}

func (e *SocialTaskExecutor) doFollow(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if socialTaskLogTarget(taskLog) == "" {
		return "", socialTaskActionInputError("follow target is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doPost(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	hasContent := taskLog != nil && taskLog.Content != nil && strings.TrimSpace(*taskLog.Content) != ""
	hasMedia := taskLog != nil && taskLog.Payload.Post != nil && len(taskLog.Payload.Post.Media) > 0
	if !hasContent && !hasMedia {
		return "", socialTaskActionInputError("post content or media is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doUpdateProfile(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog == nil || taskLog.Payload.Profile == nil || taskLog.Payload.Profile.IsZero() {
		return "", socialTaskActionInputError("profile payload is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doUpdateAvatar(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog == nil || taskLog.Payload.Avatar == nil || taskLog.Payload.Avatar.IsZero() {
		return "", socialTaskActionInputError("avatar media is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doUpdateBanner(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog == nil || taskLog.Payload.Banner == nil || taskLog.Payload.Banner.IsZero() {
		return "", socialTaskActionInputError("banner media is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doLike(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if socialTaskLogTarget(taskLog) == "" {
		return "", socialTaskActionInputError("like target (post URL/ID) is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func (e *SocialTaskExecutor) doRetweet(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
	if socialTaskLogTarget(taskLog) == "" {
		return "", socialTaskActionInputError("retweet target (post URL/ID) is required")
	}
	return e.executePlatformAction(ctx, taskLog)
}

func socialTaskLogTarget(taskLog *dbent.SocialTaskLog) string {
	if taskLog == nil {
		return ""
	}
	if target := strings.TrimSpace(taskLog.Payload.Target); target != "" {
		return target
	}
	if taskLog.Target == nil {
		return ""
	}
	return strings.TrimSpace(*taskLog.Target)
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
		Where(
			socialtasklog.StatusEQ(SocialTaskLogStatusPending),
			socialtasklog.ChargeStatusEQ(SocialTaskChargeStatusNotCharged),
		).
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
		message := safeSocialTaskFailureMessage(fmt.Errorf("social platform executor queue is not configured; task was not charged"))
		if err := e.markPendingTasksFailedNotCharged(ctx, ids, message); err != nil {
			return 0, err
		}
		return 0, nil
	}
	enqueued, failedIDs := e.EnqueueBatch(ids)
	if len(failedIDs) > 0 {
		message := safeSocialTaskFailureMessage(fmt.Errorf("social platform executor queue is full; task was not charged"))
		if err := e.markPendingTasksFailedNotCharged(ctx, failedIDs, message); err != nil {
			return enqueued, err
		}
	}
	return enqueued, nil
}

func (e *SocialTaskExecutor) markPendingTasksFailedNotCharged(ctx context.Context, taskLogIDs []int64, message string) error {
	accounts := NewSocialAccountServiceWithCredentialEncryptor(e.entClient, e.credentialEncryptor)
	for _, taskLogID := range taskLogIDs {
		if _, err := accounts.MarkTaskLogFailedNotCharged(ctx, taskLogID, message); err != nil {
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

func (e *SocialTaskExecutor) recordAccountBillingFailure(ctx context.Context, taskLog *dbent.SocialTaskLog, message string) {
	if e == nil || e.entClient == nil || taskLog == nil || taskLog.SocialAccountID <= 0 {
		return
	}
	update := e.entClient.SocialAccount.UpdateOneID(taskLog.SocialAccountID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored)
	if strings.TrimSpace(message) != "" {
		update.SetTaskMessage(message)
	} else {
		update.ClearTaskMessage()
	}
	if _, err := update.Save(ctx); err != nil {
		slog.Warn("failed to update social account billing failure state", "task_log_id", taskLog.ID, "account_id", taskLog.SocialAccountID, "error", err)
	}
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
	update := e.entClient.SocialAccount.UpdateOneID(taskLog.SocialAccountID).
		SetTaskMessage(safeSocialTaskFailureMessage(execErr))
	accountStatus, taskStatus, shouldUpdate := socialAccountStateForExecutionFailure(kind)
	if !shouldUpdate && taskLog.Action == SocialTaskActionLogin {
		taskStatus = SocialTaskStatusFailed
		shouldUpdate = true
	}
	if !shouldUpdate {
		if _, err := update.Save(ctx); err != nil {
			slog.Warn(
				"failed to update social account latest task message",
				"task_log_id", taskLog.ID,
				"account_id", taskLog.SocialAccountID,
				"failure_kind", kind,
				"error", err,
			)
		}
		return
	}
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
	rawMessage := strings.Join(strings.Fields(fmt.Sprint(err)), " ")
	normalizedMessage := strings.ToLower(rawMessage)
	if rawMessage == "" {
		return "任务执行失败，本次未扣费"
	}
	if kind, ok := socialExecutionFailureKind(err); ok && kind == SocialExecutionFailurePasswordInvalid {
		return "密码错误，本次未扣费"
	}
	if message, ok := knownSocialTaskFailureMessage(rawMessage); ok {
		return message
	}
	if IsTwitterPlatformFailureMessage(rawMessage) {
		return rawMessage
	}
	switch {
	case strings.Contains(normalizedMessage, "avatar image must be 400x400 pixels"):
		return "头像图片尺寸必须为 400x400，本次未扣费"
	case strings.Contains(normalizedMessage, "banner image must be 1500x500 pixels"):
		return "背景图图片尺寸必须为 1500x500，本次未扣费"
	case strings.Contains(normalizedMessage, "media asset is unavailable"):
		return "任务媒体资源不可用，本次未扣费"
	case strings.Contains(normalizedMessage, "media source is not supported for socialops execution"):
		return "媒体引用暂未开放，本次未扣费"
	case strings.Contains(normalizedMessage, "video media is not supported for socialops execution"):
		return "视频发帖媒体暂未开放，本次未扣费"
	case strings.Contains(normalizedMessage, "post media content type is not supported"):
		return "发帖媒体类型暂不支持，本次未扣费"
	case strings.Contains(normalizedMessage, "twitter media upload returned invalid media id"),
		strings.Contains(normalizedMessage, "twitter media upload returned invalid response"),
		strings.Contains(normalizedMessage, "twitter media upload returned no media id"),
		strings.Contains(normalizedMessage, "twitter media upload returned processing failed"),
		strings.Contains(normalizedMessage, "twitter media upload returned processing timeout"):
		return "平台媒体上传失败，本次未扣费"
	case strings.Contains(normalizedMessage, "target not found"):
		return "执行目标不存在，本次未扣费"
	case strings.Contains(normalizedMessage, "content is too long"),
		strings.Contains(normalizedMessage, "duplicate"),
		strings.Contains(normalizedMessage, "already"),
		strings.Contains(normalizedMessage, "restricted"):
		return "内容或目标状态不符合平台要求，本次未扣费"
	}

	kind, ok := socialExecutionFailureKind(err)
	if ok {
		switch kind {
		case SocialExecutionFailurePasswordInvalid:
			return "密码错误，本次未扣费"
		case SocialExecutionFailureAuthMissing, SocialExecutionFailureAuthInvalid:
			return "账号认证信息不可用，本次未扣费"
		case SocialExecutionFailureProxyMissing, SocialExecutionFailureProxyInvalid, SocialExecutionFailureProxyUnavailable:
			return "执行代理不可用，本次未扣费"
		case SocialExecutionFailureNetwork:
			return "平台网络请求失败，本次未扣费"
		case SocialExecutionFailureConfiguration:
			return "登录依赖服务未配置，本次未扣费"
		case SocialExecutionFailureUnsupported:
			return "该动作暂不支持，本次未扣费"
		case SocialExecutionFailureActionInput:
			return "任务参数不完整，本次未扣费"
		case SocialExecutionFailureChallengeRequired:
			return "账号需要额外验证，本次未扣费"
		case SocialExecutionFailureAccountLimited:
			return "账号状态或频率受限，本次未扣费"
		case SocialExecutionFailurePlatform:
			return rawMessage
		}
	}
	message := normalizedMessage
	switch {
	case strings.Contains(message, "wrong password") ||
		strings.Contains(message, "incorrect password") ||
		strings.Contains(message, "invalid password") ||
		strings.Contains(message, "password is incorrect") ||
		strings.Contains(message, "password you entered is incorrect") ||
		strings.Contains(message, "密码错误"):
		return "密码错误，本次未扣费"
	case strings.Contains(message, "queue is full"):
		return "任务队列繁忙，本次未扣费"
	case strings.Contains(message, "billing failed"):
		return "执行已完成，但扣费确认异常，请联系管理员处理"
	case strings.Contains(message, "device fingerprint provider is not configured") ||
		strings.Contains(message, "twitter login is not configured"):
		return "登录依赖服务未配置，本次未扣费"
	case strings.Contains(message, "auth cookie") || strings.Contains(message, "authentication failed") || strings.Contains(message, "oauth") || strings.Contains(message, "token"):
		return "账号认证信息不可用，本次未扣费"
	case strings.Contains(message, "global proxy"):
		return "全局代理不可用，本次未扣费"
	case strings.Contains(message, "proxy"):
		return "执行代理不可用，本次未扣费"
	case strings.Contains(message, "network") || strings.Contains(message, "timeout") || strings.Contains(message, "connection"):
		return "平台网络请求失败，本次未扣费"
	case strings.Contains(message, "unsupported action"):
		return "该动作暂不支持，本次未扣费"
	case strings.Contains(message, "target is required") || strings.Contains(message, "content is required"):
		return "任务参数不完整，本次未扣费"
	case strings.Contains(message, "target not found"):
		return "执行目标不存在，本次未扣费"
	case strings.Contains(message, "challenge required") ||
		strings.Contains(message, "captcha challenge") ||
		strings.Contains(message, "verification required") ||
		strings.Contains(message, "additional verification") ||
		strings.Contains(message, "confirm your identity"):
		return "账号需要额外验证，本次未扣费"
	case strings.Contains(message, "rate limit") || strings.Contains(message, "too frequent") || strings.Contains(message, "follow limit") || strings.Contains(message, "suspended") || strings.Contains(message, "locked"):
		return "账号状态或频率受限，本次未扣费"
	case strings.Contains(message, "content is too long") || strings.Contains(message, "duplicate") || strings.Contains(message, "already") || strings.Contains(message, "restricted"):
		return "内容或目标状态不符合平台要求，本次未扣费"
	case strings.Contains(message, "access denied"):
		return "平台拒绝执行，本次未扣费"
	default:
		return rawMessage
	}
}

func knownSocialTaskFailureMessage(message string) (string, bool) {
	if translated, ok := KnownTwitterTaskFailureMessage(message); ok {
		return translated, true
	}
	if IsTwitterPlatformFailureMessage(message) {
		return "", false
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(message), " "))
	switch {
	case strings.Contains(normalized, "wrong password") ||
		strings.Contains(normalized, "incorrect password") ||
		strings.Contains(normalized, "invalid password") ||
		strings.Contains(normalized, "password is incorrect") ||
		strings.Contains(normalized, "password you entered is incorrect") ||
		strings.Contains(normalized, "密码错误"):
		return "密码错误，本次未扣费", true
	case strings.Contains(normalized, "could not find your account") ||
		strings.Contains(normalized, "account not found") ||
		strings.Contains(normalized, "user not found"):
		return "账号不存在，本次未扣费", true
	case strings.Contains(normalized, "challenge required") ||
		strings.Contains(normalized, "captcha challenge") ||
		strings.Contains(normalized, "verification required") ||
		strings.Contains(normalized, "verify login") ||
		strings.Contains(normalized, "additional verification") ||
		strings.Contains(normalized, "confirm your identity"):
		return "账号需要额外验证，本次未扣费", true
	case strings.Contains(normalized, "suspended") ||
		strings.Contains(normalized, "locked") ||
		strings.Contains(normalized, "automated") ||
		strings.Contains(normalized, "rate limit") ||
		strings.Contains(normalized, "too frequent") ||
		strings.Contains(normalized, "follow limit"):
		return "账号状态或频率受限，本次未扣费", true
	case strings.Contains(normalized, "content is too long") ||
		strings.Contains(normalized, "duplicate") ||
		strings.Contains(normalized, "already") ||
		strings.Contains(normalized, "restricted"):
		return "内容或目标状态不符合平台要求，本次未扣费", true
	case strings.Contains(normalized, "authentication failed"):
		return "账号认证信息不可用，本次未扣费", true
	}
	return "", false
}

func socialAccountStateForExecutionFailure(kind SocialExecutionFailureKind) (accountStatus, taskStatus string, shouldUpdate bool) {
	switch kind {
	case SocialExecutionFailureAuthMissing:
		return SocialAccountStatusNotStored, SocialTaskStatusManualReview, true
	case SocialExecutionFailureAuthInvalid, SocialExecutionFailurePasswordInvalid:
		return SocialAccountStatusInvalid, SocialTaskStatusManualReview, true
	case SocialExecutionFailureAccountLimited:
		return SocialAccountStatusLimited, SocialTaskStatusManualReview, true
	case SocialExecutionFailureChallengeRequired:
		return SocialAccountStatusPendingCheck, SocialTaskStatusManualReview, true
	case SocialExecutionFailureProxyMissing, SocialExecutionFailureProxyInvalid, SocialExecutionFailureProxyUnavailable:
		return "", SocialTaskStatusIPUnavailable, true
	default:
		return "", "", false
	}
}
