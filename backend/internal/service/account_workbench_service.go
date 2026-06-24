package service

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

type AccountWorkbenchTaskMode string

const (
	AccountWorkbenchTaskModeUser  AccountWorkbenchTaskMode = "user"
	AccountWorkbenchTaskModeAdmin AccountWorkbenchTaskMode = "admin"
)

type AccountWorkbenchTaskInput struct {
	Mode             AccountWorkbenchTaskMode
	UserID           int64
	AccountIDs       []int64
	Action           string
	Target           *string
	Content          *string
	TargetPool       []string
	ContentPool      []string
	Payload          *SocialTaskPayload
	TemplateSnapshot *SocialTaskTemplateSnapshot
	IdempotencyKey   string
	BillingRequestID *string
}

type AccountWorkbenchSubmitResult struct {
	Submitted    int
	Enqueued     int
	FailedClosed int
	Logs         []*SocialTaskLog
}

type accountWorkbenchBillingBucket struct {
	count    int
	platform string
}

type accountWorkbenchPendingTask struct {
	accountID     int64
	userID        int64
	proxyID       *int64
	proxySnapshot *string
	target        *string
	content       *string
	payload       *SocialTaskPayload
}

type AccountWorkbenchService struct {
	accounts      *SocialAccountService
	ips           *SocialIPService
	globalProxies *GlobalProxyService
	billing       *SocialBillingService
	executor      *SocialTaskExecutor
}

func NewAccountWorkbenchService(accounts *SocialAccountService, ips *SocialIPService, billing *SocialBillingService, executor *SocialTaskExecutor) *AccountWorkbenchService {
	return &AccountWorkbenchService{accounts: accounts, ips: ips, billing: billing, executor: executor}
}

func NewAccountWorkbenchServiceWithGlobalProxies(accounts *SocialAccountService, ips *SocialIPService, globalProxies *GlobalProxyService, billing *SocialBillingService, executor *SocialTaskExecutor) *AccountWorkbenchService {
	return &AccountWorkbenchService{accounts: accounts, ips: ips, globalProxies: globalProxies, billing: billing, executor: executor}
}

