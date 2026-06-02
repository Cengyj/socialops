//go:build unit

package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestSocialAccountHandlerSubmitTaskFailsClosedWithoutCharging(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-task@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_task").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_task").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newSocialAccountHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target",
		"client_request_id": "g008-submit-1"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"failed_closed":1`)
	require.Zero(t, userRepo.deductCalls)
	require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)

	logs, err := client.SocialTaskLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, service.SocialTaskLogStatusFailed, logs[0].Status)
	require.Equal(t, service.SocialTaskChargeStatusNotCharged, logs[0].ChargeStatus)
	require.Zero(t, logs[0].ChargedAmount)
	require.Nil(t, logs[0].ChargeSource)
	require.NotNil(t, logs[0].ResultMessage)
	require.Contains(t, *logs[0].ResultMessage, "not configured")
	require.NotNil(t, logs[0].IdempotencyKey)
	require.Equal(t, "g008-submit-1", *logs[0].IdempotencyKey)
}

func TestSocialAccountHandlerSubmitTaskUsesExecutorQueue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-queue@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_queue").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_queue").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newSocialAccountHandlerForTestWithExecutor(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target",
		"client_request_id": "g008-submit-queue"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"enqueued":1`)
	require.Contains(t, rec.Body.String(), `"failed_closed":0`)
	require.Zero(t, userRepo.deductCalls)
	require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)

	logs, err := client.SocialTaskLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, service.SocialTaskLogStatusPending, logs[0].Status)
	require.Equal(t, service.SocialTaskChargeStatusNotCharged, logs[0].ChargeStatus)
	require.Zero(t, logs[0].ChargedAmount)
}

func TestSocialAccountHandlerEstimateTaskDeduplicatesAccountIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "estimate-dedupe@example.com")
	account := client.SocialAccount.Create().
		SetName("@estimate_dedupe").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("estimate_dedupe").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := newSocialAccountHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks/estimate", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`, `+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.EstimateTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"action_count":1`)
	require.NotContains(t, rec.Body.String(), `"action_count":2`)
}

