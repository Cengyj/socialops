package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration112UsesIdempotentAddColumn(t *testing.T) {
	content, err := FS.ReadFile("112_add_payment_order_provider_key_snapshot.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS provider_key VARCHAR(30)")
	require.NotContains(t, sql, "ADD COLUMN provider_key VARCHAR(30);")
}

func TestMigration118DoesNotForceOverwriteAuthSourceGrantDefaults(t *testing.T) {
	content, err := FS.ReadFile("118_wechat_dual_mode_and_auth_source_defaults.sql")
	require.NoError(t, err)

	sql := string(content)
	require.NotContains(t, sql, "UPDATE settings")
	require.NotContains(t, sql, "SET value = 'false'")
	require.True(t, strings.Contains(sql, "ON CONFLICT (key) DO NOTHING"))
	require.Contains(t, sql, "THEN ''")
}

func TestAuthIdentityReportTypeWideningRunsBeforeLongReportWritersAndStillReconcilesAt121(t *testing.T) {
	preflightContent, err := FS.ReadFile("108a_widen_auth_identity_migration_report_type.sql")
	require.NoError(t, err)

	preflightSQL := string(preflightContent)
	require.Contains(t, preflightSQL, "ALTER TABLE auth_identity_migration_reports")
	require.Contains(t, preflightSQL, "ALTER COLUMN report_type TYPE VARCHAR(80)")

	content, err := FS.ReadFile("109_auth_identity_compat_backfill.sql")
	require.NoError(t, err)

	sql := string(content)
	require.NotContains(t, sql, "ALTER TABLE auth_identity_migration_reports")

	followupContent, err := FS.ReadFile("121_auth_identity_migration_report_type_widen.sql")
	require.NoError(t, err)

	followupSQL := string(followupContent)
	require.Contains(t, followupSQL, "ALTER TABLE auth_identity_migration_reports")
	require.Contains(t, followupSQL, "ALTER COLUMN report_type TYPE VARCHAR(80)")
}