func (s *AccountWorkbenchService) SubmitTask(ctx context.Context, input *AccountWorkbenchTaskInput) (*AccountWorkbenchSubmitResult, error) {
	if s == nil || s.accounts == nil || s.billing == nil {
		return nil, infraerrors.ServiceUnavailable("SOCIAL_TASK_SERVICE_UNAVAILABLE", "social task service is unavailable")
	}
	if err := normalizeAccountWorkbenchTaskInput(input); err != nil {
		return nil, err
	}
	accounts, err := s.validTaskAccounts(ctx, input)
	if err != nil {
		return nil, err
	}
	accountByID := make(map[int64]*SocialAccount, len(accounts))
	for _, account := range accounts {
		accountByID[account.ID] = account
	}

	logs := make([]*SocialTaskLog, 0, len(input.AccountIDs))
	pending := make([]accountWorkbenchPendingTask, 0, len(input.AccountIDs))
	buckets := make(map[int64]accountWorkbenchBillingBucket)

	for index, accountID := range input.AccountIDs {
		account := accountByID[accountID]
		ownerID := *account.AssignedUserID
		target := selectTaskTemplateValue(input.Target, input.TargetPool, index)
		content := selectTaskTemplateValue(input.Content, input.ContentPool, index)
		payload := buildAccountWorkbenchTaskPayload(input, index)
		if input.IdempotencyKey != "" {
			existing, err := s.accounts.FindTaskLogByIdempotency(ctx, ownerID, accountID, input.Action, input.IdempotencyKey)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				if !socialTaskLogMatchesAccountWorkbenchTask(existing, input, target, content, payload) {
					return nil, ErrSocialTaskIdempotencyConflict
				}
				logs = append(logs, existing)
				continue
			}
		}
		active, err := s.accounts.HasActiveTaskLogForAccount(ctx, ownerID, accountID)
		if err != nil {
			return nil, err
		}
		if active {
			return nil, ErrSocialTaskAccountBusy
		}
		if err := addAccountWorkbenchBillingBucket(input.Mode, buckets, ownerID, account.Platform); err != nil {
			return nil, err
		}

		var taskProxyID *int64
		var taskProxySnapshot *string
		defaultProxySnapshot := trimPtr(account.DefaultProxySnapshot)
		if defaultProxySnapshot == "" && input.Mode == AccountWorkbenchTaskModeUser {
			if input.Action != SocialTaskActionLogin {
				return nil, infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "default social IP is required for execution")
			}
			var err error
			taskProxySnapshot, err = s.resolveGlobalFallbackProxy(ctx)
			if err != nil {
				return nil, err
			}
		}
		if defaultProxySnapshot != "" {
			var err error
			taskProxyID, taskProxySnapshot, err = s.resolveAccountDefaultProxy(ctx, ownerID, account)
			if err != nil {
				return nil, err
			}
		}
		pending = append(pending, accountWorkbenchPendingTask{
			accountID:     accountID,
			userID:        ownerID,
			proxyID:       taskProxyID,
			proxySnapshot: taskProxySnapshot,
			target:        target,
			content:       content,
			payload:       payload,
		})
	}

	if IsBillableSocialTaskAction(input.Action) {
		for userID, bucket := range buckets {
			if _, err := s.billing.EnsureCanAffordForPlatform(ctx, userID, bucket.platform, bucket.count); err != nil {
				return nil, err
			}
		}
	}

	taskLogIDs := make([]int64, 0, len(pending))
	for _, task := range pending {
		log, err := s.accounts.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
			AccountID:        task.accountID,
			UserID:           task.userID,
			Action:           input.Action,
			Target:           task.target,
			Content:          task.content,
			Payload:          task.payload,
			TemplateSnapshot: cloneSocialTaskTemplateSnapshot(input.TemplateSnapshot),
			Status:           SocialTaskLogStatusPending,
			ProxyID:          task.proxyID,
			ProxySnapshot:    task.proxySnapshot,
			BillingRequestID: input.BillingRequestID,
			IdempotencyKey:   optionalAccountWorkbenchString(input.IdempotencyKey),
		})
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
		taskLogIDs = append(taskLogIDs, log.ID)
	}

	enqueued, failedClosed, err := s.enqueueOrFailClosed(ctx, taskLogIDs, &logs)
	if err != nil {
		return nil, err
	}
	return &AccountWorkbenchSubmitResult{
		Submitted:    len(logs),
		Enqueued:     enqueued,
		FailedClosed: failedClosed,
		Logs:         logs,
	}, nil
}

