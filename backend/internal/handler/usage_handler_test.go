//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/domain"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageHandlerRepoStub struct {
	listParams   pagination.PaginationParams
	listFilters  usagestats.UsageLogFilters
	statsFilters usagestats.UsageLogFilters
	listItems    []service.UsageLog
	getByIDItem  *service.UsageLog
}

type usageHandlerMediaResolverStub struct {
	ref      *domain.SocialTaskMediaRef
	resolved *service.ResolvedSocialTaskMedia
	err      error
}

func (s *usageHandlerRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	s.listParams = params
	s.listFilters = filters
	return s.listItems, &pagination.PaginationResult{Total: int64(len(s.listItems)), Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (s *usageHandlerRepoStub) GetStatsWithFilters(_ context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	s.statsFilters = filters
	return &usagestats.UsageStats{}, nil
}

func (s *usageHandlerRepoStub) GetByID(context.Context, int64, int64) (*service.UsageLog, error) {
	if s.getByIDItem == nil {
		return nil, service.ErrUsageLogNotFound
	}
	return s.getByIDItem, nil
}

func (s *usageHandlerMediaResolverStub) Resolve(_ context.Context, _ int64, ref *domain.SocialTaskMediaRef) (*service.ResolvedSocialTaskMedia, error) {
	if ref != nil {
		cloned := *ref
		s.ref = &cloned
	} else {
		s.ref = nil
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.resolved, nil
}

func TestUsageHandlerListPassesSocialTaskFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &usageHandlerRepoStub{}
	handler := NewUsageHandler(service.NewUsageService(repo, nil), nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/usage?page=2&page_size=5&sort_by=operation&sort_order=asc&operation=follow&status=success&start_date=2026-06-01T00:00:00Z&end_date=2026-06-02T00:00:00Z", nil)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.List(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 2, repo.listParams.Page)
	require.Equal(t, 5, repo.listParams.PageSize)
	require.Equal(t, "operation", repo.listParams.SortBy)
	require.Equal(t, "asc", repo.listParams.SortOrder)
	require.Equal(t, int64(42), repo.listFilters.UserID)
	require.Equal(t, "follow", repo.listFilters.Model)
	require.Equal(t, "success", repo.listFilters.Status)
	require.NotNil(t, repo.listFilters.StartTime)
	require.NotNil(t, repo.listFilters.EndTime)
	require.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), *repo.listFilters.StartTime)
	require.Equal(t, time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), *repo.listFilters.EndTime)
}

func TestUsageHandlerStatsPassesSocialTaskFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &usageHandlerRepoStub{}
	handler := NewUsageHandler(service.NewUsageService(repo, nil), nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/usage/stats?operation=like&status=failed&start_date=2026-06-01T00:00:00Z&end_date=2026-06-02T00:00:00Z", nil)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.Stats(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.statsFilters.UserID)
	require.Equal(t, "like", repo.statsFilters.Model)
	require.Equal(t, "failed", repo.statsFilters.Status)
	require.NotNil(t, repo.statsFilters.StartTime)
	require.NotNil(t, repo.statsFilters.EndTime)
}

