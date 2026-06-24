//go:build integration

package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSocialOpsCoreMigrationsConvergeOnPostgresSchema(t *testing.T) {
	ctx := context.Background()

	socialAccounts := requirePostgresColumns(t, ctx, "social_accounts")
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
		require.Contains(t, socialAccounts, name, "social_accounts should include current SocialOps account column %s", name)
	}
	for _, removed := range []string{
		"source",
		"account_id",
		"bound_ip",
		"deleted_at",
		"user_workbench_deleted_at",
	} {
		require.NotContains(t, socialAccounts, removed, "social_accounts should not retain legacy column %s", removed)
	}

	socialAccountIndexes := requirePostgresIndexes(t, ctx, "social_accounts")
	uniqueAccountIndex := socialAccountIndexes["idx_social_accounts_platform_name_key_unique"]
	require.True(t, uniqueAccountIndex.Unique)
	require.Empty(t, uniqueAccountIndex.Predicate, "platform/name uniqueness should no longer depend on deleted_at")
	require.Contains(t, uniqueAccountIndex.Definition, "platform_key")
	require.Contains(t, uniqueAccountIndex.Definition, "name_key")
	require.Contains(t, socialAccountIndexes, "idx_social_accounts_platform_user_id")

	socialIps := requirePostgresColumns(t, ctx, "social_ips")
	for _, name := range []string{"user_id", "name", "ip_type", "endpoint", "status", "latency_ms", "last_check_at", "deleted_at"} {
		require.Contains(t, socialIps, name, "social_ips should include proxy column %s", name)
	}

	socialTaskLogs := requirePostgresColumns(t, ctx, "social_task_logs")
	for _, name := range []string{
		"social_account_id",
		"user_id",
		"action",
		"payload",
		"template_snapshot",
		"status",
		"charged_amount",
		"charge_status",
		"proxy_id",
		"idempotency_key",
	} {
		require.Contains(t, socialTaskLogs, name, "social_task_logs should include execution column %s", name)
	}
	require.Equal(t, "jsonb", socialTaskLogs["payload"].UDTName)
	require.Equal(t, "NO", socialTaskLogs["payload"].IsNullable)
	require.Contains(t, socialTaskLogs["payload"].Default, "{}")
	require.Equal(t, "jsonb", socialTaskLogs["template_snapshot"].UDTName)
	require.Equal(t, "NO", socialTaskLogs["template_snapshot"].IsNullable)
	require.Contains(t, socialTaskLogs["template_snapshot"].Default, "{}")

	socialTaskIndexes := requirePostgresIndexes(t, ctx, "social_task_logs")
	activeTaskIndex := socialTaskIndexes["idx_social_task_logs_one_active_per_account"]
	require.True(t, activeTaskIndex.Unique)
	require.Contains(t, activeTaskIndex.Predicate, "pending")
	require.Contains(t, activeTaskIndex.Predicate, "running")
	require.Contains(t, activeTaskIndex.Definition, "social_account_id")
	idempotencyIndex := socialTaskIndexes["idx_social_task_logs_user_account_action_idem_unique"]
	require.True(t, idempotencyIndex.Unique)
	require.Contains(t, idempotencyIndex.Predicate, "idempotency_key")
	require.Contains(t, idempotencyIndex.Predicate, "IS NOT NULL")

	settings := requirePostgresColumns(t, ctx, "settings")
	require.Contains(t, settings, "key")
	require.Contains(t, settings, "value")
	require.Equal(t, "text", settings["value"].UDTName)
}

func TestSocialOpsCoreMigrationsProvideTaskLogDefaultsAndActiveTaskGuard(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)

	var userID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, 'hash')
		RETURNING id
	`, "migration-defaults-user@example.test").Scan(&userID))

	var accountID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO social_accounts (
			name,
			platform,
			platform_key,
			name_key,
			identity_key,
			assigned_user_id,
			account_status,
			task_status
		)
		VALUES (
			'migration_defaults_account',
			'x_twitter',
			'x_twitter',
			'migration_defaults_account',
			'migration_defaults_account',
			$1,
			'available',
			'stored'
		)
		RETURNING id
	`, userID).Scan(&accountID))

	var payload string
	var templateSnapshot string
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO social_task_logs (social_account_id, user_id, action, status)
		VALUES ($1, $2, 'follow', 'pending')
		RETURNING payload::text, template_snapshot::text
	`, accountID, userID).Scan(&payload, &templateSnapshot))
	require.JSONEq(t, `{}`, payload)
	require.JSONEq(t, `{}`, templateSnapshot)

	_, err := tx.ExecContext(ctx, `
		INSERT INTO social_task_logs (social_account_id, user_id, action, status)
		VALUES ($1, $2, 'like', 'running')
	`, accountID, userID)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "unique")
}

type postgresColumnInfo struct {
	IsNullable string
	Default    string
	UDTName    string
}

func requirePostgresColumns(t *testing.T, ctx context.Context, table string) map[string]postgresColumnInfo {
	t.Helper()

	rows, err := integrationDB.QueryContext(ctx, `
		SELECT column_name, is_nullable, COALESCE(column_default, ''), udt_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = $1
	`, table)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	columns := make(map[string]postgresColumnInfo)
	for rows.Next() {
		var name string
		var info postgresColumnInfo
		require.NoError(t, rows.Scan(&name, &info.IsNullable, &info.Default, &info.UDTName))
		columns[name] = info
	}
	require.NoError(t, rows.Err())
	require.NotEmpty(t, columns, "table %s should exist", table)
	return columns
}

type postgresIndexInfo struct {
	Unique     bool
	Definition string
	Predicate  string
}

func requirePostgresIndexes(t *testing.T, ctx context.Context, table string) map[string]postgresIndexInfo {
	t.Helper()

	rows, err := integrationDB.QueryContext(ctx, `
		SELECT
			idx.relname,
			i.indisunique,
			pg_get_indexdef(i.indexrelid),
			COALESCE(pg_get_expr(i.indpred, i.indrelid), '')
		FROM pg_class tbl
		JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
		JOIN pg_index i ON i.indrelid = tbl.oid
		JOIN pg_class idx ON idx.oid = i.indexrelid
		WHERE ns.nspname = 'public'
		  AND tbl.relname = $1
	`, table)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	indexes := make(map[string]postgresIndexInfo)
	for rows.Next() {
		var name string
		var info postgresIndexInfo
		require.NoError(t, rows.Scan(&name, &info.Unique, &info.Definition, &info.Predicate))
		indexes[name] = info
	}
	require.NoError(t, rows.Err())
	require.NotEmpty(t, indexes, "table %s should have indexes", table)
	return indexes
}
