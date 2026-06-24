//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthIdentityHistoricalExternalBackfillMigration(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	migrationPath := filepath.Join("..", "..", "migrations", "115_auth_identity_legacy_external_backfill.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	prepareHistoricalExternalIdentitiesTable(t, tx, ctx)
	truncateAuthIdentityHistoricalFixtureTables(t, tx, ctx)

	var linuxDoUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-linuxdo@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&linuxDoUserID))

	var wechatUnionUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-wechat-union@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&wechatUnionUserID))

	var wechatOpenIDOnlyUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-wechat-openid@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&wechatOpenIDOnlyUserID))

	var syntheticAuthIdentityID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO auth_identities (user_id, provider_type, provider_key, provider_subject, metadata)
VALUES ($1, 'wechat', 'wechat-main', 'openid-synthetic', '{"backfill_source":"synthetic_email"}'::jsonb)
RETURNING id`, wechatOpenIDOnlyUserID).Scan(&syntheticAuthIdentityID))

	var linuxDoHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-user-1', NULL, 'linux-user', 'Linux User', '{"source":"stored"}')
RETURNING id
`, linuxDoUserID).Scan(&linuxDoHistoricalID))

	var wechatUnionHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-union-1', 'union-1', 'wechat-union-user', 'WeChat Union User', '{"channel":"oa","appid":"wx-app-1"}')
RETURNING id
`, wechatUnionUserID).Scan(&wechatUnionHistoricalID))

	var wechatOpenIDHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-only-1', NULL, 'wechat-openid-user', 'WeChat OpenID User', '{"channel":"oa","appid":"wx-app-2"}')
RETURNING id
`, wechatOpenIDOnlyUserID).Scan(&wechatOpenIDHistoricalID))

	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var linuxDoCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identities
WHERE user_id = $1
  AND provider_type = 'linuxdo'
  AND provider_key = 'linuxdo'
  AND provider_subject = 'linuxdo-user-1'
`, linuxDoUserID).Scan(&linuxDoCount))
	require.Equal(t, 1, linuxDoCount)

	var wechatSubject string
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT provider_subject
FROM auth_identities
WHERE user_id = $1
  AND provider_type = 'wechat'
  AND provider_key = 'wechat-main'
  AND provider_subject = 'union-1'
`, wechatUnionUserID).Scan(&wechatSubject))
	require.Equal(t, "union-1", wechatSubject)

	var wechatChannelCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_channels channel
JOIN auth_identities ai ON ai.id = channel.identity_id
WHERE ai.user_id = $1
  AND channel.provider_type = 'wechat'
  AND channel.provider_key = 'wechat-main'
  AND channel.channel = 'oa'
  AND channel.channel_app_id = 'wx-app-1'
  AND channel.channel_subject = 'openid-union-1'
`, wechatUnionUserID).Scan(&wechatChannelCount))
	require.Equal(t, 1, wechatChannelCount)

	var historicalOpenIDOnlyReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'wechat_openid_only_requires_remediation'
  AND report_key = $1
`, "legacy_external_identity:"+strconv.FormatInt(wechatOpenIDHistoricalID, 10)).Scan(&historicalOpenIDOnlyReportCount))
	require.Equal(t, 1, historicalOpenIDOnlyReportCount)

	var syntheticReviewCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'wechat_openid_only_requires_remediation'
  AND report_key = $1
`, "synthetic_auth_identity:"+strconv.FormatInt(syntheticAuthIdentityID, 10)).Scan(&syntheticReviewCount))
	require.Equal(t, 1, syntheticReviewCount)

	var unionHistoricalReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'wechat_openid_only_requires_remediation'
  AND report_key = $1
`, "legacy_external_identity:"+strconv.FormatInt(wechatUnionHistoricalID, 10)).Scan(&unionHistoricalReportCount))
	require.Zero(t, unionHistoricalReportCount)
	require.NotZero(t, linuxDoHistoricalID)
}

