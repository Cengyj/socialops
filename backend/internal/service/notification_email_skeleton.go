package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

const (
	notificationEmailTemplateKeyPrefix    = "notification_email_template:"
	notificationEmailPreferenceKeyPrefix  = "notification_email_preference:"
	notificationEmailDeliveryKeyPrefix    = "notification_email_delivery:"
	notificationEmailLocaleUserKeyPrefix  = "notification_email_locale:user:"
	notificationEmailLocaleEmailKeyPrefix = "notification_email_locale:email:"
)

var notificationEmailPlaceholderPattern = regexp.MustCompile(`{{\s*([a-zA-Z][a-zA-Z0-9_]*)\s*}}`)

type NotificationEmailEventInfo struct {
	Event        string
	Label        string
	Description  string
	Category     string
	Optional     bool
	Placeholders []string
}

type NotificationEmailTemplate struct {
	Event        string
	Locale       string
	Subject      string
	HTML         string
	IsCustom     bool
	UpdatedAt    *time.Time
	Placeholders []string
}

type NotificationEmailPreviewInput struct {
	Event     string
	Locale    string
	Subject   string
	HTML      string
	Variables map[string]string
}

type NotificationEmailPreview struct {
	Subject string
	HTML    string
}

type NotificationEmailUnsubscribeResult struct {
	Event string
	Email string
	Done  bool
}

type notificationEmailStoredTemplate struct {
	Subject   string    `json:"subject"`
	HTML      string    `json:"html"`
	UpdatedAt time.Time `json:"updated_at"`
}

type notificationEmailWrappedError struct {
	err error
	tag string
}

func (e notificationEmailWrappedError) Error() string { return e.err.Error() }
func (e notificationEmailWrappedError) Unwrap() error { return e.err }

func NewNotificationEmailService(settingRepo SettingRepository, emailService *EmailService) *NotificationEmailService {
	svc := &NotificationEmailService{settingRepo: settingRepo, emailService: emailService}
	if emailService != nil {
		emailService.SetNotificationEmailService(svc)
	}
	return svc
}

func notificationEmailTemplateErr(err error) error {
	if err == nil {
		return nil
	}
	return notificationEmailWrappedError{err: err, tag: "template"}
}

func notificationEmailConfigErr(err error) error {
	if err == nil {
		return nil
	}
	return notificationEmailWrappedError{err: err, tag: "config"}
}

func notificationEmailDeliveryErr(err error) error {
	if err == nil {
		return nil
	}
	return notificationEmailWrappedError{err: err, tag: "delivery"}
}

func shouldFallbackNotificationEmail(err error) bool {
	var wrapped notificationEmailWrappedError
	return errors.As(err, &wrapped) && (wrapped.tag == "template" || wrapped.tag == "config")
}

func isNotificationEmailDeliveryError(err error) bool {
	var wrapped notificationEmailWrappedError
	return errors.As(err, &wrapped) && wrapped.tag == "delivery"
}

func (s *NotificationEmailService) SupportedLocales() []string {
	return []string{"zh", "en"}
}

func (s *NotificationEmailService) ListEventInfos() []NotificationEmailEventInfo {
	return []NotificationEmailEventInfo{
		{Event: NotificationEmailEventAuthVerifyCode, Label: "Auth verification code", Category: "auth", Placeholders: []string{"verification_code", "expires_in_minutes"}},
		{Event: NotificationEmailEventAuthPasswordReset, Label: "Password reset", Category: "auth", Placeholders: []string{"reset_url", "expires_in_minutes"}},
		{Event: NotificationEmailEventNotificationEmailVerifyCode, Label: "Notification email verification", Category: "auth", Placeholders: []string{"verification_code"}},
		{Event: NotificationEmailEventSubscriptionPurchaseSuccess, Label: "Subscription purchase success", Category: "subscription", Placeholders: []string{"subscription_group", "plan_name", "expires_at"}},
		{Event: NotificationEmailEventSubscriptionExpiryReminder, Label: "Subscription expiry reminder", Category: "subscription", Optional: true, Placeholders: []string{"subscription_group", "expiry_time", "days_remaining", "unsubscribe_url"}},
		{Event: NotificationEmailEventBalanceLow, Label: "Balance low", Category: "billing", Optional: true, Placeholders: []string{"current_balance", "balance", "threshold", "recharge_url", "unsubscribe_url"}},
		{Event: NotificationEmailEventBalanceRechargeSuccess, Label: "Balance recharge success", Category: "billing", Placeholders: []string{"recharge_amount", "amount"}},
	}
}