func normalizeAccountWorkbenchTaskInput(input *AccountWorkbenchTaskInput) error {
	if input == nil {
		return infraerrors.BadRequest("SOCIAL_TASK_INPUT_REQUIRED", "social task input is required")
	}
	action, ok := NormalizeSocialTaskAction(input.Action)
	if !ok {
		return ErrSocialTaskUnsupportedAction
	}
	input.Action = action
	if err := EnsureExecutableSocialTaskAction(input.Action); err != nil {
		return err
	}
	if len(input.AccountIDs) == 0 {
		return infraerrors.BadRequest("SOCIAL_TASK_ACCOUNTS_REQUIRED", "at least one social account is required")
	}
	for _, accountID := range input.AccountIDs {
		if accountID <= 0 {
			return infraerrors.BadRequest("SOCIAL_TASK_ACCOUNT_ID_INVALID", "social account id must be positive")
		}
	}
	input.AccountIDs = UniqueInt64sPreserveOrder(input.AccountIDs)
	input.TargetPool = normalizeAccountWorkbenchTaskValues(input.TargetPool)
	input.ContentPool = normalizeAccountWorkbenchTaskValues(input.ContentPool)
	switch input.Action {
	case SocialTaskActionFollow, SocialTaskActionLike, SocialTaskActionRetweet:
		if isEmptyTaskValue(input.Target) && len(input.TargetPool) == 0 && !taskPayloadHasTarget(input.Payload) {
			return infraerrors.BadRequest("SOCIAL_TASK_TARGET_REQUIRED", "target is required for this action")
		}
	case SocialTaskActionPost:
		hasMedia := taskPayloadHasPostMedia(input.Payload)
		if isEmptyTaskValue(input.Content) && len(input.ContentPool) == 0 && !hasMedia {
			return infraerrors.BadRequest("SOCIAL_TASK_POST_CONFIGURATION_REQUIRED", "post content or media is required")
		}
		if input.Payload != nil && input.Payload.Post != nil {
			if err := validateSocialTaskSupportedPostMedia(input.Payload.Post.Media); err != nil {
				return infraerrors.BadRequest("SOCIAL_TASK_MEDIA_UNSUPPORTED", err.Error())
			}
		}
	case SocialTaskActionUpdateProfile:
		if input.Payload == nil || input.Payload.Profile == nil || input.Payload.Profile.IsZero() {
			return infraerrors.BadRequest("SOCIAL_TASK_PAYLOAD_REQUIRED", "profile payload is required")
		}
	case SocialTaskActionUpdateAvatar:
		if input.Payload == nil || input.Payload.Avatar == nil || input.Payload.Avatar.IsZero() {
			return infraerrors.BadRequest("SOCIAL_TASK_PAYLOAD_REQUIRED", "avatar media is required")
		}
		if err := validateSocialTaskExecutableInlineMediaSource("avatar", input.Payload.Avatar); err != nil {
			return infraerrors.BadRequest("SOCIAL_TASK_MEDIA_UNSUPPORTED", err.Error())
		}
		if err := validateSocialTaskImageMedia("avatar", input.Payload.Avatar); err != nil {
			return infraerrors.BadRequest("SOCIAL_TASK_MEDIA_UNSUPPORTED", err.Error())
		}
	case SocialTaskActionUpdateBanner:
		if input.Payload == nil || input.Payload.Banner == nil || input.Payload.Banner.IsZero() {
			return infraerrors.BadRequest("SOCIAL_TASK_PAYLOAD_REQUIRED", "banner media is required")
		}
		if err := validateSocialTaskExecutableInlineMediaSource("banner", input.Payload.Banner); err != nil {
			return infraerrors.BadRequest("SOCIAL_TASK_MEDIA_UNSUPPORTED", err.Error())
		}
		if err := validateSocialTaskImageMedia("banner", input.Payload.Banner); err != nil {
			return infraerrors.BadRequest("SOCIAL_TASK_MEDIA_UNSUPPORTED", err.Error())
		}
	}
	return nil
}

func normalizeAccountWorkbenchTaskValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func isEmptyTaskValue(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == ""
}

func selectTaskTemplateValue(primary *string, pool []string, index int) *string {
	if primary != nil {
		trimmed := strings.TrimSpace(*primary)
		if trimmed != "" {
			return &trimmed
		}
	}
	if len(pool) == 0 {
		return nil
	}
	value := pool[index%len(pool)]
	return &value
}