func TestAuthIdentityHistoricalExternalBackfillMigration_IsSafeWhenHistoricalTableMissing(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	migrationPath := filepath.Join("..", "..", "migrations", "115_auth_identity_legacy_external_backfill.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	var beforeCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
`).Scan(&beforeCount))

	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var afterCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
	`).Scan(&afterCount))
	require.Equal(t, beforeCount, afterCount)
}

func TestAuthIdentityHistoricalExternalMigrations_ChainHandlesMalformedAndNonObjectMetadata(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	migration115Path := filepath.Join("..", "..", "migrations", "115_auth_identity_legacy_external_backfill.sql")
	migration115SQL, err := os.ReadFile(migration115Path)
	require.NoError(t, err)

	migration116Path := filepath.Join("..", "..", "migrations", "116_auth_identity_legacy_external_safety_reports.sql")
	migration116SQL, err := os.ReadFile(migration116Path)
	require.NoError(t, err)

	prepareHistoricalExternalIdentitiesTable(t, tx, ctx)
	truncateAuthIdentityHistoricalFixtureTables(t, tx, ctx)

	var linuxDoMalformedUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-linuxdo-malformed@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&linuxDoMalformedUserID))

	var linuxDoArrayUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-linuxdo-array@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&linuxDoArrayUserID))

	var wechatUnionArrayUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-wechat-array@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&wechatUnionArrayUserID))

	var wechatOpenIDArrayUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-wechat-openid-array@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&wechatOpenIDArrayUserID))

	var linuxDoMalformedHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-malformed', NULL, 'historical-linuxdo-malformed', 'Historical LinuxDo Malformed', '{invalid')
RETURNING id
`, linuxDoMalformedUserID).Scan(&linuxDoMalformedHistoricalID))

	var linuxDoArrayHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-array', NULL, 'historical-linuxdo-array', 'Historical LinuxDo Array', '["historical-linuxdo-array"]')
RETURNING id
`, linuxDoArrayUserID).Scan(&linuxDoArrayHistoricalID))

	var wechatUnionArrayHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-array', 'union-array', 'historical-wechat-array', 'Historical WeChat Array', '["historical-wechat-array"]')
RETURNING id
`, wechatUnionArrayUserID).Scan(&wechatUnionArrayHistoricalID))

	var wechatOpenIDArrayHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-array-only', NULL, 'historical-wechat-array-only', 'Historical WeChat Array Only', '["historical-wechat-openid-array"]')
RETURNING id
`, wechatOpenIDArrayUserID).Scan(&wechatOpenIDArrayHistoricalID))

	_, err = tx.ExecContext(ctx, string(migration115SQL))
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, string(migration116SQL))
	require.NoError(t, err)

	var linuxDoMalformedMetadataType string
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT jsonb_typeof(metadata)
FROM auth_identities
WHERE user_id = $1
  AND provider_type = 'linuxdo'
  AND provider_key = 'linuxdo'
  AND provider_subject = 'linuxdo-malformed'
`, linuxDoMalformedUserID).Scan(&linuxDoMalformedMetadataType))
	require.Equal(t, "object", linuxDoMalformedMetadataType)

	var linuxDoArrayMetadataType string
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT jsonb_typeof(metadata)
FROM auth_identities
WHERE user_id = $1
  AND provider_type = 'linuxdo'
  AND provider_key = 'linuxdo'
  AND provider_subject = 'linuxdo-array'
`, linuxDoArrayUserID).Scan(&linuxDoArrayMetadataType))
	require.Equal(t, "object", linuxDoArrayMetadataType)

	var wechatUnionArrayMetadataType string
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT jsonb_typeof(metadata)
FROM auth_identities
WHERE user_id = $1
  AND provider_type = 'wechat'
  AND provider_key = 'wechat-main'
  AND provider_subject = 'union-array'