func (s *NotificationEmailService) ListTemplates(ctx context.Context) ([]NotificationEmailTemplate, error) {
	events := s.ListEventInfos()
	locales := s.SupportedLocales()
	out := make([]NotificationEmailTemplate, 0, len(events)*len(locales))
	for _, event := range events {
		for _, locale := range locales {
			tmpl, err := s.GetTemplate(ctx, event.Event, locale)
			if err != nil {
				return nil, err
			}
			out = append(out, tmpl)
		}
	}
	return out, nil
}

func (s *NotificationEmailService) GetTemplate(ctx context.Context, event, locale string) (NotificationEmailTemplate, error) {
	info := s.findEvent(event)
	if info.Event == "" {
		return NotificationEmailTemplate{}, ErrSettingNotFound
	}
	locale = normalizeNotificationEmailLocale(locale)

	if s != nil && s.settingRepo != nil {
		raw, err := s.settingRepo.GetValue(ctx, notificationEmailTemplateKey(info.Event, locale))
		if err == nil && strings.TrimSpace(raw) != "" {
			var stored notificationEmailStoredTemplate
			if json.Unmarshal([]byte(raw), &stored) == nil {
				updatedAt := stored.UpdatedAt
				return NotificationEmailTemplate{
					Event:        info.Event,
					Locale:       locale,
					Subject:      stored.Subject,
					HTML:         stored.HTML,
					IsCustom:     true,
					UpdatedAt:    &updatedAt,
					Placeholders: append([]string(nil), info.Placeholders...),
				}, nil
			}
		}
	}

	return NotificationEmailTemplate{
		Event:        info.Event,
		Locale:       locale,
		Subject:      defaultNotificationEmailSubject(info, locale),
		HTML:         defaultNotificationEmailHTML(info),
		Placeholders: append([]string(nil), info.Placeholders...),
	}, nil
}

func (s *NotificationEmailService) UpdateTemplate(ctx context.Context, event, locale, subject, htmlBody string) (NotificationEmailTemplate, error) {
	info := s.findEvent(event)
	if info.Event == "" {
		return NotificationEmailTemplate{}, ErrSettingNotFound
	}
	if err := validateNotificationEmailPlaceholders(info, subject, htmlBody); err != nil {
		return NotificationEmailTemplate{}, err
	}
	locale = normalizeNotificationEmailLocale(locale)
	now := time.Now().UTC()
	if s != nil && s.settingRepo != nil {
		payload, err := json.Marshal(notificationEmailStoredTemplate{Subject: strings.TrimSpace(subject), HTML: htmlBody, UpdatedAt: now})
		if err != nil {
			return NotificationEmailTemplate{}, err
		}
		if err := s.settingRepo.Set(ctx, notificationEmailTemplateKey(info.Event, locale), string(payload)); err != nil {
			return NotificationEmailTemplate{}, err
		}
	}
	return NotificationEmailTemplate{
		Event:        info.Event,
		Locale:       locale,
		Subject:      strings.TrimSpace(subject),
		HTML:         htmlBody,
		IsCustom:     true,
		UpdatedAt:    &now,
		Placeholders: append([]string(nil), info.Placeholders...),
	}, nil
}

func (s *NotificationEmailService) RestoreOfficialTemplate(ctx context.Context, event, locale string) (NotificationEmailTemplate, error) {
	info := s.findEvent(event)
	if info.Event == "" {
		return NotificationEmailTemplate{}, ErrSettingNotFound
	}
	locale = normalizeNotificationEmailLocale(locale)
	if s != nil && s.settingRepo != nil {
		if err := s.settingRepo.Delete(ctx, notificationEmailTemplateKey(info.Event, locale)); err != nil && !errors.Is(err, ErrSettingNotFound) {
			return NotificationEmailTemplate{}, err
		}
	}
	return s.GetTemplate(ctx, info.Event, locale)
}

