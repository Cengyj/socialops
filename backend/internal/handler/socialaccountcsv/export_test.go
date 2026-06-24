package socialaccountcsv

import (
	"bytes"
	"encoding/csv"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/stretchr/testify/require"
)

func TestWriteDeliveryExportPreservesDeliveryFields(t *testing.T) {
	createdAt := time.Date(2026, 6, 10, 1, 2, 3, 0, time.UTC)
	updatedAt := time.Date(2026, 6, 10, 4, 5, 6, 0, time.UTC)
	account := &service.SocialAccount{
		Platform:             "x_twitter",
		Username:             "export_user",
		Name:                 "@export_user",
		PlatformUserID:       stringPtr("platform-123"),
		Password:             stringPtr("account-secret"),
		Phone:                stringPtr("+15550001111"),
		Email:                stringPtr("export@example.com"),
		EmailPassword:        stringPtr("mail-secret"),
		TwoFactor:            stringPtr("totp-secret"),
		BackupCode:           stringPtr("backup-code"),
		EmailClientID:        stringPtr("client-id"),
		EmailToken:           stringPtr("mail-token"),
		RegistrationIP:       stringPtr("198.51.100.10"),
		AuthCookie:           stringPtr("ct0=export; auth_token=export"),
		ExecutionAuth:        stringPtr("encrypted-execution-auth-ciphertext"),
		DefaultProxySnapshot: stringPtr(`{"id":301,"endpoint":"http://proxy.example:8080"}`),
		AccountStatus:        service.SocialAccountStatusAvailable,
		TaskStatus:           service.SocialTaskStatusStored,
		Remark:               stringPtr("delivery note"),
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}

	var buf bytes.Buffer
	require.NoError(t, WriteDeliveryExport(&buf, []*service.SocialAccount{account}))

	records, err := csv.NewReader(bytes.NewReader(buf.Bytes())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, deliveryExportHeader, records[0])
	exported := make(map[string]string, len(records[0]))
	for index, header := range records[0] {
		exported[header] = records[1][index]
	}
	require.Equal(t, "x_twitter", exported["platform"])
	require.Equal(t, "@export_user", exported["name"])
	require.Equal(t, "account-secret", exported["password"])
	require.Equal(t, "mail-secret", exported["email_password"])
	require.Equal(t, "totp-secret", exported["two_factor"])
	require.Equal(t, "backup-code", exported["backup_code"])
	require.Equal(t, "client-id", exported["email_client_id"])
	require.Equal(t, "mail-token", exported["email_token"])
	require.Equal(t, "ct0=export; auth_token=export", exported["auth_cookie"])
	require.Equal(t, "encrypted-execution-auth-ciphertext", exported["execution_auth"])
	require.Equal(t, `{"id":301,"endpoint":"http://proxy.example:8080"}`, exported["default_proxy_snapshot"])
	require.Equal(t, "delivery note", exported["remark"])
	require.Equal(t, "2026-06-10T01:02:03Z", exported["created_at"])
	require.Equal(t, "2026-06-10T04:05:06Z", exported["updated_at"])
}

func stringPtr(value string) *string {
	return &value
}
