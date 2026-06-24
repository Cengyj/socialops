//go:build unit

package service

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupShape_IsSocialOpsSubscriptionGroupOnly(t *testing.T) {
	groupType := reflect.TypeOf(Group{})
	forbiddenFields := []string{
		"AllowImageGeneration",
		"ImageRateIndependent",
		"ImageRateMultiplier",
		"ImagePrice1K",
		"ImagePrice2K",
		"ImagePrice4K",
		"ProviderCodeOnly",
		"FallbackGroupID",
		"FallbackGroupIDOnInvalidRequest",
		"ModelRouting",
		"ModelRoutingEnabled",
		"MCPXMLInject",
		"SupportedModelScopes",
		"AllowMessagesDispatch",
		"RequireOAuthOnly",
		"RequirePrivacySet",
		"DefaultMappedModel",
		"MessagesDispatchModelConfig",
	}

	for _, field := range forbiddenFields {
		_, ok := groupType.FieldByName(field)
		require.Falsef(t, ok, "Group must not retain removed gateway field %s", field)
	}
}