func (s *NotificationEmailService) PreviewTemplate(ctx context.Context, input NotificationEmailPreviewInput) (NotificationEmailPreview, error) {
	subject := strings.TrimSpace(input.Subject)
	body := input.HTML
	if subject == "" || strings.TrimSpace(body) == "" {
		tmpl, err := s.GetTemplate(ctx, input.Event, input.Locale)
		if err != nil {
			return NotificationEmailPreview{}, err
		}
		if subject == "" {
			subject = tmpl.Subject
		}
		if strings.TrimSpace(body) == "" {
			body = tmpl.HTML
		}
	}
	return renderNotificationEmail(input.Event, subject, body, input.Variables, nil)
}

func (s *NotificationEmailService) Send(ctx context.Context, input NotificationEmailSendInput) error {
	if input.RecipientEmail == "" {
		return nil
	}
	if input.Event == "" {
		input.Event = NotificationEmailEventBalanceLow
	}
	if unsubscribed, err := s.IsUnsubscribed(ctx, input.RecipientEmail, input.Event); err != nil {
		return err
	} else if unsubscribed {
		return nil
	}

	deliveryKey := notificationEmailDeliveryKey(input.Event, input.SourceType, input.SourceID, input.RecipientEmail, input.ReminderKey)
	legacyKey := legacyNotificationEmailDeliveryKey(input.Event, input.SourceType, input.SourceID, input.RecipientEmail, input.ReminderKey)
	if exists, err := s.deliveryExists(ctx, deliveryKey, legacyKey); err != nil {
		return err
	} else if exists {
		return nil
	}

	tmpl, err := s.GetTemplate(ctx, input.Event, input.Locale)
	if err != nil {
		return notificationEmailTemplateErr(err)
	}
	vars := mergeNotificationEmailVariables(input)
	preview, err := renderNotificationEmail(input.Event, tmpl.Subject, tmpl.HTML, vars, nil)
	if err != nil {
		return notificationEmailTemplateErr(err)
	}
	if s == nil || s.emailService == nil {
		return notificationEmailConfigErr(errors.New("email service is unavailable"))
	}
	if err := s.emailService.SendEmail(ctx, input.RecipientEmail, preview.Subject, preview.HTML); err != nil {
		return notificationEmailDeliveryErr(err)
	}
	if s.settingRepo != nil && input.SourceType != "" && input.SourceID != "" && input.ReminderKey != "" {
		_ = s.settingRepo.Set(ctx, deliveryKey, time.Now().UTC().Format(time.RFC3339))
	}
	return nil
}

