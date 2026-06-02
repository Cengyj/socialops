//go:build unit

package service

import (
	"encoding/json"
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
		require.Falsef(t, ok, "Group must not retain AI gateway field %s", field)
	}
}

func TestAPIKeyAuthGroupSnapshot_DoesNotSerializeAIGatewayFields(t *testing.T) {
	payload, err := json.Marshal(APIKeyAuthGroupSnapshot{
		ID:               7,
		Name:             "starter",
		Platform:         "social",
		Status:           StatusActive,
		SubscriptionType: SubscriptionTypeSubscription,
		RateMultiplier:   1,
	})
	require.NoError(t, err)

	var fields map[string]any
	require.NoError(t, json.Unmarshal(payload, &fields))
	forbiddenJSONKeys := []string{
		"allow_image_generation",
		"image_rate_independent",
		"image_rate_multiplier",
		"image_price_1k",
		"image_price_2k",
		"image_price_4k",
		"provider_code_only",
		"fallback_group_id",
		"fallback_group_id_on_invalid_request",
		"model_routing",
		"model_routing_enabled",
		"mcp_xml_inject",
		"supported_model_scopes",
		"allow_messages_dispatch",
		"default_mapped_model",
		"messages_dispatch_model_config",
	}
	for _, key := range forbiddenJSONKeys {
		require.NotContainsf(t, fields, key, "auth cache snapshot must not serialize AI gateway field %s", key)
	}
}
