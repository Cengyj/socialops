package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// TwitterEmailCodeConfig configures the external email verification-code API
// used when Twitter's login flow issues an email challenge (LoginAcid).
type TwitterEmailCodeConfig struct {
	URL         string
	MaxAttempts int
	PollDelay   time.Duration
	Timeout     time.Duration
}

type httpEmailCodeResolver struct {
	cfg        TwitterEmailCodeConfig
	httpClient twitterHTTPClient
	sleep      func(time.Duration)
}

// NewHTTPEmailCodeResolver builds an email verification-code resolver that polls
// the configured external API. It returns a twitterEmailCodeResolver function
// compatible with the login session. The mailbox API is reached directly
// (no proxy), mirroring upstream FlyingBird behaviour.
func NewHTTPEmailCodeResolver(cfg TwitterEmailCodeConfig) twitterEmailCodeResolver {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 10
	}
	if cfg.PollDelay <= 0 {
		cfg.PollDelay = 3 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	resolver := &httpEmailCodeResolver{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
		sleep:      func(d time.Duration) { time.Sleep(d) },
	}
	return resolver.resolve
}

type twitterEmailCodeResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	VerifyCode  string `json:"verify_code"`
	AccessToken string `json:"access_token"`
}

func (r *httpEmailCodeResolver) resolve(ctx context.Context, req SocialAccountCredentialRequest) (string, error) {
	if r == nil || strings.TrimSpace(r.cfg.URL) == "" {
		return "", fmt.Errorf("twitter email verification provider is not configured")
	}
	if strings.TrimSpace(req.Email) == "" {
		return "", fmt.Errorf("account email is required for email verification")
	}

	// max_time is "now" in Hawaiian time minus 115 minutes, matching upstream.
	hawaiian := time.Now().UTC().Add(-10 * time.Hour).Add(-115 * time.Minute)
	payload := map[string]string{
		"email":         req.Email,
		"password":      req.EmailPassword,
		"refresh_token": req.EmailToken,
		"client_id":     req.EmailClientID,
		"operation":     "login",
		"max_time":      hawaiian.Format("2006-01-02 15:04:05"),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to build email verification request: %w", err)
	}

	accessToken := ""
	var lastErr error
	for attempt := 0; attempt < r.cfg.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		parsed, err := r.requestOnce(ctx, body)
		if err != nil {
			lastErr = err
			r.sleep(r.cfg.PollDelay)
			continue
		}
		if parsed.Success && strings.TrimSpace(parsed.VerifyCode) != "" {
			return strings.TrimSpace(parsed.VerifyCode), nil
		}

		// The API may hand back an access_token to switch from refresh_token auth.
		if accessToken == "" && strings.TrimSpace(parsed.AccessToken) != "" {
			accessToken = strings.TrimSpace(parsed.AccessToken)
			delete(payload, "refresh_token")
			payload["access_token"] = accessToken
			if body, err = json.Marshal(payload); err != nil {
				return "", fmt.Errorf("failed to rebuild email verification request: %w", err)
			}
		}

		// A non-"mail not found" business error is terminal.
		if !parsed.Success && strings.TrimSpace(parsed.Message) != "" && !isEmailCodeRetryable(parsed.Message) {
			return "", fmt.Errorf("%s", strings.TrimSpace(parsed.Message))
		}
		lastErr = fmt.Errorf("email verification code not yet available")
		r.sleep(r.cfg.PollDelay)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("email verification code timed out")
	}
	return "", lastErr
}

func (r *httpEmailCodeResolver) requestOnce(ctx context.Context, body []byte) (*twitterEmailCodeResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := readTwitterResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		return nil, fmt.Errorf("email verification API HTTP %d", resp.StatusCode)
	}
	if len(respBody) == 0 {
		return nil, fmt.Errorf("email verification API returned empty response")
	}
	var parsed twitterEmailCodeResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func isEmailCodeRetryable(message string) bool {
	return strings.Contains(message, "未找到符合条件的邮件")
}