`, wechatUnionArrayUserID).Scan(&wechatUnionArrayMetadataType))
	require.Equal(t, "object", wechatUnionArrayMetadataType)

	var invalidJSONReportDetailsType string
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT jsonb_typeof(details)
FROM auth_identity_migration_reports
WHERE report_type = 'legacy_external_identity_invalid_metadata_json'
  AND report_key = $1
`, "legacy_external_identity:"+strconv.FormatInt(linuxDoMalformedHistoricalID, 10)).Scan(&invalidJSONReportDetailsType))
	require.Equal(t, "object", invalidJSONReportDetailsType)

	var openIDOnlyReportDetailsType string
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT jsonb_typeof(details)
FROM auth_identity_migration_reports
WHERE report_type = 'wechat_openid_only_requires_remediation'
  AND report_key = $1
`, "legacy_external_identity:"+strconv.FormatInt(wechatOpenIDArrayHistoricalID, 10)).Scan(&openIDOnlyReportDetailsType))
	require.Equal(t, "object", openIDOnlyReportDetailsType)

	var preservedArrayMetadataCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identities
WHERE id IN (
	SELECT id
	FROM auth_identities
	WHERE (user_id = $1 AND provider_subject = 'linuxdo-array')
	   OR (user_id = $2 AND provider_subject = 'union-array')
)
  AND metadata ? '_legacy_metadata_raw_json'
`, linuxDoArrayUserID, wechatUnionArrayUserID).Scan(&preservedArrayMetadataCount))
	require.Equal(t, 2, preservedArrayMetadataCount)

	require.NotZero(t, linuxDoArrayHistoricalID)
	require.NotZero(t, wechatUnionArrayHistoricalID)
}

func TestAuthIdentityHistoricalExternalSafetyMigration_ReportsConflictsAndDowngradesInvalidJSON(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	migrationPath := filepath.Join("..", "..", "migrations", "116_auth_identity_legacy_external_safety_reports.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	prepareHistoricalExternalIdentitiesTable(t, tx, ctx)
	truncateAuthIdentityHistoricalFixtureTables(t, tx, ctx)

	userIDs := make([]int64, 0, 8)
	for _, email := range []string{
		"linuxdo-conflict-historical@example.com",
		"linuxdo-conflict-owner@example.com",
		"wechat-conflict-historical@example.com",
		"wechat-conflict-owner@example.com",
		"wechat-channel-historical@example.com",
		"wechat-channel-owner@example.com",
		"linuxdo-invalid-json@example.com",
		"wechat-openid-invalid-json@example.com",
	} {
		var userID int64
		require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ($1, 'hash', 'user', 'active', 0, 1)
RETURNING id`, email).Scan(&userID))
		userIDs = append(userIDs, userID)
	}

	linuxdoConflictHistoricalUserID := userIDs[0]
	linuxdoConflictOwnerUserID := userIDs[1]
	wechatConflictHistoricalUserID := userIDs[2]
	wechatConflictOwnerUserID := userIDs[3]
	wechatChannelHistoricalUserID := userIDs[4]
	wechatChannelOwnerUserID := userIDs[5]
	linuxdoInvalidJSONUserID := userIDs[6]
	wechatInvalidOpenIDUserID := userIDs[7]

	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO auth_identities (user_id, provider_type, provider_key, provider_subject, metadata)
VALUES ($1, 'linuxdo', 'linuxdo', 'linuxdo-conflict', '{}'::jsonb)
RETURNING id`, linuxdoConflictOwnerUserID).Scan(new(int64)))

	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO auth_identities (user_id, provider_type, provider_key, provider_subject, metadata)
VALUES ($1, 'wechat', 'wechat-main', 'union-conflict', '{}'::jsonb)
RETURNING id`, wechatConflictOwnerUserID).Scan(new(int64)))

	var wechatChannelOwnerIdentityID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO auth_identities (user_id, provider_type, provider_key, provider_subject, metadata)
VALUES ($1, 'wechat', 'wechat-main', 'union-channel-owner', '{}'::jsonb)
RETURNING id`, wechatChannelOwnerUserID).Scan(&wechatChannelOwnerIdentityID))

	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO auth_identity_channels (
	identity_id,
	provider_type,
	provider_key,
	channel,
	channel_app_id,
	channel_subject,
	metadata
)
VALUES ($1, 'wechat', 'wechat-main', 'oa', 'wx-app-conflict', 'openid-channel-conflict', '{}'::jsonb)
RETURNING id`, wechatChannelOwnerIdentityID).Scan(new(int64)))

	var linuxdoConflictHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-conflict', NULL, 'historical-linuxdo', 'Historical LinuxDo Conflict', '{"source":"stored"}')
RETURNING id
`, linuxdoConflictHistoricalUserID).Scan(&linuxdoConflictHistoricalID))

	var wechatConflictHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-union-conflict', 'union-conflict', 'historical-wechat', 'Historical WeChat Conflict', '{"channel":"oa","appid":"wx-app-conflict-canon"}')