func TestMigration119DefersPaymentIndexRolloutToOnlineFollowup(t *testing.T) {
	content, err := FS.ReadFile("119_enforce_payment_orders_out_trade_no_unique.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "120_enforce_payment_orders_out_trade_no_unique_notx.sql")
	require.Contains(t, sql, "NULL;")
	require.NotContains(t, sql, "CREATE UNIQUE INDEX")
	require.NotContains(t, sql, "DROP INDEX")

	followupContent, err := FS.ReadFile("120_enforce_payment_orders_out_trade_no_unique_notx.sql")
	require.NoError(t, err)

	followupSQL := string(followupContent)
	require.Contains(t, followupSQL, "explicit duplicate out_trade_no precheck")
	require.Contains(t, followupSQL, "stale invalid paymentorder_out_trade_no_unique index")
	require.Contains(t, followupSQL, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS paymentorder_out_trade_no_unique")
	require.NotContains(t, followupSQL, "DROP INDEX CONCURRENTLY IF EXISTS paymentorder_out_trade_no_unique")
	require.Contains(t, followupSQL, "DROP INDEX CONCURRENTLY IF EXISTS paymentorder_out_trade_no")
	require.Contains(t, followupSQL, "WHERE out_trade_no <> ''")

	alignmentContent, err := FS.ReadFile("120a_align_payment_orders_out_trade_no_index_name.sql")
	require.NoError(t, err)

	alignmentSQL := string(alignmentContent)
	require.Contains(t, alignmentSQL, "paymentorder_out_trade_no_unique")
	require.Contains(t, alignmentSQL, "RENAME TO paymentorder_out_trade_no")
}

func TestMigration110SeedsAuthSourceSignupGrantsDisabledByDefault(t *testing.T) {
	content, err := FS.ReadFile("110_pending_auth_and_provider_default_grants.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "('auth_source_default_email_grant_on_signup', 'false')")
	require.Contains(t, sql, "('auth_source_default_linuxdo_grant_on_signup', 'false')")
	require.Contains(t, sql, "('auth_source_default_oidc_grant_on_signup', 'false')")
	require.Contains(t, sql, "('auth_source_default_wechat_grant_on_signup', 'false')")
	require.NotContains(t, sql, "('auth_source_default_email_grant_on_signup', 'true')")
}

func TestMigration122ScrubsPendingOAuthCompletionTokensAtRest(t *testing.T) {
	content, err := FS.ReadFile("122_pending_auth_completion_token_cleanup.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "UPDATE pending_auth_sessions")
	require.Contains(t, sql, "completion_response")
	require.Contains(t, sql, "access_token")
	require.Contains(t, sql, "refresh_token")
	require.Contains(t, sql, "expires_in")
	require.Contains(t, sql, "token_type")
}

func TestMigration123BackfillsLegacyAuthSourceGrantDefaultsSafely(t *testing.T) {
	content, err := FS.ReadFile("123_fix_legacy_auth_source_grant_on_signup_defaults.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "110_pending_auth_and_provider_default_grants.sql")
	require.Contains(t, sql, "schema_migrations")
	require.Contains(t, sql, "updated_at")
	require.Contains(t, sql, "'_grant_on_signup'")
	require.Contains(t, sql, "value = 'false'")
	require.Contains(t, sql, "auth_identity_migration_reports")
}

func TestMigration124BackfillsLegacyOIDCSecurityFlagsSafely(t *testing.T) {
	content, err := FS.ReadFile("124_backfill_legacy_oidc_security_flags.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "oidc_connect_use_pkce")
	require.Contains(t, sql, "oidc_connect_validate_id_token")
	require.Contains(t, sql, "ON CONFLICT (key) DO NOTHING")
	require.Contains(t, sql, "oidc_connect_enabled")
	require.Contains(t, sql, "'false'")
}

func TestMigration134AddsAffiliateLedgerAuditFieldsWithoutJSONCast(t *testing.T) {
	content, err := FS.ReadFile("134_affiliate_ledger_audit_snapshots.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS source_order_id BIGINT")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS balance_after DECIMAL(20,8)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS aff_quota_after DECIMAL(20,8)")
	require.Contains(t, sql, "substring(")
	require.Contains(t, sql, `"rebateAmount"`)
	require.Contains(t, sql, "COUNT(*) OVER (PARTITION BY ra.order_id) AS order_match_count")
	require.Contains(t, sql, "COUNT(*) OVER (PARTITION BY ual.id) AS ledger_match_count")
	require.NotContains(t, sql, "detail::jsonb")
}

func TestMigration135AllowsGitHubAndGoogleAuthProviders(t *testing.T) {
	content, err := FS.ReadFile("135_allow_email_oauth_provider_types.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "users_signup_source_check")
	require.Contains(t, sql, "auth_identities_provider_type_check")
	require.Contains(t, sql, "auth_identity_channels_provider_type_check")
	require.Contains(t, sql, "pending_auth_sessions_provider_type_check")
	require.Contains(t, sql, "'github'")
	require.Contains(t, sql, "'google'")
}

func TestMigration153ProtectsSocialTaskUsageLedgerIdempotency(t *testing.T) {
	content, err := FS.ReadFile("153_social_task_usage_ledger_idempotency.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "WHERE request_id IS NOT NULL")
	require.Contains(t, sql, "api_key_id IS NULL")
	require.Contains(t, sql, "model = 'social-action'")
	require.Contains(t, sql, "ROW_NUMBER() OVER (PARTITION BY request_id ORDER BY id)")
	require.Contains(t, sql, "SET request_id = NULL")
	require.NotContains(t, sql, "CONCURRENTLY")

	indexContent, err := FS.ReadFile("154_social_task_usage_ledger_idempotency_notx.sql")
	require.NoError(t, err)

	indexSQL := string(indexContent)
	require.Contains(t, indexSQL, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_social_task_request_unique")
	require.Contains(t, indexSQL, "ON usage_logs (request_id)")
	require.Contains(t, indexSQL, "WHERE request_id IS NOT NULL")
	require.Contains(t, indexSQL, "api_key_id IS NULL")
	require.Contains(t, indexSQL, "model = 'social-action'")
	require.NotContains(t, indexSQL, "ON usage_logs (request_id, api_key_id)")
}

func TestMigration168ProtectsOneActiveSocialTaskPerAccount(t *testing.T) {
	content, err := FS.ReadFile("168_social_task_active_account_uniqueness.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ROW_NUMBER() OVER")
	require.Contains(t, sql, "PARTITION BY social_account_id")
	require.Contains(t, sql, "WHERE status IN ('pending', 'running')")
	require.Contains(t, sql, "SET status = 'failed'")
	require.Contains(t, sql, "charged_amount = 0")
	require.Contains(t, sql, "charge_status = 'not_charged'")
	require.NotContains(t, sql, "CONCURRENTLY")

	indexContent, err := FS.ReadFile("169_social_task_active_account_uniqueness_notx.sql")
	require.NoError(t, err)

	indexSQL := string(indexContent)
	require.Contains(t, indexSQL, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_social_task_logs_one_active_per_account")
	require.Contains(t, indexSQL, "ON social_task_logs (social_account_id)")
	require.Contains(t, indexSQL, "WHERE status IN ('pending', 'running')")
}

func TestSocialOpsCoreMigrationsConvergeOnCurrentDataModel(t *testing.T) {
	settingsSQL := readMigrationSQL(t, "005_schema_parity.sql")
	require.Contains(t, settingsSQL, "CREATE TABLE IF NOT EXISTS settings")
	require.Contains(t, settingsSQL, "key         VARCHAR(100) NOT NULL UNIQUE")
	require.Contains(t, settingsSQL, "value       TEXT NOT NULL")

	fieldModelSQL := readMigrationSQL(t, "159_social_account_field_model.sql")
	require.Contains(t, fieldModelSQL, "ADD COLUMN IF NOT EXISTS platform_user_id VARCHAR(100)")
	require.Contains(t, fieldModelSQL, "ADD COLUMN IF NOT EXISTS execution_auth TEXT")
	require.Contains(t, fieldModelSQL, "ADD COLUMN IF NOT EXISTS default_proxy_snapshot TEXT")

	identitySQL := readMigrationSQL(t, "160_social_account_identity_model.sql")
	require.Contains(t, identitySQL, "DROP COLUMN IF EXISTS account_id")
	require.Contains(t, identitySQL, "DROP COLUMN IF EXISTS bound_ip")

	dropSourceSQL := readMigrationSQL(t, "155_drop_social_account_source.sql")
	require.Contains(t, dropSourceSQL, "DROP COLUMN IF EXISTS source")

	deleteStateSQL := readMigrationSQL(t, "167_drop_social_account_workbench_deleted_at.sql")
	require.Contains(t, deleteStateSQL, "DROP COLUMN IF EXISTS user_workbench_deleted_at")
	require.Contains(t, deleteStateSQL, "DROP COLUMN IF EXISTS deleted_at")
	require.Regexp(t, `(?is)CREATE UNIQUE INDEX IF NOT EXISTS idx_social_accounts_platform_name_key_unique\s+ON social_accounts\(platform_key,\s*name_key\);`, deleteStateSQL)
	require.NotRegexp(t, `(?is)CREATE UNIQUE INDEX IF NOT EXISTS idx_social_accounts_platform_name_key_unique\s+ON social_accounts\(platform_key,\s*name_key\)\s+WHERE deleted_at IS NULL`, deleteStateSQL)

	payloadSQL := readMigrationSQL(t, "163_social_task_payload_snapshot.sql")
	require.Contains(t, payloadSQL, "ADD COLUMN IF NOT EXISTS payload JSONB NOT NULL DEFAULT '{}'::jsonb")
	require.Contains(t, payloadSQL, "ADD COLUMN IF NOT EXISTS template_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb")

	activeTaskIndexSQL := readMigrationSQL(t, "169_social_task_active_account_uniqueness_notx.sql")
	require.Contains(t, activeTaskIndexSQL, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_social_task_logs_one_active_per_account")
	require.Contains(t, activeTaskIndexSQL, "ON social_task_logs (social_account_id)")
	require.Contains(t, activeTaskIndexSQL, "WHERE status IN ('pending', 'running')")
}

func readMigrationSQL(t *testing.T, name string) string {
	t.Helper()

	content, err := FS.ReadFile(name)
	require.NoError(t, err)
	return string(content)
}

func TestMigration157WidensPaymentOrderMonetaryPrecision(t *testing.T) {
	content, err := FS.ReadFile("157_widen_payment_order_amount_precision.sql")
	require.NoError(t, err)

	sql := string(content)
	for _, column := range []string{"amount", "pay_amount", "refund_amount"} {
		require.Contains(t, sql, "ALTER COLUMN "+column+" TYPE DECIMAL(20,8)")
	}
}

func TestMigration158WidensRedeemCodeLength(t *testing.T) {
	content, err := FS.ReadFile("158_widen_redeem_code_length.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ALTER TABLE redeem_codes")
	require.Contains(t, sql, "ALTER COLUMN code TYPE VARCHAR(128)")
	require.NotContains(t, sql, "DROP")
}