func TestSocialAccountHandlerSubmitTaskDeduplicatesAccountIDsWithoutIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-dedupe@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_dedupe").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_dedupe").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newSocialAccountHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`, `+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"submitted":1`)
	require.Contains(t, rec.Body.String(), `"failed_closed":1`)
	require.Contains(t, rec.Body.String(), `"action_count":1`)
	require.Zero(t, userRepo.deductCalls)

	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestSocialAccountHandlerEstimateTaskRejectsNonPositiveAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "estimate-invalid-id@example.com")
	account := client.SocialAccount.Create().
		SetName("@estimate_invalid_id").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("estimate_invalid_id").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := newSocialAccountHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks/estimate", bytes.NewBufferString(`{
		"account_ids": [0, `+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.EstimateTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_ACCOUNT_ID_INVALID")
}

func TestSocialAccountHandlerSubmitTaskRejectsNonPositiveAccountIDWithoutLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-invalid-id@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_invalid_id").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_invalid_id").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newSocialAccountHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks", bytes.NewBufferString(`{
		"account_ids": [-1, `+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_ACCOUNT_ID_INVALID")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSocialAccountHandlerRejectsUnavailableAccountWithoutLogOrCharge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "limited-task@example.com")
	account := client.SocialAccount.Create().
		SetName("@limited_task").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("limited_task").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusLimited).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newSocialAccountHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "account is not available")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSocialAccountHandlerRejectsInsufficientFundsWithoutLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "no-funds@example.com")
	account := client.SocialAccount.Create().
		SetName("@no_funds").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("no_funds").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}}
	handler := newSocialAccountHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_INSUFFICIENT_FUNDS")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSocialAccountHandlerRejectsStaleDefaultProxyWithoutLogOrCharge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "offline-default-proxy@example.com")
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("offline default proxy").
		SetIPType("residential").
		SetStatus(service.SocialIPStatusOffline).
		SaveX(ctx)
	snapshotIP := &service.SocialIP{ID: ip.ID, UserID: user.ID, Name: ip.Name, IPType: ip.IPType, Status: ip.Status}
	snapshot := service.SocialIPTaskSnapshot(snapshotIP)
	account := client.SocialAccount.Create().
		SetName("@offline_proxy_task").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("offline_proxy_task").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SetBoundIP(snapshot).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newSocialAccountHandlerForTestWithExecutor(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_NOT_AVAILABLE")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSocialAccountHandlerListsOnlyCurrentUserResources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSocialAccountHandlerTestClient(t)
	owner := createSocialHandlerUser(t, ctx, client, "list-owner@example.com")
	other := createSocialHandlerUser(t, ctx, client, "list-other@example.com")
	ownAccount := client.SocialAccount.Create().
		SetName("@own_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("own_account").
		SetAssignedUserID(owner.ID).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@other_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("other_account").
		SetAssignedUserID(other.ID).
		SaveX(ctx)
	client.SocialIP.Create().
		SetUserID(other.ID).
		SetName("other proxy").
		SetIPType("residential").
		SaveX(ctx)
	handler := newSocialAccountHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: owner.ID, Balance: 0}})

	rec := invokeSocialHandlerAsUser(t, owner.ID, http.MethodGet, "/api/v1/social-accounts", nil, handler.ListMyAccounts)
	require.Equal(t, http.StatusOK, rec.Code)
	requireSinglePaginatedID(t, rec.Body.Bytes(), ownAccount.ID)
}

func newSocialAccountHandlerForTest(client *dbent.Client, userRepo *socialHandlerBillingUserRepo) *SocialAccountHandler {
	subRepo := &socialHandlerSubscriptionRepo{}
	billing := service.NewSocialBillingService(userRepo, subRepo, nil, nil)
	return NewSocialAccountHandler(
		service.NewSocialAccountService(client),
		service.NewSocialIPService(client),
		billing,
		nil,
	)
}

func newSocialAccountHandlerForTestWithExecutor(client *dbent.Client, userRepo *socialHandlerBillingUserRepo) *SocialAccountHandler {
	subRepo := &socialHandlerSubscriptionRepo{}
	billing := service.NewSocialBillingService(userRepo, subRepo, nil, nil)
	executor := service.NewSocialTaskExecutor(client, billing, service.SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 10, MinIntervalMs: 1})
	return NewSocialAccountHandler(
		service.NewSocialAccountService(client),
		service.NewSocialIPService(client),
		billing,
		executor,
	)
}

func newSocialAccountHandlerTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func createSocialHandlerUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) *dbent.User {
	t.Helper()
	return client.User.Create().
		SetEmail(email).
		SetPasswordHash("hashed-password").
		SaveX(ctx)
}

func invokeSocialHandlerAsUser(t *testing.T, userID int64, method, path string, body []byte, fn gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
	fn(ginCtx)
	return rec
}

func requireSinglePaginatedID(t *testing.T, raw []byte, want int64) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID int64 `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, want, resp.Data.Items[0].ID)
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

type socialHandlerBillingUserRepo struct {
	service.UserRepository
	user        *service.User
	deductCalls int
}

func (r *socialHandlerBillingUserRepo) GetByID(_ context.Context, id int64) (*service.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, service.ErrUserNotFound
	}
	out := *r.user
	return &out, nil
}

func (r *socialHandlerBillingUserRepo) DeductBalance(_ context.Context, id int64, amount float64) error {
	if r.user == nil || r.user.ID != id {
		return service.ErrUserNotFound
	}
	r.deductCalls++
	r.user.Balance -= amount
	return nil
}

type socialHandlerSubscriptionRepo struct {
	service.UserSubscriptionRepository
	subs []service.UserSubscription
}

func (r *socialHandlerSubscriptionRepo) ListActiveByUserID(_ context.Context, userID int64) ([]service.UserSubscription, error) {
	out := make([]service.UserSubscription, 0, len(r.subs))
	for _, sub := range r.subs {
		if sub.UserID == userID && sub.Status == service.SubscriptionStatusActive {
			out = append(out, sub)
		}
	}
	return out, nil
}

func (r *socialHandlerSubscriptionRepo) IncrementUsage(_ context.Context, _ int64, _ float64) error {
	return nil
}
