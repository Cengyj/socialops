package service

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

const (
	emailSendTimeout = 30 * time.Second
	defaultSiteName  = "SocialOps"
) // BalanceNotifyService handles balance threshold notifications.
type BalanceNotifyService struct {
	emailService             *EmailService
	settingRepo              SettingRepository
	notificationEmailService *NotificationEmailService
}

// NewBalanceNotifyService creates a new BalanceNotifyService.
func NewBalanceNotifyService(emailService *EmailService, settingRepo SettingRepository) *BalanceNotifyService {
	return &BalanceNotifyService{
		emailService: emailService,
		settingRepo:  settingRepo,
	}
}

func (s *BalanceNotifyService) SetNotificationEmailService(notificationEmailService *NotificationEmailService) {
	s.notificationEmailService = notificationEmailService
}

// resolveBalanceThreshold returns the effective balance threshold.
// For percentage type, it computes threshold = totalRecharged * percentage / 100.
func resolveBalanceThreshold(threshold float64, thresholdType string, totalRecharged float64) float64 {
	if thresholdType == thresholdTypePercentage && totalRecharged > 0 {
		return totalRecharged * threshold / 100
	}
	return threshold
}

// CheckBalanceAfterDeduction checks if balance crossed below threshold after deduction.
// Notification is sent only on first crossing: oldBalance >= threshold && newBalance < threshold.
func (s *BalanceNotifyService) CheckBalanceAfterDeduction(ctx context.Context, user *User, oldBalance, cost float64) {
	if !s.canNotifyBalance(user) {
		return
	}
	effectiveThreshold, rechargeURL, ok := s.resolveUserEffectiveThreshold(ctx, user)
	if !ok {
		return
	}
	newBalance := oldBalance - cost
	if !crossedDownward(oldBalance, newBalance, effectiveThreshold) {
		return
	}
	s.dispatchBalanceLowEmail(ctx, user, newBalance, effectiveThreshold, rechargeURL)
}

// canNotifyBalance checks nil guards and user-level toggle.
func (s *BalanceNotifyService) canNotifyBalance(user *User) bool {
	if user == nil || s.emailService == nil || s.settingRepo == nil {
		return false
	}
	return user.BalanceNotifyEnabled
}

// resolveUserEffectiveThreshold reads global + user config, returns the effective threshold.
// Returns ok=false when notifications should be skipped.
func (s *BalanceNotifyService) resolveUserEffectiveThreshold(ctx context.Context, user *User) (effectiveThreshold float64, rechargeURL string, ok bool) {
	globalEnabled, globalThreshold, rechargeURL := s.getBalanceNotifyConfig(ctx)
	if !globalEnabled {
		return 0, "", false
	}
	threshold := globalThreshold
	if user.BalanceNotifyThreshold != nil {
		threshold = *user.BalanceNotifyThreshold
	}
	if threshold <= 0 {
		return 0, "", false
	}
	effectiveThreshold = resolveBalanceThreshold(threshold, user.BalanceNotifyThresholdType, user.TotalRecharged)
	if effectiveThreshold <= 0 {
		return 0, "", false
	}
	return effectiveThreshold, rechargeURL, true
}

// crossedDownward returns true when oldV was at-or-above threshold but newV dropped below it.
func crossedDownward(oldV, newV, threshold float64) bool {
	return oldV >= threshold && newV < threshold
}

// dispatchBalanceLowEmail collects recipients and sends the alert in a goroutine.
func (s *BalanceNotifyService) dispatchBalanceLowEmail(ctx context.Context, user *User, newBalance, threshold float64, rechargeURL string) {
	siteName := s.getSiteName(ctx)
	recipients := s.collectBalanceNotifyRecipients(user)
	slog.Info("CheckBalanceAfterDeduction: sending notification",
		"user_id", user.ID, "recipients", recipients, "new_balance", newBalance, "threshold", threshold)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in balance notification", "recover", r)
			}
		}()
		s.sendBalanceLowEmails(recipients, user.ID, user.Username, user.Email, newBalance, threshold, siteName, rechargeURL)
	}()
}

// filterVerifiedEmails returns deduplicated, non-disabled, verified emails.
func filterVerifiedEmails(entries []NotifyEmailEntry) []string {
	var recipients []string
	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.Disabled || !entry.Verified {
			continue
		}
		email := strings.TrimSpace(entry.Email)
		if email == "" {
			continue
		}
		lower := strings.ToLower(email)
		if seen[lower] {
			continue
		}
		seen[lower] = true
		recipients = append(recipients, email)
	}
	return recipients
}

// collectBalanceNotifyRecipients returns verified, non-disabled email recipients.
// Only emails with verified=true and disabled=false are included.
func (s *BalanceNotifyService) collectBalanceNotifyRecipients(user *User) []string {
	return filterVerifiedEmails(user.BalanceNotifyExtraEmails)
}

// sendEmails sends an email to all recipients with shared timeout and error logging.
func (s *BalanceNotifyService) sendEmails(recipients []string, subject, body string, logAttrs ...any) {
	if len(recipients) == 0 {
		slog.Warn("sendEmails: no recipients", "subject", subject)
		return
	}
	for _, to := range recipients {
		ctx, cancel := context.WithTimeout(context.Background(), emailSendTimeout)
		if err := s.emailService.SendEmail(ctx, to, subject, body); err != nil {
			attrs := append([]any{"to", to, "error", err}, logAttrs...)
			slog.Error("failed to send notification", attrs...)
		} else {
			slog.Info("notification email sent successfully", "to", to, "subject", subject)
		}
		cancel()
	}
}

