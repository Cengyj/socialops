package schema

import (
	"testing"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"github.com/stretchr/testify/require"
)

func TestSocialAccountSchemaMatchesCurrentPoolContract(t *testing.T) {
	fields := SocialAccount{}.Fields()

	for _, name := range []string{
		"platform_key",
		"name_key",
		"identity_kind",
		"identity_key",
		"platform_user_id",
		"auth_cookie",
		"execution_auth",
		"default_proxy_snapshot",
		"assigned_user_id",
	} {
		require.NotNil(t, requireEntFieldDescriptor(t, fields, name))
	}

	for _, removed := range []string{
		"source",
		"account_id",
		"bound_ip",
		"deleted_at",
		"user_workbench_deleted_at",
	} {
		requireNoEntField(t, fields, removed)
	}

	requireEntIndex(t, SocialAccount{}.Indexes(), entIndexExpectation{
		fields:     []string{"platform_key", "name_key"},
		unique:     true,
		storageKey: "idx_social_accounts_platform_name_key_unique",
	})
	requireEntIndex(t, SocialAccount{}.Indexes(), entIndexExpectation{
		fields:     []string{"platform_user_id"},
		storageKey: "idx_social_accounts_platform_user_id",
	})
}

func TestSocialTaskLogSchemaMatchesCurrentExecutionContract(t *testing.T) {
	fields := SocialTaskLog{}.Fields()

	action := requireEntFieldDescriptor(t, fields, "action")
	require.Contains(t, action.Comment, "update_profile")
	require.Contains(t, action.Comment, "update_avatar")
	require.Contains(t, action.Comment, "update_banner")
	require.NotContains(t, action.Comment, "message")

	for _, name := range []string{"payload", "template_snapshot"} {
		descriptor := requireEntFieldDescriptor(t, fields, name)
		require.False(t, descriptor.Optional, "task snapshot field %s must be non-null in Ent schema", name)
		require.NotNil(t, descriptor.Default, "task snapshot field %s should default to an empty object", name)
		require.Equal(t, "jsonb", descriptor.SchemaType[dialect.Postgres])
	}

	requireEntIndex(t, SocialTaskLog{}.Indexes(), entIndexExpectation{
		fields:     []string{"social_account_id"},
		unique:     true,
		storageKey: "idx_social_task_logs_one_active_per_account",
		where:      "status IN ('pending', 'running')",
	})
	requireEntIndex(t, SocialTaskLog{}.Indexes(), entIndexExpectation{
		fields:     []string{"user_id", "social_account_id", "action", "idempotency_key"},
		unique:     true,
		storageKey: "idx_social_task_logs_user_account_action_idem_unique",
		where:      "idempotency_key IS NOT NULL",
	})
}

func TestTaskSettingsSchemaUsesSettingsKeyValueStore(t *testing.T) {
	fields := Setting{}.Fields()

	key := requireEntFieldDescriptor(t, fields, "key")
	require.True(t, key.Unique)
	require.Equal(t, 100, key.Size)

	value := requireEntFieldDescriptor(t, fields, "value")
	require.Equal(t, "text", value.SchemaType[dialect.Postgres])

	updatedAt := requireEntFieldDescriptor(t, fields, "updated_at")
	require.Equal(t, "timestamptz", updatedAt.SchemaType[dialect.Postgres])
}

type entIndexExpectation struct {
	fields     []string
	unique     bool
	storageKey string
	where      string
}

func requireNoEntField(t *testing.T, fields []ent.Field, name string) {
	t.Helper()

	for _, entField := range fields {
		if entField.Descriptor().Name == name {
			require.Failf(t, "unexpected field descriptor", "schema should not include field %s", name)
		}
	}
}

func requireEntIndex(t *testing.T, indexes []ent.Index, expected entIndexExpectation) {
	t.Helper()

	for _, entIndex := range indexes {
		descriptor := entIndex.Descriptor()
		if !sameStrings(descriptor.Fields, expected.fields) {
			continue
		}
		if descriptor.Unique != expected.unique {
			continue
		}
		if expected.storageKey != "" && descriptor.StorageKey != expected.storageKey {
			continue
		}
		if expected.where != "" && entIndexWhere(descriptor.Annotations) != expected.where {
			continue
		}
		return
	}

	require.Failf(t, "missing index descriptor", "schema should include index %+v", expected)
}

func entIndexWhere(annotations []entschema.Annotation) string {
	for _, annotation := range annotations {
		if indexAnnotation, ok := annotation.(*entsql.IndexAnnotation); ok {
			return indexAnnotation.Where
		}
	}
	return ""
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