func TestUsageHandlerListReturnsSafeSocialTaskFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawResult := "authorization Bearer abc token=secret proxy=http://127.0.0.1:8080 trace_id=trace-123"
	target := "https://x.com/openai/status/1"
	content := "hello world"
	repo := &usageHandlerRepoStub{listItems: []service.UsageLog{{
		ID:              101,
		UserID:          42,
		SocialAccountID: 7,
		Platform:        "x_twitter",
		AccountName:     "delivery-account",
		Operation:       service.SocialTaskActionFollow,
		Status:          service.SocialTaskLogStatusFailed,
		Quantity:        1,
		Cost:            0,
		ChargeStatus:    service.SocialTaskChargeStatusNotCharged,
		Target:          &target,
		Content:         &content,
		Payload: &domain.SocialTaskPayload{
			Target: target,
			Post: &domain.SocialPostPayload{
				Text:         content,
				QuotePostURL: "https://x.com/openai/status/2",
				Media: []domain.SocialTaskMediaRef{{
					Source:      "inline",
					URL:         "data:image/png;base64,QUJD",
					StorageKey:  "media/private/post-image-1.png",
					ContentType: "image/png",
					FileName:    "post-image-1.png",
					SHA256:      "secret-sha",
					ByteSize:    1234,
					Width:       400,
					Height:      400,
				}},
			},
			Avatar: &domain.SocialTaskMediaRef{
				Source:      "inline",
				URL:         "data:image/png;base64,QUJD",
				StorageKey:  "media/private/avatar.png",
				ContentType: "image/png",
				FileName:    "avatar.png",
				SHA256:      "avatar-sha",
				ByteSize:    2048,
				Width:       400,
				Height:      400,
			},
		},
		TemplateSnapshot: &domain.SocialTaskTemplateSnapshot{
			TemplateID:   "tmpl_1",
			TemplateName: "Rich post",
			TemplateType: service.SocialTaskActionPost,
			Params: domain.SocialTaskTemplateParams{
				Targets:      []string{target},
				Contents:     []string{content},
				QuotePostURL: "https://x.com/openai/status/2",
				Media: []domain.SocialTaskMediaRef{{
					Source:      "inline",
					URL:         "data:image/png;base64,QUJD",
					StorageKey:  "media/private/post-image-1.png",
					ContentType: "image/png",
					FileName:    "post-image-1.png",
					SHA256:      "secret-sha",
					ByteSize:    1234,
					Width:       400,
					Height:      400,
				}},
			},
		},
		ResultMessage:   &rawResult,
		CreatedAt:       time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}}}
	handler := NewUsageHandler(service.NewUsageService(repo, nil), nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/usage", nil)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.List(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data struct {
			Items []struct {
				SocialAccountID int64   `json:"social_account_id"`
				Platform        string  `json:"platform"`
				AccountName     string  `json:"account_name"`
				ChargeStatus    string  `json:"charge_status"`
				Target          *string `json:"target"`
				Content         *string `json:"content"`
				Payload         *struct {
					Target string `json:"target"`
					Post   *struct {
						Text         string `json:"text"`
						QuotePostURL string `json:"quote_post_url"`
						Media        []struct {
							Source      string `json:"source"`
							URL         string `json:"url"`
							StorageKey  string `json:"storage_key"`
							ContentType string `json:"content_type"`
							FileName    string `json:"file_name"`
							SHA256      string `json:"sha256"`
							ByteSize    int64  `json:"byte_size"`
							Width       int    `json:"width"`
							Height      int    `json:"height"`
						} `json:"media"`
					} `json:"post"`
					Avatar *struct {
						URL         string `json:"url"`
						StorageKey  string `json:"storage_key"`
						FileName    string `json:"file_name"`
						ContentType string `json:"content_type"`
						SHA256      string `json:"sha256"`
						ByteSize    int64  `json:"byte_size"`
						Width       int    `json:"width"`
						Height      int    `json:"height"`
					} `json:"avatar"`
				} `json:"payload"`
				TemplateSnapshot *struct {
					TemplateID   string `json:"template_id"`
					TemplateName string `json:"template_name"`
					TemplateType string `json:"template_type"`
					Params       struct {
						Targets      []string `json:"targets"`
						Contents     []string `json:"contents"`
						QuotePostURL string   `json:"quote_post_url"`
						Media        []struct {
							URL         string `json:"url"`
							StorageKey  string `json:"storage_key"`
							FileName    string `json:"file_name"`
							ContentType string `json:"content_type"`
							SHA256      string `json:"sha256"`
							ByteSize    int64  `json:"byte_size"`
							Width       int    `json:"width"`
							Height      int    `json:"height"`
						} `json:"media"`
					} `json:"params"`
				} `json:"template_snapshot"`
				ResultMessage   *string `json:"result_message"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, int64(7), body.Data.Items[0].SocialAccountID)
	require.Equal(t, "x_twitter", body.Data.Items[0].Platform)
	require.Equal(t, "delivery-account", body.Data.Items[0].AccountName)
	require.Equal(t, service.SocialTaskChargeStatusNotCharged, body.Data.Items[0].ChargeStatus)
	require.NotNil(t, body.Data.Items[0].Target)
	require.Equal(t, target, *body.Data.Items[0].Target)
	require.NotNil(t, body.Data.Items[0].Content)
	require.Equal(t, content, *body.Data.Items[0].Content)
	require.NotNil(t, body.Data.Items[0].Payload)
	require.Equal(t, target, body.Data.Items[0].Payload.Target)
	require.NotNil(t, body.Data.Items[0].Payload.Post)
	require.Equal(t, content, body.Data.Items[0].Payload.Post.Text)
	require.Equal(t, "https://x.com/openai/status/2", body.Data.Items[0].Payload.Post.QuotePostURL)
	require.Len(t, body.Data.Items[0].Payload.Post.Media, 1)
	require.Equal(t, "post-image-1.png", body.Data.Items[0].Payload.Post.Media[0].FileName)
	require.Equal(t, "image/png", body.Data.Items[0].Payload.Post.Media[0].ContentType)
	require.EqualValues(t, 1234, body.Data.Items[0].Payload.Post.Media[0].ByteSize)
	require.Empty(t, body.Data.Items[0].Payload.Post.Media[0].URL)
	require.Empty(t, body.Data.Items[0].Payload.Post.Media[0].StorageKey)
	require.Empty(t, body.Data.Items[0].Payload.Post.Media[0].SHA256)
	require.NotNil(t, body.Data.Items[0].Payload.Avatar)
	require.Equal(t, "avatar.png", body.Data.Items[0].Payload.Avatar.FileName)
	require.Empty(t, body.Data.Items[0].Payload.Avatar.URL)
	require.Empty(t, body.Data.Items[0].Payload.Avatar.StorageKey)
	require.Empty(t, body.Data.Items[0].Payload.Avatar.SHA256)
	require.NotNil(t, body.Data.Items[0].TemplateSnapshot)
	require.Equal(t, "tmpl_1", body.Data.Items[0].TemplateSnapshot.TemplateID)
	require.Equal(t, "Rich post", body.Data.Items[0].TemplateSnapshot.TemplateName)
	require.Equal(t, service.SocialTaskActionPost, body.Data.Items[0].TemplateSnapshot.TemplateType)
	require.Equal(t, []string{target}, body.Data.Items[0].TemplateSnapshot.Params.Targets)
	require.Equal(t, []string{content}, body.Data.Items[0].TemplateSnapshot.Params.Contents)
	require.Equal(t, "https://x.com/openai/status/2", body.Data.Items[0].TemplateSnapshot.Params.QuotePostURL)
	require.Len(t, body.Data.Items[0].TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "post-image-1.png", body.Data.Items[0].TemplateSnapshot.Params.Media[0].FileName)
	require.Empty(t, body.Data.Items[0].TemplateSnapshot.Params.Media[0].URL)
	require.Empty(t, body.Data.Items[0].TemplateSnapshot.Params.Media[0].StorageKey)
	require.Empty(t, body.Data.Items[0].TemplateSnapshot.Params.Media[0].SHA256)
	require.NotNil(t, body.Data.Items[0].ResultMessage)
	require.Equal(t, "账号认证信息不可用，本次未扣费", *body.Data.Items[0].ResultMessage)
	require.NotContains(t, rec.Body.String(), "Bearer abc")
	require.NotContains(t, rec.Body.String(), "127.0.0.1")
	require.NotContains(t, rec.Body.String(), "trace-123")
	require.NotContains(t, rec.Body.String(), "data:image/png;base64")
	require.NotContains(t, rec.Body.String(), "media/private")
	require.NotContains(t, rec.Body.String(), "secret-sha")
}

func TestUsageHandlerGetByIDReturnsSafeStructuredTaskDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	target := "https://x.com/openai/status/1"
	content := "hello world"
	result := "follow succeeded"
	chargeSource := service.SocialTaskChargeSourceSubscription
	proxySnapshot := `{"id":8,"name":"proxy-a","endpoint":"http://user:pass@proxy.local:8080","status":"online"}`
	billingRequestID := "sub:task-501"
	idempotencyKey := "usage-detail-501"
	executedAt := time.Date(2026, 6, 1, 0, 0, 2, 0, time.UTC)
	repo := &usageHandlerRepoStub{
		getByIDItem: &service.UsageLog{
			ID:              501,
			UserID:          42,
			SocialAccountID: 9,
			Platform:        "x_twitter",
			AccountName:     "x-main",
			Operation:       service.SocialTaskActionPost,
			Status:          service.SocialTaskLogStatusSuccess,
			Quantity:        1,
			Cost:            service.SocialTaskUnitPrice,
			ChargeStatus:    service.SocialTaskChargeStatusCharged,
			ChargeSource:    &chargeSource,
			Target:          &target,
			Content:         &content,
			Payload: &domain.SocialTaskPayload{
				Target: target,
				Post: &domain.SocialPostPayload{
					Text:         content,
					QuotePostURL: "https://x.com/openai/status/2",
					Media: []domain.SocialTaskMediaRef{{
						Source:      "inline",
						URL:         "data:image/png;base64,QUJD",
						StorageKey:  "media/private/post-image-1.png",
						ContentType: "image/png",
						FileName:    "post-image-1.png",
						SHA256:      "secret-sha",
						ByteSize:    1234,
						Width:       400,
						Height:      400,
					}},
				},
				Profile: &domain.SocialProfileUpdateParams{
					DisplayName: "OpenAI News",
					Location:    "San Francisco",
					URL:         "https://openai.com",
				},
				Avatar: &domain.SocialTaskMediaRef{
					Source:      "inline",
					URL:         "data:image/png;base64,QUJD",
					StorageKey:  "media/private/avatar.png",
					ContentType: "image/png",
					FileName:    "avatar.png",
					ByteSize:    2048,
					Width:       400,
					Height:      400,
				},
			},
			TemplateSnapshot: &domain.SocialTaskTemplateSnapshot{
				TemplateID:   "tmpl_1",
				TemplateName: "Rich post",
				TemplateType: service.SocialTaskActionPost,
				Params: domain.SocialTaskTemplateParams{
					Targets:      []string{target},
					Contents:     []string{content},
					QuotePostURL: "https://x.com/openai/status/2",
					Media: []domain.SocialTaskMediaRef{{
						Source:      "inline",
						URL:         "data:image/png;base64,QUJD",
						StorageKey:  "media/private/post-image-1.png",
						ContentType: "image/png",
						FileName:    "post-image-1.png",
						SHA256:      "secret-sha",
						ByteSize:    1234,
						Width:       400,
						Height:      400,
					}},
					Avatar: &domain.SocialTaskMediaRef{
						Source:      "inline",
						URL:         "data:image/png;base64,QUJD",
						StorageKey:  "media/private/avatar.png",
						ContentType: "image/png",
						FileName:    "avatar.png",
						SHA256:      "avatar-sha",
						ByteSize:    2048,
						Width:       400,
						Height:      400,
					},
				},
			},
			ResultMessage: &result,
			ProxySnapshot: &proxySnapshot,
			BillingRequestID: &billingRequestID,
			IdempotencyKey: &idempotencyKey,
			CreatedAt:     time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			CompletedAt:   &executedAt,
		},
	}
	handler := NewUsageHandler(service.NewUsageService(repo, nil), nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/usage/501", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "501"}}
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.GetByID(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data struct {
			ChargeSource     *string `json:"charge_source"`
			ProxySnapshot    *string `json:"proxy_snapshot"`
			BillingRequestID *string `json:"billing_request_id"`
			IdempotencyKey   *string `json:"idempotency_key"`
			CompletedAt      *string `json:"completed_at"`
			Target        *string `json:"target"`
			Content       *string `json:"content"`
			ResultMessage *string `json:"result_message"`
			Payload       *struct {
				Target string `json:"target"`
				Post   *struct {
					Text         string `json:"text"`
					QuotePostURL string `json:"quote_post_url"`
					Media        []struct {
						Source      string `json:"source"`
						URL         string `json:"url"`
						StorageKey  string `json:"storage_key"`
						ContentType string `json:"content_type"`
						FileName    string `json:"file_name"`
						SHA256      string `json:"sha256"`
						ByteSize    int64  `json:"byte_size"`
						Width       int    `json:"width"`
						Height      int    `json:"height"`
					} `json:"media"`
				} `json:"post"`
				Profile *struct {
					DisplayName string `json:"display_name"`
					Location    string `json:"location"`
					URL         string `json:"url"`
				} `json:"profile"`
				Avatar *struct {
					URL         string `json:"url"`
					StorageKey  string `json:"storage_key"`
					FileName    string `json:"file_name"`
					ContentType string `json:"content_type"`
					ByteSize    int64  `json:"byte_size"`
					Width       int    `json:"width"`
					Height      int    `json:"height"`
				} `json:"avatar"`
			} `json:"payload"`
			TemplateSnapshot *struct {
				TemplateID   string `json:"template_id"`
				TemplateName string `json:"template_name"`
				TemplateType string `json:"template_type"`
				Params       struct {
					Targets      []string `json:"targets"`
					Contents     []string `json:"contents"`
					QuotePostURL string   `json:"quote_post_url"`
					Media        []struct {
						URL         string `json:"url"`
						StorageKey  string `json:"storage_key"`
						FileName    string `json:"file_name"`
						ContentType string `json:"content_type"`
						ByteSize    int64  `json:"byte_size"`
						Width       int    `json:"width"`
						Height      int    `json:"height"`
					} `json:"media"`
					Avatar *struct {
						URL         string `json:"url"`
						StorageKey  string `json:"storage_key"`
						FileName    string `json:"file_name"`
						ContentType string `json:"content_type"`
						ByteSize    int64  `json:"byte_size"`
						Width       int    `json:"width"`
						Height      int    `json:"height"`
					} `json:"avatar"`
				} `json:"params"`
			} `json:"template_snapshot"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotNil(t, body.Data.Target)
	require.Equal(t, target, *body.Data.Target)
	require.NotNil(t, body.Data.Content)
	require.Equal(t, content, *body.Data.Content)
	require.NotNil(t, body.Data.ResultMessage)
	require.Equal(t, "follow succeeded", *body.Data.ResultMessage)
	require.NotNil(t, body.Data.ChargeSource)
	require.Equal(t, service.SocialTaskChargeSourceSubscription, *body.Data.ChargeSource)
	require.NotNil(t, body.Data.ProxySnapshot)
	require.Contains(t, *body.Data.ProxySnapshot, `"name":"proxy-a"`)
	require.Contains(t, *body.Data.ProxySnapshot, `"endpoint":"http://proxy.local:8080"`)
	require.NotContains(t, *body.Data.ProxySnapshot, "user:pass@")
	require.NotNil(t, body.Data.BillingRequestID)
	require.Equal(t, billingRequestID, *body.Data.BillingRequestID)
	require.NotNil(t, body.Data.IdempotencyKey)
	require.Equal(t, idempotencyKey, *body.Data.IdempotencyKey)
	require.NotNil(t, body.Data.CompletedAt)
	require.Equal(t, executedAt.Format(time.RFC3339), *body.Data.CompletedAt)
	require.NotNil(t, body.Data.Payload)
	require.Equal(t, target, body.Data.Payload.Target)
	require.NotNil(t, body.Data.Payload.Post)
	require.Equal(t, content, body.Data.Payload.Post.Text)
	require.Equal(t, "https://x.com/openai/status/2", body.Data.Payload.Post.QuotePostURL)
	require.Len(t, body.Data.Payload.Post.Media, 1)
	require.Equal(t, "inline", body.Data.Payload.Post.Media[0].Source)
	require.Equal(t, "post-image-1.png", body.Data.Payload.Post.Media[0].FileName)
	require.Equal(t, "image/png", body.Data.Payload.Post.Media[0].ContentType)
	require.EqualValues(t, 1234, body.Data.Payload.Post.Media[0].ByteSize)
	require.Equal(t, 400, body.Data.Payload.Post.Media[0].Width)
	require.Equal(t, 400, body.Data.Payload.Post.Media[0].Height)
	require.Empty(t, body.Data.Payload.Post.Media[0].URL)
	require.Empty(t, body.Data.Payload.Post.Media[0].StorageKey)
	require.Empty(t, body.Data.Payload.Post.Media[0].SHA256)
	require.NotNil(t, body.Data.Payload.Profile)
	require.Equal(t, "OpenAI News", body.Data.Payload.Profile.DisplayName)
	require.Equal(t, "San Francisco", body.Data.Payload.Profile.Location)
	require.Equal(t, "https://openai.com", body.Data.Payload.Profile.URL)
	require.NotNil(t, body.Data.Payload.Avatar)
	require.Equal(t, "avatar.png", body.Data.Payload.Avatar.FileName)
	require.Equal(t, "image/png", body.Data.Payload.Avatar.ContentType)
	require.EqualValues(t, 2048, body.Data.Payload.Avatar.ByteSize)
	require.Empty(t, body.Data.Payload.Avatar.URL)
	require.Empty(t, body.Data.Payload.Avatar.StorageKey)
	require.NotNil(t, body.Data.TemplateSnapshot)
	require.Equal(t, "tmpl_1", body.Data.TemplateSnapshot.TemplateID)
	require.Equal(t, "Rich post", body.Data.TemplateSnapshot.TemplateName)
	require.Equal(t, service.SocialTaskActionPost, body.Data.TemplateSnapshot.TemplateType)
	require.Equal(t, []string{target}, body.Data.TemplateSnapshot.Params.Targets)
	require.Equal(t, []string{content}, body.Data.TemplateSnapshot.Params.Contents)
	require.Equal(t, "https://x.com/openai/status/2", body.Data.TemplateSnapshot.Params.QuotePostURL)
	require.Len(t, body.Data.TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "post-image-1.png", body.Data.TemplateSnapshot.Params.Media[0].FileName)
	require.Equal(t, "image/png", body.Data.TemplateSnapshot.Params.Media[0].ContentType)
	require.Empty(t, body.Data.TemplateSnapshot.Params.Media[0].URL)
	require.Empty(t, body.Data.TemplateSnapshot.Params.Media[0].StorageKey)
	require.NotNil(t, body.Data.TemplateSnapshot.Params.Avatar)
	require.Equal(t, "avatar.png", body.Data.TemplateSnapshot.Params.Avatar.FileName)
	require.Empty(t, body.Data.TemplateSnapshot.Params.Avatar.URL)
	require.Empty(t, body.Data.TemplateSnapshot.Params.Avatar.StorageKey)
	require.NotContains(t, rec.Body.String(), "data:image/png;base64")
	require.NotContains(t, rec.Body.String(), "media/private")
	require.NotContains(t, rec.Body.String(), "secret-sha")
	require.NotContains(t, rec.Body.String(), "user:pass@")
}

func TestUsageHandlerGetByIDRedactsLegacyPlainProxySnapshotValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proxySnapshot := "http://user:pass@proxy.local:8080"
	repo := &usageHandlerRepoStub{
		getByIDItem: &service.UsageLog{
			ID:              777,
			UserID:          42,
			SocialAccountID: 9,
			Platform:        "x_twitter",
			AccountName:     "x-main",
			Operation:       service.SocialTaskActionFollow,
			Status:          service.SocialTaskLogStatusSuccess,
			Quantity:        1,
			Cost:            service.SocialTaskUnitPrice,
			ChargeStatus:    service.SocialTaskChargeStatusCharged,
			ProxySnapshot:   &proxySnapshot,
			CreatedAt:       time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	handler := NewUsageHandler(service.NewUsageService(repo, nil), nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/usage/777", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "777"}}
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.GetByID(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "http://proxy.local:8080")
	require.NotContains(t, rec.Body.String(), "user:pass@")
}

func TestUsageHandlerPreviewTaskMediaRequiresAuthSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUsageHandler(nil, nil)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/usage/501/media?scope=payload&section=post&index=0", nil)
	ginCtx.Params = gin.Params{{Key: "id", Value: "501"}}

	handler.PreviewTaskMedia(ginCtx)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), "User not authenticated")
}