RETURNING id
`, wechatConflictHistoricalUserID).Scan(&wechatConflictHistoricalID))

	var wechatChannelConflictHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-channel-conflict', 'union-channel-historical', 'historical-wechat-channel', 'Historical WeChat Channel Conflict', '{"channel":"oa","appid":"wx-app-conflict"}')
RETURNING id
`, wechatChannelHistoricalUserID).Scan(&wechatChannelConflictHistoricalID))

	var linuxdoInvalidJSONHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-invalid-json', NULL, 'historical-linuxdo-invalid', 'Historical LinuxDo Invalid JSON', '{invalid')
RETURNING id
`, linuxdoInvalidJSONUserID).Scan(&linuxdoInvalidJSONHistoricalID))

	var wechatInvalidOpenIDHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-invalid-json-only', NULL, 'historical-wechat-invalid', 'Historical WeChat Invalid JSON', '{still-invalid')
RETURNING id
`, wechatInvalidOpenIDUserID).Scan(&wechatInvalidOpenIDHistoricalID))

	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var linuxdoConflictReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'legacy_external_identity_conflict'
  AND report_key = $1
`, "legacy_external_identity:"+strconv.FormatInt(linuxdoConflictHistoricalID, 10)).Scan(&linuxdoConflictReportCount))
	require.Equal(t, 1, linuxdoConflictReportCount)

	var wechatConflictReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'legacy_external_identity_conflict'
  AND report_key = $1
`, "legacy_external_identity:"+strconv.FormatInt(wechatConflictHistoricalID, 10)).Scan(&wechatConflictReportCount))
	require.Equal(t, 1, wechatConflictReportCount)

	var channelConflictReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'legacy_external_channel_conflict'
  AND report_key = $1
`, "legacy_external_identity:"+strconv.FormatInt(wechatChannelConflictHistoricalID, 10)).Scan(&channelConflictReportCount))
	require.Equal(t, 1, channelConflictReportCount)

	var invalidJSONReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'legacy_external_identity_invalid_metadata_json'
  AND report_key IN ($1, $2)
`, "legacy_external_identity:"+strconv.FormatInt(linuxdoInvalidJSONHistoricalID, 10), "legacy_external_identity:"+strconv.FormatInt(wechatInvalidOpenIDHistoricalID, 10)).Scan(&invalidJSONReportCount))
	require.Equal(t, 2, invalidJSONReportCount)

	var linuxdoInvalidIdentityCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identities
WHERE user_id = $1
  AND provider_type = 'linuxdo'
  AND provider_key = 'linuxdo'
  AND provider_subject = 'linuxdo-invalid-json'
