package dto

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Wei-Shaw/socialops/internal/service"
)

// TestPublicSettingsInjectionPayload_SchemaDoesNotDrift guarantees the SSR
// injection struct exposes every JSON field consumed by the frontend.
//
// Why this test exists: before we extracted a named PublicSettingsInjectionPayload
// type, the inline struct was manually kept in sync with dto.PublicSettings and
// drifted — feature flags added to dto.PublicSettings were not always mirrored
// into the SSR payload, so the frontend could read `undefined` on refresh until
// the async /api/v1/settings/public round-trip finished.
//
// This test compares the two JSON-tag sets and fails if either side drifts.
// Adding a new feature flag with only a DTO entry will fail until the injection
// struct is updated; injecting a field without a DTO contract fails too.
func TestPublicSettingsInjectionPayload_SchemaDoesNotDrift(t *testing.T) {
	injection := jsonTags(reflect.TypeOf(service.PublicSettingsInjectionPayload{}))
	dtoKeys := jsonTags(reflect.TypeOf(PublicSettings{}))

	var missing []string
	for key := range dtoKeys {
		if _, ok := injection[key]; ok {
			continue
		}
		missing = append(missing, key)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("service.PublicSettingsInjectionPayload is missing JSON fields present on dto.PublicSettings: %s\n"+
			"add the field to PublicSettingsInjectionPayload and GetPublicSettingsForInjection, or remove the unused public DTO field.",
			strings.Join(missing, ", "))
	}

	var extra []string
	for key := range injection {
		if _, ok := dtoKeys[key]; ok {
			continue
		}
		extra = append(extra, key)
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		t.Fatalf("service.PublicSettingsInjectionPayload has JSON fields missing from dto.PublicSettings: %s\n"+
			"add the field to dto.PublicSettings and SettingHandler.GetPublicSettings, or remove the unused injection field.",
			strings.Join(extra, ", "))
	}
}

func jsonTags(t reflect.Type) map[string]struct{} {
	out := make(map[string]struct{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.SplitN(tag, ",", 2)[0]
		if name == "" {
			continue
		}
		out[name] = struct{}{}
	}
	return out
}
