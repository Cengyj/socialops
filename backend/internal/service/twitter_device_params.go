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

// TwitterDeviceParamProvider supplies a real device fingerprint used to seed the
// Twitter login auth headers. A real fingerprint (client_uuid, device id,
// attest token, etc.) materially lowers Twitter's risk-control rejection rate
// compared to a static one, so the login flow fails closed when it is missing.
type TwitterDeviceParamProvider interface {
	Fetch(ctx context.Context) (*twitterAuthHeaders, error)
}

// TwitterDeviceParamConfig configures the external device-fingerprint API.
type TwitterDeviceParamConfig struct {
	URL        string
	Collection string
	MaxRetries int
	RetryDelay time.Duration
	Timeout    time.Duration
}

type httpDeviceParamProvider struct {
	cfg        TwitterDeviceParamConfig
	httpClient twitterHTTPClient
}

// NewHTTPDeviceParamProvider builds a device-fingerprint provider that calls the
// configured external API. It does not use a proxy (the fingerprint service is
// reached directly), mirroring the upstream FlyingBird behaviour.
func NewHTTPDeviceParamProvider(cfg TwitterDeviceParamConfig) *httpDeviceParamProvider {
	if cfg.Collection == "" {
		cfg.Collection = "twitter_parameter"
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 3 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &httpDeviceParamProvider{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

type twitterDeviceParamResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
	Data    struct {
		AttestToken          string `json:"attest_token"`
		SystemUserAgent      string `json:"system_user_agent"`
		ClientAdid           string `json:"client_adid"`
		ClientUuid           string `json:"client_uuid"`
		TwitterDisplaySize   string `json:"twitter_display_size"`
		ClientAppsetId       string `json:"client_appset_id"`
		ClientDeviceId       string `json:"client_device_id"`
		UserAgent            string `json:"user_agent"`
		ClientVersion        string `json:"client_version"`
		Timezone             string `json:"timezone"`
		OsVersion            string `json:"os_version"`
		AccessToken          string `json:"access_token"`
		TokenSecret          string `json:"token_secret"`
		OsSecurityPatchLevel string `json:"os_security_patch_level"`
	} `json:"data"`
}

func (p *httpDeviceParamProvider) Fetch(ctx context.Context) (*twitterAuthHeaders, error) {
	if p == nil || strings.TrimSpace(p.cfg.URL) == "" {
		return nil, newSocialExecutionError(SocialExecutionFailureConfiguration, "twitter device fingerprint provider is not configured", nil)
	}
	reqBody, err := json.Marshal(map[string]string{"collection": p.cfg.Collection})
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to build device fingerprint request", err)
	}

	var lastErr error
	for attempt := 1; attempt <= p.cfg.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, newSocialExecutionError(SocialExecutionFailureNetwork, "device fingerprint request cancelled", ctx.Err())
		default:
		}

		headers, err := p.fetchOnce(ctx, reqBody)
		if err == nil {
			return headers, nil
		}
		lastErr = err
		if attempt < p.cfg.MaxRetries {
			select {
			case <-time.After(p.cfg.RetryDelay):
			case <-ctx.Done():
				return nil, newSocialExecutionError(SocialExecutionFailureNetwork, "device fingerprint request cancelled", ctx.Err())
			}
		}
	}
	return nil, newSocialExecutionError(SocialExecutionFailureNetwork, "failed to fetch twitter device fingerprint", lastErr)
}

func (p *httpDeviceParamProvider) fetchOnce(ctx context.Context, reqBody []byte) (*twitterAuthHeaders, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.URL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := readTwitterResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device fingerprint API HTTP %d", resp.StatusCode)
	}
	var parsed twitterDeviceParamResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if !parsed.Success || parsed.Code != 200 {
		return nil, fmt.Errorf("device fingerprint API error: %s", strings.TrimSpace(parsed.Message))
	}

	headers := defaultTwitterAuthHeaders()
	applyTwitterDeviceParam(headers, &parsed)
	if strings.TrimSpace(headers.ClientUuid) == "" || strings.TrimSpace(headers.ClientDeviceId) == "" {
		return nil, fmt.Errorf("device fingerprint API returned incomplete parameters")
	}
	return headers, nil
}

func applyTwitterDeviceParam(headers *twitterAuthHeaders, parsed *twitterDeviceParamResponse) {
	if headers == nil || parsed == nil {
		return
	}
	d := parsed.Data
	setNonEmpty(&headers.AttestToken, d.AttestToken)
	setNonEmpty(&headers.SystemUserAgent, d.SystemUserAgent)
	setNonEmpty(&headers.ClientUuid, d.ClientUuid)
	setNonEmpty(&headers.TwitterDisplaySize, d.TwitterDisplaySize)
	setNonEmpty(&headers.ClientAppsetId, d.ClientAppsetId)
	setNonEmpty(&headers.ClientAdid, d.ClientAdid)
	setNonEmpty(&headers.ClientDeviceId, d.ClientDeviceId)
	setNonEmpty(&headers.UserAgent, d.UserAgent)
	setNonEmpty(&headers.ClientVersion, d.ClientVersion)
	setNonEmpty(&headers.Timezone, d.Timezone)
	setNonEmpty(&headers.OsVersion, d.OsVersion)
	setNonEmpty(&headers.OsSecurityPatchLevel, d.OsSecurityPatchLevel)
	setNonEmpty(&headers.AccessToken, d.AccessToken)
	setNonEmpty(&headers.TokenSecret, d.TokenSecret)
}

func setNonEmpty(dst *string, value string) {
	if dst == nil {
		return
	}
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		*dst = trimmed
	}
}