`, linuxdoInvalidJSONUserID).Scan(&linuxdoInvalidIdentityCount))
	require.Equal(t, 1, linuxdoInvalidIdentityCount)

	var wechatOpenIDOnlyReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'wechat_openid_only_requires_remediation'
  AND report_key = $1
`, "legacy_external_identity:"+strconv.FormatInt(wechatInvalidOpenIDHistoricalID, 10)).Scan(&wechatOpenIDOnlyReportCount))
	require.Equal(t, 1, wechatOpenIDOnlyReportCount)
}

func TestAuthIdentityHistoricalExternalSafetyMigration_IsSafeWhenHistoricalTableMissing(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	migrationPath := filepath.Join("..", "..", "migrations", "116_auth_identity_legacy_external_safety_reports.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	var beforeCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
`).Scan(&beforeCount))

	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var afterCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
	`).Scan(&afterCount))
	require.Equal(t, beforeCount, afterCount)
}

func TestAuthIdentityHistoricalExternalBackfillMigration_SkipsAmbiguousCanonicalSubjects(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	migrationPath := filepath.Join("..", "..", "migrations", "115_auth_identity_legacy_external_backfill.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	prepareHistoricalExternalIdentitiesTable(t, tx, ctx)
	truncateAuthIdentityHistoricalFixtureTables(t, tx, ctx)

	var linuxDoFirstUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-linuxdo-ambiguous-a@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&linuxDoFirstUserID))

	var linuxDoSecondUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-linuxdo-ambiguous-b@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&linuxDoSecondUserID))

	var wechatFirstUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-wechat-ambiguous-a@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&wechatFirstUserID))

	var wechatSecondUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-wechat-ambiguous-b@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&wechatSecondUserID))

	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-ambiguous-subject', NULL, 'historical-linuxdo-ambiguous-a', 'Historical LinuxDo Ambiguous A', '{"source":"stored"}')
RETURNING id
`, linuxDoFirstUserID).Scan(new(int64)))

	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-ambiguous-subject', NULL, 'historical-linuxdo-ambiguous-b', 'Historical LinuxDo Ambiguous B', '{"source":"stored"}')
RETURNING id
`, linuxDoSecondUserID).Scan(new(int64)))

	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-ambiguous-a', 'union-ambiguous-subject', 'historical-wechat-ambiguous-a', 'Historical WeChat Ambiguous A', '{"channel":"oa","appid":"wx-ambiguous-a"}')
RETURNING id
`, wechatFirstUserID).Scan(new(int64)))

	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-ambiguous-b', 'union-ambiguous-subject', 'historical-wechat-ambiguous-b', 'Historical WeChat Ambiguous B', '{"channel":"oa","appid":"wx-ambiguous-b"}')
RETURNING id
`, wechatSecondUserID).Scan(new(int64)))

	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var linuxDoIdentityCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identities
WHERE provider_type = 'linuxdo'
  AND provider_key = 'linuxdo'
  AND provider_subject = 'linuxdo-ambiguous-subject'
`).Scan(&linuxDoIdentityCount))
	require.Zero(t, linuxDoIdentityCount)

	var wechatIdentityCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identities
WHERE provider_type = 'wechat'
  AND provider_key = 'wechat-main'
  AND provider_subject = 'union-ambiguous-subject'
`).Scan(&wechatIdentityCount))
	require.Zero(t, wechatIdentityCount)

	var wechatChannelCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_channels
WHERE provider_type = 'wechat'
  AND provider_key = 'wechat-main'
  AND channel = 'oa'
  AND channel_app_id IN ('wx-ambiguous-a', 'wx-ambiguous-b')
