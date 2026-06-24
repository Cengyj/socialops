package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type groupHandlerRepoStub struct {
	platform         string
	status           string
	subscriptionType string
	search           string
	isExclusive      *bool
}

func (s *groupHandlerRepoStub) Create(context.Context, *service.Group) error {
	panic("unexpected Create call")
}

func (s *groupHandlerRepoStub) GetByID(context.Context, int64) (*service.Group, error) {
	panic("unexpected GetByID call")
}

func (s *groupHandlerRepoStub) GetByIDLite(context.Context, int64) (*service.Group, error) {
	panic("unexpected GetByIDLite call")
}

func (s *groupHandlerRepoStub) Update(context.Context, *service.Group) error {
	panic("unexpected Update call")
}

func (s *groupHandlerRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *groupHandlerRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected DeleteCascade call")
}

func (s *groupHandlerRepoStub) List(context.Context, pagination.PaginationParams) ([]service.Group, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *groupHandlerRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, platform, status, subscriptionType, search string, isExclusive *bool) ([]service.Group, *pagination.PaginationResult, error) {
	s.platform = platform
	s.status = status
	s.subscriptionType = subscriptionType
	s.search = search
	s.isExclusive = isExclusive

	return []service.Group{
			{
				ID:                  10,
				Name:                "Social Starter",
				Platform:            "x_twitter",
				Status:              service.StatusActive,
				SubscriptionType:    service.SubscriptionTypeSubscription,
				RateMultiplier:      1,
				DefaultValidityDays: 30,
				Hydrated:            true,
			},
		}, paginationResultForGroupHandlerTest(42, params),
		nil
}

func (s *groupHandlerRepoStub) ListActive(context.Context) ([]service.Group, error) {
	panic("unexpected ListActive call")
}

func (s *groupHandlerRepoStub) ListActiveByPlatform(context.Context, string) ([]service.Group, error) {
	panic("unexpected ListActiveByPlatform call")
}

func (s *groupHandlerRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected ExistsByName call")
}

func (s *groupHandlerRepoStub) UpdateSortOrders(context.Context, []service.GroupSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders call")
}

func paginationResultForGroupHandlerTest(total int64, params pagination.PaginationParams) *pagination.PaginationResult {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	return &pagination.PaginationResult{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Pages:    int((total + int64(pageSize) - 1) / int64(pageSize)),
	}
}

func TestGroupHandlerListPassesSubscriptionTypeToRepository(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &groupHandlerRepoStub{}
	handler := NewGroupHandler(repo)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups?page=2&page_size=5&platform=x_twitter&status=active&subscription_type=subscription&search=starter&is_exclusive=false", nil)

	handler.List(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "x_twitter", repo.platform)
	require.Equal(t, service.StatusActive, repo.status)
	require.Equal(t, service.SubscriptionTypeSubscription, repo.subscriptionType)
	require.Equal(t, "starter", repo.search)
	require.NotNil(t, repo.isExclusive)
	require.False(t, *repo.isExclusive)
	require.Contains(t, rec.Body.String(), `"total":42`)
	require.Contains(t, rec.Body.String(), `"Social Starter"`)
}