func TestUsageHandlerPreviewTaskMediaStreamsOwnedTaskAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &usageHandlerRepoStub{
		getByIDItem: &service.UsageLog{
			ID:     501,
			UserID: 42,
			Payload: &domain.SocialTaskPayload{
				Post: &domain.SocialPostPayload{
					Media: []domain.SocialTaskMediaRef{{
						Source:      "library",
						StorageKey:  "social-task/42/post.png",
						ContentType: "image/png",
						FileName:    "post.png",
					}},
				},
			},
		},
	}
	resolver := &usageHandlerMediaResolverStub{
		resolved: &service.ResolvedSocialTaskMedia{
			ContentType: "image/png",
			FileName:    "post.png",
			Body:        []byte("preview"),
		},
	}
	handler := NewUsageHandler(service.NewUsageService(repo, nil).WithMediaResolver(resolver), nil)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	ginCtx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/usage/501/media?scope=payload&section=post&index=0",
		nil,
	)
	ginCtx.Params = gin.Params{{Key: "id", Value: "501"}}

	handler.PreviewTaskMedia(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), "post.png")
	require.Equal(t, "social-task/42/post.png", resolver.ref.StorageKey)
	require.NotEmpty(t, rec.Body.Bytes())
}

func TestUsageHandlerPreviewTaskMediaRejectsInvalidLocator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUsageHandler(service.NewUsageService(&usageHandlerRepoStub{
		getByIDItem: &service.UsageLog{ID: 501, UserID: 42},
	}, nil).WithMediaResolver(&usageHandlerMediaResolverStub{}), nil)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	ginCtx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/usage/501/media?scope=payload&section=post&index=oops",
		nil,
	)
	ginCtx.Params = gin.Params{{Key: "id", Value: "501"}}

	handler.PreviewTaskMedia(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "USAGE_TASK_MEDIA_LOCATOR_INVALID")
}
