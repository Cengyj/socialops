package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

var ErrSocialAccountExecutionAuthInvalid = errors.New("social account execution auth is invalid")

// twitterExecutionAuthPayload is the only plaintext shape accepted before
// execution_auth is encrypted for storage. It is built from access_token,
// token_secret, and screen_name; full cookies, device fingerprints, guest
// tokens, and request header fields stay out of this value.
type twitterExecutionAuthPayload struct {
	AccessToken string `json:"access_token"`
	TokenSecret string `json:"token_secret"`
	ScreenName  string `json:"screen_name"`
}

func twitterExecutionAuthFromHeaders(headers *twitterAuthHeaders) (string, error) {
	if headers == nil {
		return "", fmt.Errorf("twitter auth headers are unavailable")
	}
	payload := twitterExecutionAuthPayload{
		AccessToken: strings.TrimSpace(headers.AccessToken),
		TokenSecret: strings.TrimSpace(headers.TokenSecret),
		ScreenName:  normalizeTwitterExecutionScreenName(headers.ScreenName),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal twitter execution auth payload: %w", err)
	}
	return string(encoded), nil
}

func twitterAuthHeadersFromExecutionAuth(raw string) (*twitterAuthHeaders, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthMissing, "account execution auth is required", nil)
	}
	parsed, err := parseTwitterExecutionAuthMap(raw)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthInvalid, "account execution auth is invalid", err)
	}
	if twitterExecutionAuthHasUnsupportedShape(parsed) {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthInvalid, "account execution auth uses an unsupported shape", nil)
	}
	return twitterAuthHeadersFromMap(parsed)
}

func normalizeTwitterExecutionAuthForStorage(raw string, screenName string) (string, error) {
	original := raw
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return trimmed, nil
	}
	parsed, err := parseTwitterExecutionAuthMap(trimmed)
	if err != nil {
		return original, nil
	}
	if twitterExecutionAuthHasUnsupportedShape(parsed) {
		return "", invalidSocialAccountExecutionAuthError(ErrSocialAccountExecutionAuthInvalid)
	}
	if normalizeTwitterExecutionScreenName(stringMapValue(parsed, "screen_name")) == "" {
		if fallback := normalizeTwitterExecutionScreenName(screenName); fallback != "" {
			parsed["screen_name"] = fallback
		}
	}
	headers, err := twitterAuthHeadersFromMap(parsed)
	if err != nil {
		return "", invalidSocialAccountExecutionAuthError(ErrSocialAccountExecutionAuthInvalid)
	}
	if strings.TrimSpace(headers.AccessToken) == "" || strings.TrimSpace(headers.TokenSecret) == "" {
		return "", invalidSocialAccountExecutionAuthError(ErrSocialAccountExecutionAuthInvalid)
	}
	encoded, err := twitterExecutionAuthFromHeaders(headers)
	if err != nil {
		return "", err
	}
	return encoded, nil
}

func normalizeTwitterExecutionAuthForEncryptedStorage(raw string, screenName string, encryptor ExecutionAuthEncryptor) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return trimmed, nil
	}
	if !strings.HasPrefix(trimmed, "{") {
		return trimmed, nil
	}
	normalized, err := normalizeTwitterExecutionAuthForStorage(trimmed, screenName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(normalized) == "" {
		return normalized, nil
	}
	return encryptTwitterExecutionAuth(normalized, encryptor)
}

func twitterAuthHeadersFromStoredExecutionAuth(raw string, encryptor ExecutionAuthEncryptor) (*twitterAuthHeaders, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthMissing, "account execution auth is required", nil)
	}
	plaintext, err := decryptTwitterExecutionAuthCiphertext(raw, encryptor)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthInvalid, "account execution auth is invalid", err)
	}
	return twitterAuthHeadersFromExecutionAuth(plaintext)
}

func decryptTwitterExecutionAuthCiphertext(raw string, encryptor ExecutionAuthEncryptor) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || encryptor == nil {
		return "", ErrSocialAccountExecutionAuthInvalid
	}
	plaintext, err := encryptor.Decrypt(trimmed)
	if err != nil {
		return "", err
	}
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return "", ErrSocialAccountExecutionAuthInvalid
	}
	return plaintext, nil
}

func encryptTwitterExecutionAuth(plaintext string, encryptor ExecutionAuthEncryptor) (string, error) {
	if encryptor == nil {
		return "", fmt.Errorf("encrypt twitter execution auth: credential encryptor is required")
	}
	encrypted, err := encryptor.Encrypt(strings.TrimSpace(plaintext))
	if err != nil {
		return "", fmt.Errorf("encrypt twitter execution auth: %w", err)
	}
	return encrypted, nil
}

func invalidSocialAccountExecutionAuthError(cause error) error {
	return infraerrors.BadRequest("SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID", "account execution auth is invalid").WithCause(cause)
}

func parseTwitterExecutionAuthMap(raw string) (map[string]any, error) {
	return parseJSONMap(strings.TrimSpace(raw))
}

func parseJSONMap(raw string) (map[string]any, error) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func twitterAuthHeadersFromMap(data map[string]any) (*twitterAuthHeaders, error) {
	headers := defaultTwitterAuthHeaders()
	headers.AccessToken = stringMapValue(data, "access_token")
	headers.TokenSecret = stringMapValue(data, "token_secret")
	headers.ScreenName = normalizeTwitterExecutionScreenName(stringMapValue(data, "screen_name"))
	if strings.TrimSpace(headers.AccessToken) == "" || strings.TrimSpace(headers.TokenSecret) == "" {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthInvalid, "account execution auth is missing OAuth credentials", nil)
	}
	if strings.TrimSpace(headers.ClientUuid) == "" {
		headers.ClientUuid = generateTwitterClientUUID()
	}
	if strings.TrimSpace(headers.TraceId) == "" {
		headers.TraceId = generateTwitterTraceID()
	}
	if strings.TrimSpace(headers.AcceptEncoding) == "" || strings.Contains(strings.ToLower(headers.AcceptEncoding), "br") {
		headers.AcceptEncoding = "gzip"
	}
	return headers, nil
}

func twitterExecutionAuthHasUnsupportedShape(data map[string]any) bool {
	if data == nil {
		return false
	}
	if kind := stringMapValue(data, "kind"); kind != "" {
		return true
	}
	if encryption := stringMapValue(data, "encryption"); encryption != "" {
		return true
	}
	allowed := map[string]struct{}{
		"access_token": {},
		"token_secret": {},
		"screen_name":  {},
	}
	for key := range data {
		if _, ok := allowed[key]; !ok {
			return true
		}
	}
	return false
}

func normalizeTwitterExecutionScreenName(value string) string {
	return strings.TrimLeft(strings.TrimSpace(value), "@")
}

func stringMapValue(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	switch value := data[key].(type) {
	case string:
		return strings.TrimSpace(value)
	case json.Number:
		return strings.TrimSpace(value.String())
	case float64:
		return strings.TrimSpace(strconv.FormatFloat(value, 'f', -1, 64))
	default:
		return ""
	}
}
