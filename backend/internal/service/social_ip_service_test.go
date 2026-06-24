//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestSocialIPServiceListByUserNormalizesFilters(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	owner := client.User.Create().
		SetEmail("proxy-list-owner@example.com").
		SetPasswordHash("hash").
		SetRole("user").
		SetStatus("active").
		SaveX(ctx)
	other := client.User.Create().
		SetEmail("proxy-list-other@example.com").
		SetPasswordHash("hash").
		SetRole("user").
		SetStatus("active").
		SaveX(ctx)

	client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("qa owner proxy").
		SetIPType(SocialIPTypeResidential).
		SetEndpoint("http://owner.proxy.example:8080").
		SetStatus(SocialIPStatusOnline).
		SetRemark("primary").
		SaveX(ctx)
	client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("qa owner static proxy").
		SetIPType(SocialIPTypeStatic).
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)
	client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("qa offline proxy").
		SetIPType(SocialIPTypeResidential).
		SetStatus(SocialIPStatusOffline).
		SaveX(ctx)
	client.SocialIP.Create().
		SetUserID(other.ID).
		SetName("qa owner proxy for another user").
		SetIPType(SocialIPTypeResidential).
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)

	items, result, err := NewSocialIPService(client).ListByUser(ctx, owner.ID, pagination.PaginationParams{Page: 1, PageSize: 20}, SocialIPListFilters{
		Status: " online ",
		IPType: " residential ",
		Search: " owner ",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	require.Equal(t, "qa owner proxy", items[0].Name)
}