`).Scan(&wechatChannelCount))
	require.Zero(t, wechatChannelCount)
}

func TestAuthIdentityHistoricalExternalMigrations_ReportAmbiguousCanonicalSubjectsWithoutWinnerAttribution(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	migration115Path := filepath.Join("..", "..", "migrations", "115_auth_identity_legacy_external_backfill.sql")
	migration115SQL, err := os.ReadFile(migration115Path)
	require.NoError(t, err)

	migration116Path := filepath.Join("..", "..", "migrations", "116_auth_identity_legacy_external_safety_reports.sql")
	migration116SQL, err := os.ReadFile(migration116Path)
	require.NoError(t, err)

	prepareHistoricalExternalIdentitiesTable(t, tx, ctx)
	truncateAuthIdentityHistoricalFixtureTables(t, tx, ctx)

	var linuxDoFirstUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-linuxdo-conflict-a@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&linuxDoFirstUserID))

	var linuxDoSecondUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-linuxdo-conflict-b@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&linuxDoSecondUserID))

	var wechatFirstUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-wechat-conflict-a@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&wechatFirstUserID))

	var wechatSecondUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-wechat-conflict-b@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&wechatSecondUserID))

	var linuxDoFirstHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-conflict-subject', NULL, 'historical-linuxdo-conflict-a', 'Historical LinuxDo Conflict A', '{"source":"stored"}')
RETURNING id
`, linuxDoFirstUserID).Scan(&linuxDoFirstHistoricalID))

	var linuxDoSecondHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-conflict-subject', NULL, 'historical-linuxdo-conflict-b', 'Historical LinuxDo Conflict B', '{"source":"stored"}')
RETURNING id
`, linuxDoSecondUserID).Scan(&linuxDoSecondHistoricalID))

	var wechatFirstHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-conflict-a', 'union-conflict-subject', 'historical-wechat-conflict-a', 'Historical WeChat Conflict A', '{"channel":"oa","appid":"wx-conflict-a"}')
RETURNING id
`, wechatFirstUserID).Scan(&wechatFirstHistoricalID))

	var wechatSecondHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'wechat', 'openid-conflict-b', 'union-conflict-subject', 'historical-wechat-conflict-b', 'Historical WeChat Conflict B', '{"channel":"oa","appid":"wx-conflict-b"}')
RETURNING id
`, wechatSecondUserID).Scan(&wechatSecondHistoricalID))

	_, err = tx.ExecContext(ctx, string(migration115SQL))
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, string(migration116SQL))
	require.NoError(t, err)

	var identityCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identities
WHERE (provider_type = 'linuxdo' AND provider_key = 'linuxdo' AND provider_subject = 'linuxdo-conflict-subject')
   OR (provider_type = 'wechat' AND provider_key = 'wechat-main' AND provider_subject = 'union-conflict-subject')
`).Scan(&identityCount))
	require.Zero(t, identityCount)

	var conflictReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'legacy_external_identity_conflict'
  AND report_key IN ($1, $2, $3, $4)
`, "legacy_external_identity:"+strconv.FormatInt(linuxDoFirstHistoricalID, 10), "legacy_external_identity:"+strconv.FormatInt(linuxDoSecondHistoricalID, 10), "legacy_external_identity:"+strconv.FormatInt(wechatFirstHistoricalID, 10), "legacy_external_identity:"+strconv.FormatInt(wechatSecondHistoricalID, 10)).Scan(&conflictReportCount))
	require.Equal(t, 4, conflictReportCount)

	var winnerAttributedReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'legacy_external_identity_conflict'
  AND report_key IN ($1, $2, $3, $4)
  AND details ->> 'existing_identity_id' IS NOT NULL
`, "legacy_external_identity:"+strconv.FormatInt(linuxDoFirstHistoricalID, 10), "legacy_external_identity:"+strconv.FormatInt(linuxDoSecondHistoricalID, 10), "legacy_external_identity:"+strconv.FormatInt(wechatFirstHistoricalID, 10), "legacy_external_identity:"+strconv.FormatInt(wechatSecondHistoricalID, 10)).Scan(&winnerAttributedReportCount))
	require.Zero(t, winnerAttributedReportCount)
}