// sendBalanceLowEmails sends balance low notification to all recipients.
func (s *BalanceNotifyService) sendBalanceLowEmails(recipients []string, userID int64, userName, userEmail string, balance, threshold float64, siteName, rechargeURL string) {
	displayName := userName
	if displayName == "" {
		displayName = userEmail
	}
	if s.notificationEmailService != nil {
		fallbackRecipients := make([]string, 0, len(recipients))
		for _, to := range recipients {
			ctx, cancel := context.WithTimeout(context.Background(), emailSendTimeout)
			err := s.notificationEmailService.Send(ctx, NotificationEmailSendInput{
				Event:          NotificationEmailEventBalanceLow,
				RecipientEmail: to,
				RecipientName:  displayName,
				UserID:         userID,
				SourceType:     "balance_low",
				SourceID:       firstNonEmpty(strconv.FormatInt(userID, 10), userEmail),
				ReminderKey:    time.Now().UTC().Format("2006-01-02"),
				Variables: map[string]string{
					"current_balance": fmt.Sprintf("%.2f", balance),
					"threshold":       fmt.Sprintf("%.2f", threshold),
					"recharge_url":    rechargeURL,
				},
			})
			cancel()
			if err != nil {
				if shouldFallbackNotificationEmail(err) {
					slog.Warn("template balance low notification failed; falling back to built-in body", "to", to, "err", err.Error())
					fallbackRecipients = append(fallbackRecipients, to)
				} else {
					slog.Warn("template balance low notification delivery failed; not sending fallback to avoid duplicates", "to", to, "err", err.Error())
				}
			}
		}
		if len(fallbackRecipients) == 0 {
			return
		}
		recipients = fallbackRecipients
	}
	subject := fmt.Sprintf("[%s] 余额不足提醒 / Balance Low Alert", sanitizeEmailHeader(siteName))
	body := s.buildBalanceLowEmailBody(html.EscapeString(displayName), balance, threshold, html.EscapeString(siteName), rechargeURL)
	s.sendEmails(recipients, subject, body, "user_email", userEmail, "balance", balance)
}

// sanitizeEmailHeader removes CR/LF characters to prevent SMTP header injection.
func sanitizeEmailHeader(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// balanceLowEmailTemplate is the HTML template for balance low notifications.
// Format args: siteName, userName, userName, balance, threshold, threshold.
// The recharge button is appended dynamically when rechargeURL is set.
const balanceLowEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #f59e0b 0%%, #d97706 100%%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 40px 30px; text-align: center; }
        .balance { font-size: 36px; font-weight: bold; color: #dc2626; margin: 20px 0; }
        .info { color: #666; font-size: 14px; line-height: 1.6; margin-top: 20px; }
        .recharge-btn { display: inline-block; margin-top: 24px; padding: 12px 32px; background: linear-gradient(135deg, #f59e0b 0%%, #d97706 100%%); color: #fff; text-decoration: none; border-radius: 6px; font-size: 16px; font-weight: bold; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header"><h1>%s</h1></div>
        <div class="content">
            <p style="font-size: 18px; color: #333;">%s，您的余额不足</p>
            <p style="color: #666;">Dear %s, your balance is running low</p>
            <div class="balance">$%.2f</div>
            <div class="info">
                <p>您的账户余额已低于提醒阈值 <strong>$%.2f</strong>。</p>
                <p>Your account balance has fallen below the alert threshold of <strong>$%.2f</strong>.</p>
                <p>请及时充值以免服务中断。</p>
                <p>Please top up to avoid service interruption.</p>
            </div>
            %s
        </div>
        <div class="footer"><p>此邮件由系统自动发送，请勿回复。</p></div>
    </div>
</body>
</html>`

// buildBalanceLowEmailBody builds HTML email for balance low notification.
func (s *BalanceNotifyService) buildBalanceLowEmailBody(userName string, balance, threshold float64, siteName, rechargeURL string) string {
	rechargeBlock := ""
	if rechargeURL != "" {
		rechargeBlock = fmt.Sprintf(`<a href="%s" class="recharge-btn">立即充值 / Top Up Now</a>`, html.EscapeString(rechargeURL))
	}
	return fmt.Sprintf(balanceLowEmailTemplate, siteName, userName, userName, balance, threshold, threshold, rechargeBlock)
}

// getBalanceNotifyConfig reads global balance notification settings.
func (s *BalanceNotifyService) getBalanceNotifyConfig(ctx context.Context) (enabled bool, threshold float64, rechargeURL string) {
	if s.settingRepo == nil {
		return false, 0, ""
	}
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return false, 0, ""
	}
	enabled = settings[SettingKeyBalanceLowNotifyEnabled] == "true"
	if v := settings[SettingKeyBalanceLowNotifyThreshold]; v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			threshold = f
		}
	}
	rechargeURL = strings.TrimRight(settings[SettingKeyBalanceLowNotifyRechargeURL], "/")
	return enabled, threshold, rechargeURL
}

// getSiteName reads the site name from settings.
func (s *BalanceNotifyService) getSiteName(ctx context.Context) string {
	if s.settingRepo == nil {
		return defaultSiteName
	}
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil || settings[SettingKeySiteName] == "" {
		return defaultSiteName
	}
	return settings[SettingKeySiteName]
}