func shuffledTaskValues(values []string) []string {
	normalized := normalizeAccountWorkbenchTaskValues(values)
	if len(normalized) < 2 {
		return normalized
	}
	shuffled := append([]string(nil), normalized...)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func (s *AccountWorkbenchService) validTaskAccounts(ctx context.Context, input *AccountWorkbenchTaskInput) ([]*SocialAccount, error) {
	accounts := make([]*SocialAccount, 0, len(input.AccountIDs))
	for _, accountID := range input.AccountIDs {
		account, err := s.accounts.GetByID(ctx, accountID)
		if err != nil {
			return nil, err
		}
		if account.AssignedUserID == nil {
			return nil, ErrSocialAccountNotAssigned
		}
		if input.Mode == AccountWorkbenchTaskModeUser && *account.AssignedUserID != input.UserID {
			return nil, ErrSocialAccountNotAssigned
		}
		// The login action acquires/refreshes credentials, so it is the path that
		// makes an account available. Requiring "available" here would deadlock a
		// freshly imported account that has no cookie yet. Instead require a
		// password to log in with; all other actions still require an available
		// account with valid credentials.
		if input.Action == SocialTaskActionLogin {
			if strings.TrimSpace(stringValue(account.Password)) == "" {
				return nil, infraerrors.BadRequest("SOCIAL_TASK_LOGIN_PASSWORD_REQUIRED", "account password is required to log in")
			}
		} else if account.AccountStatus != SocialAccountStatusAvailable {
			return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_NOT_AVAILABLE", "account is not available for execution")
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func taskBillingBuckets(mode AccountWorkbenchTaskMode, accounts []*SocialAccount) (map[int64]accountWorkbenchBillingBucket, error) {
	buckets := make(map[int64]accountWorkbenchBillingBucket)
	for _, account := range accounts {
		if account == nil || account.AssignedUserID == nil {
			continue
		}
		if err := addAccountWorkbenchBillingBucket(mode, buckets, *account.AssignedUserID, account.Platform); err != nil {
			return nil, err
		}
	}
	if len(buckets) == 0 {
		return nil, infraerrors.BadRequest("SOCIAL_TASK_PLATFORM_REQUIRED", "social account platform is required for billing")
	}
	return buckets, nil
}

func addAccountWorkbenchBillingBucket(mode AccountWorkbenchTaskMode, buckets map[int64]accountWorkbenchBillingBucket, userID int64, platform string) error {
	current := NormalizePlanPlatform(platform)
	if current == "" {
		return infraerrors.BadRequest("SOCIAL_TASK_PLATFORM_REQUIRED", "social account platform is required for billing")
	}
	bucket := buckets[userID]
	if mode == AccountWorkbenchTaskModeUser && len(buckets) > 0 && bucket.platform == "" {
		return ErrSocialTaskMixedPlatforms
	}
	if bucket.platform != "" && bucket.platform != current {
		return ErrSocialTaskMixedPlatforms
	}
	bucket.platform = current
	bucket.count++
	buckets[userID] = bucket
	return nil
}

func (s *AccountWorkbenchService) resolveAccountDefaultProxy(ctx context.Context, ownerID int64, account *SocialAccount) (*int64, *string, error) {
	snapshot := ""
	if account != nil {
		snapshot = trimPtr(account.DefaultProxySnapshot)
	}
	if snapshot == "" {
		return nil, nil, nil
	}
	defaultProxyID, ok := SocialIPIDFromSnapshot(snapshot)
	if !ok {
		return nil, nil, staleAccountDefaultProxyError()
	}
	proxyID, proxySnapshot, err := s.resolveTaskProxy(ctx, ownerID, defaultProxyID)
	if err != nil && infraerrors.Reason(err) == "SOCIAL_IP_NOT_FOUND" {
		return nil, nil, staleAccountDefaultProxyError()
	}
	return proxyID, proxySnapshot, err
}

func staleAccountDefaultProxyError() error {
	return infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "default social IP snapshot is stale")
}

func (s *AccountWorkbenchService) resolveGlobalFallbackProxy(ctx context.Context) (*string, error) {
	if s == nil || s.globalProxies == nil {
		return nil, infraerrors.ServiceUnavailable("GLOBAL_PROXY_SERVICE_UNAVAILABLE", "global proxy service is unavailable")
	}
	ip, err := s.globalProxies.NextAvailable(ctx)
	if err != nil {
		if infraerrors.Reason(err) == "GLOBAL_PROXY_NOT_AVAILABLE" {
			return nil, infraerrors.BadRequest("GLOBAL_PROXY_NOT_AVAILABLE", "global proxy is not available")
		}
		return nil, err
	}
	snapshot := GlobalProxyTaskSnapshot(ip)
	return &snapshot, nil
}

func (s *AccountWorkbenchService) resolveTaskProxy(ctx context.Context, ownerID, proxyID int64) (*int64, *string, error) {
	if s == nil || s.ips == nil {
		return nil, nil, infraerrors.ServiceUnavailable("SOCIAL_IP_SERVICE_UNAVAILABLE", "social IP service is unavailable")
	}
	ip, err := s.ips.GetByIDForUser(ctx, proxyID, ownerID)
	if err != nil {
		return nil, nil, err
	}
	if err := EnsureSocialIPUsableForExecution(ip); err != nil {
		return nil, nil, err
	}
	snapshot := SocialIPTaskSnapshot(ip)
	return &proxyID, &snapshot, nil
}

func (s *AccountWorkbenchService) enqueueOrFailClosed(ctx context.Context, taskLogIDs []int64, logs *[]*SocialTaskLog) (int, int, error) {
	if len(taskLogIDs) == 0 {
		return 0, 0, nil
	}
	if s.executor == nil || s.executor.isStopped() {
		message := safeSocialTaskFailureMessage(fmt.Errorf("social platform executor queue is not configured; task was not charged"))
		for _, taskLogID := range taskLogIDs {
			log, err := s.accounts.MarkTaskLogFailedNotCharged(ctx, taskLogID, message)
			if err != nil {
				return 0, 0, err
			}
			*logs = replaceAccountWorkbenchTaskLog(*logs, log)
		}
		return 0, len(taskLogIDs), nil
	}
	enqueued, failedIDs := s.executor.EnqueueBatch(taskLogIDs)
	if len(failedIDs) > 0 {
		message := safeSocialTaskFailureMessage(fmt.Errorf("social platform executor queue is full; task was not charged"))
		failedClosed := 0
		for _, taskLogID := range failedIDs {
			log, err := s.accounts.MarkTaskLogFailedNotCharged(ctx, taskLogID, message)
			if err != nil {
				return enqueued, failedClosed, err
			}
			*logs = replaceAccountWorkbenchTaskLog(*logs, log)
			failedClosed++
		}
		return enqueued, failedClosed, nil
	}
	return enqueued, 0, nil
}

func replaceAccountWorkbenchTaskLog(logs []*SocialTaskLog, replacement *SocialTaskLog) []*SocialTaskLog {
	if replacement == nil {
		return logs
	}
	for i, item := range logs {
		if item != nil && item.ID == replacement.ID {
			logs[i] = replacement
			return logs
		}
	}
	return logs
}

func optionalAccountWorkbenchString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func socialTaskLogMatchesAccountWorkbenchTask(log *SocialTaskLog, input *AccountWorkbenchTaskInput, target, content *string, payload *SocialTaskPayload) bool {
	if log == nil || input == nil {
		return false
	}
	return socialTaskLogMatchesIdempotentInput(log, &CreateSocialTaskLogInput{
		AccountID:        log.SocialAccountID,
		UserID:           log.UserID,
		Action:           input.Action,
		Target:           target,
		Content:          content,
		Payload:          payload,
		TemplateSnapshot: input.TemplateSnapshot,
	})
}

func buildAccountWorkbenchTaskPayload(input *AccountWorkbenchTaskInput, index int) *SocialTaskPayload {
	if input == nil || input.Payload == nil {
		return nil
	}
	base := cloneSocialTaskPayload(input.Payload)
	if base == nil {
		return nil
	}
	if target := selectTaskTemplateValue(input.Target, input.TargetPool, index); target != nil {
		base.Target = strings.TrimSpace(*target)
	}
	if content := selectTaskTemplateValue(input.Content, input.ContentPool, index); content != nil {
		if base.Post == nil {
			base.Post = &SocialPostPayload{}
		}
		base.Post.Text = strings.TrimSpace(*content)
	}
	if base.IsZero() {
		return nil
	}
	return base
}

func taskPayloadHasPostMedia(payload *SocialTaskPayload) bool {
	if payload == nil || payload.Post == nil {
		return false
	}
	return len(payload.Post.Media) > 0
}

func taskPayloadHasTarget(payload *SocialTaskPayload) bool {
	return payload != nil && strings.TrimSpace(payload.Target) != ""
}