func TestAuthIdentityMigrationReportTypeWideningPreflightKeeps109And116SafeBefore121(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	migration108aPath := filepath.Join("..", "..", "migrations", "108a_widen_auth_identity_migration_report_type.sql")
	migration108aSQL, err := os.ReadFile(migration108aPath)
	require.NoError(t, err)

	migration109Path := filepath.Join("..", "..", "migrations", "109_auth_identity_compat_backfill.sql")
	migration109SQL, err := os.ReadFile(migration109Path)
	require.NoError(t, err)

	migration116Path := filepath.Join("..", "..", "migrations", "116_auth_identity_legacy_external_safety_reports.sql")
	migration116SQL, err := os.ReadFile(migration116Path)
	require.NoError(t, err)

	prepareHistoricalExternalIdentitiesTable(t, tx, ctx)
	truncateAuthIdentityHistoricalFixtureTables(t, tx, ctx)

	_, err = tx.ExecContext(ctx, `
ALTER TABLE auth_identity_migration_reports
ALTER COLUMN report_type TYPE VARCHAR(40);
`)
	require.NoError(t, err)

	var oidcSyntheticUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('oidc-before-121@oidc-connect.invalid', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&oidcSyntheticUserID))

	var linuxdoHistoricalUserID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('historical-linuxdo-before-121@example.com', 'hash', 'user', 'active', 0, 1)
RETURNING id`).Scan(&linuxdoHistoricalUserID))

	var invalidMetadataHistoricalID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_external_identities (
	user_id,
	provider,
	provider_user_id,
	provider_union_id,
	provider_username,
	display_name,
	metadata
) VALUES ($1, 'linuxdo', 'linuxdo-before-121', NULL, 'historical-linuxdo-before-121', 'Historical LinuxDo Before 121', '{invalid')
RETURNING id
`, linuxdoHistoricalUserID).Scan(&invalidMetadataHistoricalID))

	_, err = tx.ExecContext(ctx, string(migration108aSQL))
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, string(migration109SQL))
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, string(migration116SQL))
	require.NoError(t, err)

	var reportTypeWidth int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT character_maximum_length
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'auth_identity_migration_reports'
  AND column_name = 'report_type'
`).Scan(&reportTypeWidth))
	require.Equal(t, 80, reportTypeWidth)

	var oidcSyntheticRecoveryReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'oidc_synthetic_email_requires_manual_recovery'
  AND report_key = $1
`, strconv.FormatInt(oidcSyntheticUserID, 10)).Scan(&oidcSyntheticRecoveryReportCount))
	require.Equal(t, 1, oidcSyntheticRecoveryReportCount)

	var invalidMetadataReportCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM auth_identity_migration_reports
WHERE report_type = 'legacy_external_identity_invalid_metadata_json'
  AND report_key = $1
`, "legacy_external_identity:"+strconv.FormatInt(invalidMetadataHistoricalID, 10)).Scan(&invalidMetadataReportCount))
	require.Equal(t, 1, invalidMetadataReportCount)
}

func prepareHistoricalExternalIdentitiesTable(t *testing.T, tx *sql.Tx, ctx context.Context) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS user_external_identities (
	id BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL,
	provider TEXT NOT NULL,
	provider_user_id TEXT NOT NULL,
	provider_union_id TEXT NULL,
	provider_username TEXT NOT NULL DEFAULT '',
	display_name TEXT NOT NULL DEFAULT '',
	profile_url TEXT NOT NULL DEFAULT '',
	avatar_url TEXT NOT NULL DEFAULT '',
	metadata TEXT NOT NULL DEFAULT '{}',
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`)
	require.NoError(t, err)
}

func truncateAuthIdentityHistoricalFixtureTables(t *testing.T, tx *sql.Tx, ctx context.Context) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
TRUNCATE TABLE
	auth_identity_channels,
	identity_adoption_decisions,
	pending_auth_sessions,
	auth_identities,
	auth_identity_migration_reports,
	user_provider_default_grants,
	user_avatars,
	user_external_identities,
	users
RESTART IDENTITY CASCADE;
`)
	require.NoError(t, err)
}