func (s *NotificationEmailService) RememberRecipientLocale(ctx context.Context, userID int64, email, acceptLanguage string) {
	if s == nil || s.settingRepo == nil {
		return
	}
	locale := normalizeNotificationEmailLocale(acceptLanguage)
	if userID > 0 {
		_ = s.settingRepo.Set(ctx, notificationEmailLocaleUserKeyPrefix+fmt.Sprint(userID), locale)
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email != "" {
		_ = s.settingRepo.Set(ctx, notificationEmailLocaleEmailKeyPrefix+email, locale)
	}
}

func (s *NotificationEmailService) ResolveRecipientLocale(ctx context.Context, userID int64, email string) string {
	if s == nil || s.settingRepo == nil {
		return "en"
	}
	if userID > 0 {
		if value, err := s.settingRepo.GetValue(ctx, notificationEmailLocaleUserKeyPrefix+fmt.Sprint(userID)); err == nil && value != "" {
			return normalizeNotificationEmailLocale(value)
		}
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email != "" {
		if value, err := s.settingRepo.GetValue(ctx, notificationEmailLocaleEmailKeyPrefix+email); err == nil && value != "" {
			return normalizeNotificationEmailLocale(value)
		}
	}
	return "en"
}

func (s *NotificationEmailService) IsUnsubscribed(ctx context.Context, email, event string) (bool, error) {
	if s == nil || s.settingRepo == nil {
		return false, nil
	}
	for _, key := range []string{
		notificationEmailPreferenceKey(event, email),
		legacyNotificationEmailPreferenceKey(event, email),
	} {
		if _, err := s.settingRepo.GetValue(ctx, key); err == nil {
			return true, nil
		} else if !errors.Is(err, ErrSettingNotFound) {
			return false, err
		}
	}
	return false, nil
}

func (s *NotificationEmailService) Unsubscribe(ctx context.Context, token string) (NotificationEmailUnsubscribeResult, error) {
	email, event, err := parseNotificationEmailUnsubscribeToken(token)
	if err != nil {
		return NotificationEmailUnsubscribeResult{}, err
	}
	info := s.findEvent(event)
	if info.Event == "" {
		return NotificationEmailUnsubscribeResult{}, ErrSettingNotFound
	}
	if !info.Optional {
		return NotificationEmailUnsubscribeResult{}, infraerrors.BadRequest("NOTIFICATION_EMAIL_TRANSACTIONAL", "transactional email events cannot be unsubscribed")
	}
	if s != nil && s.settingRepo != nil {
		if err := s.settingRepo.Set(ctx, notificationEmailPreferenceKey(event, email), "unsubscribed"); err != nil {
			return NotificationEmailUnsubscribeResult{}, err
		}
	}
	return NotificationEmailUnsubscribeResult{Event: event, Email: strings.ToLower(email), Done: true}, nil
}

func (s *NotificationEmailService) createUnsubscribeToken(_ context.Context, email, event string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || event == "" {
		return "", infraerrors.BadRequest("INVALID_UNSUBSCRIBE_TOKEN", "email and event are required")
	}
	return base64.RawURLEncoding.EncodeToString([]byte(email + "|" + event)), nil
}

func parseNotificationEmailUnsubscribeToken(token string) (string, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return "", "", infraerrors.BadRequest("INVALID_UNSUBSCRIBE_TOKEN", "unsubscribe token is invalid")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", infraerrors.BadRequest("INVALID_UNSUBSCRIBE_TOKEN", "unsubscribe token is invalid")
	}
	return parts[0], parts[1], nil
}

func (s *NotificationEmailService) deliveryExists(ctx context.Context, keys ...string) (bool, error) {
	if s == nil || s.settingRepo == nil {
		return false, nil
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, err := s.settingRepo.GetValue(ctx, key); err == nil {
			return true, nil
		} else if !errors.Is(err, ErrSettingNotFound) {
			return false, err
		}
	}
	return false, nil
}

func (s *NotificationEmailService) findEvent(event string) NotificationEmailEventInfo {
	event = strings.TrimSpace(event)
	for _, info := range s.ListEventInfos() {
		if info.Event == event {
			return info
		}
	}
	return NotificationEmailEventInfo{}
}

func validateNotificationEmailPlaceholders(info NotificationEmailEventInfo, values ...string) error {
	allowed := map[string]struct{}{
		"site_name":        {},
		"recipient_name":   {},
		"recipient_email":  {},
		"unsubscribe_url":  {},
		"current_balance":  {},
		"recharge_url":     {},
		"recharge_amount":  {},
		"subscription_url": {},
	}
	for _, name := range info.Placeholders {
		allowed[name] = struct{}{}
	}
	for _, value := range values {
		for _, match := range notificationEmailPlaceholderPattern.FindAllStringSubmatch(value, -1) {
			if _, ok := allowed[match[1]]; !ok {
				return fmt.Errorf("unsupported placeholder: %s", match[1])
			}
		}
	}
	return nil
}

func mergeNotificationEmailVariables(input NotificationEmailSendInput) map[string]string {
	vars := map[string]string{
		"recipient_email": input.RecipientEmail,
		"recipient_name":  firstNonEmpty(input.RecipientName, input.RecipientEmail),
		"site_name":       "SocialOps",
	}
	for key, value := range input.Variables {
		vars[key] = value
	}
	return vars
}

func renderNotificationEmail(_ string, subject, body string, variables map[string]string, rawHTMLVariables map[string]string) (NotificationEmailPreview, error) {
	subject = sanitizeEmailHeader(subject)
	for key, value := range variables {
		subject = strings.ReplaceAll(subject, "{{"+key+"}}", sanitizeEmailHeader(value))
		escaped := html.EscapeString(value)
		if strings.HasSuffix(key, "_url") && strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "javascript:") {
			escaped = ""
		}
		body = strings.ReplaceAll(body, "{{"+key+"}}", escaped)
	}
	for key, value := range rawHTMLVariables {
		body = strings.ReplaceAll(body, "{{"+key+"}}", value)
	}
	body = notificationEmailPlaceholderPattern.ReplaceAllString(body, "")
	subject = notificationEmailPlaceholderPattern.ReplaceAllString(subject, "")
	return NotificationEmailPreview{Subject: subject, HTML: body}, nil
}

