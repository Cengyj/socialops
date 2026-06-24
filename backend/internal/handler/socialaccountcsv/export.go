package socialaccountcsv

import (
	"encoding/csv"
	"io"

	"github.com/Wei-Shaw/socialops/internal/service"
)

var deliveryExportHeader = []string{
	"platform",
	"username",
	"name",
	"platform_user_id",
	"password",
	"phone",
	"email",
	"email_password",
	"two_factor",
	"backup_code",
	"email_client_id",
	"email_token",
	"registration_ip",
	"auth_cookie",
	"execution_auth",
	"default_proxy_snapshot",
	"account_status",
	"task_status",
	"remark",
	"created_at",
	"updated_at",
}

func WriteDeliveryExport(writer io.Writer, accounts []*service.SocialAccount) error {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(deliveryExportHeader); err != nil {
		return err
	}
	for _, account := range accounts {
		if err := csvWriter.Write(deliveryExportRow(account)); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func deliveryExportRow(account *service.SocialAccount) []string {
	return []string{
		account.Platform,
		account.Username,
		account.Name,
		ptrString(account.PlatformUserID),
		ptrString(account.Password),
		ptrString(account.Phone),
		ptrString(account.Email),
		ptrString(account.EmailPassword),
		ptrString(account.TwoFactor),
		ptrString(account.BackupCode),
		ptrString(account.EmailClientID),
		ptrString(account.EmailToken),
		ptrString(account.RegistrationIP),
		ptrString(account.AuthCookie),
		ptrString(account.ExecutionAuth),
		ptrString(account.DefaultProxySnapshot),
		account.AccountStatus,
		account.TaskStatus,
		ptrString(account.Remark),
		account.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		account.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
