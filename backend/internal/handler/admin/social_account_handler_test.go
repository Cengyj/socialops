//go:build unit

package admin

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSocialAccountAdminImportRejectsOversizedFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSocialAccountAdminHandler(nil, nil, nil, nil)

	body, contentType := multipartBody(t, "huge.csv", "text/csv", strings.Repeat("x", maxSocialAccountImportFileBytes+1))
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/social-accounts/import", body)
	ctx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ctx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "too large")
}

func TestSocialAccountAdminImportRejectsTooManyCSVRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSocialAccountAdminHandler(nil, nil, nil, nil)

	var csv strings.Builder
	csv.WriteString("name,platform\n")
	for i := 0; i <= maxSocialAccountImportRecords; i++ {
		csv.WriteString("@user")
		csv.WriteString(strconv.Itoa(i))
		csv.WriteString(",x_twitter\n")
	}
	body, contentType := multipartBody(t, "accounts.csv", "text/csv", csv.String())
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/social-accounts/import", body)
	ctx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ctx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "record limit exceeded")
}

func TestSocialAccountAdminEstimateTaskDeduplicatesAccountIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "admin-estimate-dedupe@example.com")
	account := client.SocialAccount.Create().
		SetName("@admin_estimate_dedupe").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_estimate_dedupe").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := newSocialAccountAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 1.0},
	}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/social-accounts/tasks/estimate", bytes.NewBufferString(`{
		"account_ids": [`+strconv.FormatInt(account.ID, 10)+`, `+strconv.FormatInt(account.ID, 10)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.EstimateTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"action_count":1`)
	require.NotContains(t, rec.Body.String(), `"action_count":2`)
}

func TestSocialAccountAdminSubmitTaskDeduplicatesAccountIDsWithoutIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "admin-submit-dedupe@example.com")
	account := client.SocialAccount.Create().
		SetName("@admin_submit_dedupe").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_submit_dedupe").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 1.0},
	}}
	handler := newSocialAccountAdminHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/social-accounts/tasks", bytes.NewBufferString(`{
		"account_ids": [`+strconv.FormatInt(account.ID, 10)+`, `+strconv.FormatInt(account.ID, 10)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

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

func TestSocialAccountAdminEstimateTaskRejectsNonPositiveAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "admin-estimate-invalid-id@example.com")
	account := client.SocialAccount.Create().
		SetName("@admin_estimate_invalid_id").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_estimate_invalid_id").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := newSocialAccountAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 1.0},
	}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/social-accounts/tasks/estimate", bytes.NewBufferString(`{
		"account_ids": [0, `+strconv.FormatInt(account.ID, 10)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.EstimateTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_ACCOUNT_ID_INVALID")
}

func TestSocialAccountAdminSubmitTaskRejectsNonPositiveAccountIDWithoutLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "admin-submit-invalid-id@example.com")
	account := client.SocialAccount.Create().
		SetName("@admin_submit_invalid_id").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_submit_invalid_id").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 1.0},
	}}
	handler := newSocialAccountAdminHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/social-accounts/tasks", bytes.NewBufferString(`{
		"account_ids": [-1, `+strconv.FormatInt(account.ID, 10)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_ACCOUNT_ID_INVALID")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func multipartBody(t *testing.T, filename, contentType, content string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}

func newSocialAccountAdminHandlerForTest(client *dbent.Client, userRepo *socialAccountAdminBillingUserRepo) *SocialAccountAdminHandler {
	subRepo := &socialAccountAdminSubscriptionRepo{}
	billing := service.NewSocialBillingService(userRepo, subRepo, nil, nil)
	return NewSocialAccountAdminHandler(
		service.NewSocialAccountService(client),
		service.NewSocialIPService(client),
		billing,
		nil,
	)
}

type socialAccountAdminBillingUserRepo struct {
	service.UserRepository
	users       map[int64]*service.User
	deductCalls int
}

func (r *socialAccountAdminBillingUserRepo) GetByID(_ context.Context, id int64) (*service.User, error) {
	user := r.users[id]
	if user == nil {
		return nil, service.ErrUserNotFound
	}
	out := *user
	return &out, nil
}

func (r *socialAccountAdminBillingUserRepo) DeductBalance(_ context.Context, id int64, amount float64) error {
	user := r.users[id]
	if user == nil {
		return service.ErrUserNotFound
	}
	r.deductCalls++
	user.Balance -= amount
	return nil
}

type socialAccountAdminSubscriptionRepo struct {
	service.UserSubscriptionRepository
}

func (r *socialAccountAdminSubscriptionRepo) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	return nil, nil
}

func (r *socialAccountAdminSubscriptionRepo) IncrementUsage(context.Context, int64, float64) error {
	return nil
}