func notificationEmailTemplateKey(event, locale string) string {
	return notificationEmailTemplateKeyPrefix + event + ":" + normalizeNotificationEmailLocale(locale)
}

func notificationEmailPreferenceKey(event, email string) string {
	return notificationEmailPreferenceKeyPrefix + "v2:" + notificationEmailHash(event, strings.ToLower(strings.TrimSpace(email)))
}

func legacyNotificationEmailPreferenceKey(event, email string) string {
	return notificationEmailPreferenceKeyPrefix + event + ":" + strings.ToLower(strings.TrimSpace(email)) + ":" + strings.Repeat("legacy", 20)
}

func notificationEmailDeliveryKey(event, sourceType, sourceID, email, reminderKey string) string {
	return notificationEmailDeliveryKeyPrefix + "v2:" + notificationEmailHash(event, sourceType, sourceID, strings.ToLower(strings.TrimSpace(email)), reminderKey)
}

func legacyNotificationEmailDeliveryKey(event, sourceType, sourceID, email, reminderKey string) string {
	return notificationEmailDeliveryKeyPrefix + event + ":" + sourceType + ":" + sourceID + ":" + strings.ToLower(strings.TrimSpace(email)) + ":" + reminderKey + ":" + strings.Repeat("legacy", 20)
}

func notificationEmailHash(values ...string) string {
	h := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hex.EncodeToString(h[:])[:24]
}

func normalizeNotificationEmailLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(locale, "zh") || strings.Contains(locale, "zh-") {
		return "zh"
	}
	return "en"
}

func defaultNotificationEmailSubject(info NotificationEmailEventInfo, locale string) string {
	switch info.Event {
	case NotificationEmailEventAuthVerifyCode:
		if normalizeNotificationEmailLocale(locale) == "zh" {
			return "邮箱验证码"
		}
		return "Email verification code"
	case NotificationEmailEventAuthPasswordReset:
		if normalizeNotificationEmailLocale(locale) == "zh" {
			return "重置密码"
		}
		return "Password reset"
	case NotificationEmailEventBalanceRechargeSuccess:
		return "Balance recharge success"
	case NotificationEmailEventSubscriptionPurchaseSuccess:
		return "Subscription purchase success"
	case NotificationEmailEventSubscriptionExpiryReminder:
		return "Subscription expiry reminder"
	case NotificationEmailEventBalanceLow:
		return "Balance low alert"
	default:
		return firstNonEmpty(info.Label, "SocialOps notification")
	}
}

func defaultNotificationEmailHTML(info NotificationEmailEventInfo) string {
	switch info.Event {
	case NotificationEmailEventAuthVerifyCode:
		return "<p>{{verification_code}}</p>"
	case NotificationEmailEventAuthPasswordReset:
		return `<p><a href="{{reset_url}}">{{reset_url}}</a></p>`
	case NotificationEmailEventSubscriptionExpiryReminder:
		return "<p>{{subscription_group}} {{expiry_time}} {{days_remaining}}</p>"
	case NotificationEmailEventBalanceLow:
		return `<p>{{current_balance}} {{threshold}}</p><a href="{{recharge_url}}">Recharge</a>`
	case NotificationEmailEventBalanceRechargeSuccess:
		return "<p>{{recipient_name}} {{recharge_amount}}</p>"
	case NotificationEmailEventSubscriptionPurchaseSuccess:
		return "<p>{{subscription_group}}</p>"
	default:
		return "<p>{{site_name}}</p>"
	}
}
