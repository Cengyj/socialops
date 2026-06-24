package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	twitterLoginNetworkMaxRetries = 3
	twitterLoginMaxFlowDepth      = 12
)

type twitterLoginEndpoints struct {
	guestToken      string
	onboarding      string
	onboardingFirst string
	viewerUser      string
}

func defaultTwitterLoginEndpoints() twitterLoginEndpoints {
	return twitterLoginEndpoints{
		guestToken:      "https://api.twitter.com/1.1/guest/activate.json",
		onboarding:      "https://api.twitter.com/1.1/onboarding/task.json",
		onboardingFirst: "https://api.twitter.com/1.1/onboarding/task.json?flow_name=login&api_version=1&known_device_token=&sim_country_code=cn",
		viewerUser:      defaultTwitterEndpoints().viewerUser,
	}
}

type TwitterAccountCredentialRegistrar struct {
	clientForProxy      func(proxyURL string) (twitterHTTPClient, error)
	endpoints           twitterLoginEndpoints
	now                 func() time.Time
	emailCodeResolver   twitterEmailCodeResolver
	deviceParamProv     TwitterDeviceParamProvider
	proxyHealthReporter func(context.Context, int64)
}

type twitterEmailCodeResolver func(ctx context.Context, req SocialAccountCredentialRequest) (string, error)

func NewTwitterAccountCredentialRegistrar() *TwitterAccountCredentialRegistrar {
	return &TwitterAccountCredentialRegistrar{
		clientForProxy: defaultTwitterHTTPClient,
		endpoints:      defaultTwitterLoginEndpoints(),
	}
}

func (r *TwitterAccountCredentialRegistrar) WithDeviceParamProvider(p TwitterDeviceParamProvider) *TwitterAccountCredentialRegistrar {
	if r != nil {
		r.deviceParamProv = p
	}
	return r
}

func (r *TwitterAccountCredentialRegistrar) WithEmailCodeResolver(resolver twitterEmailCodeResolver) *TwitterAccountCredentialRegistrar {
	if r != nil {
		r.emailCodeResolver = resolver
	}
	return r
}

func (r *TwitterAccountCredentialRegistrar) WithProxyHealthReporter(reporter func(context.Context, int64)) *TwitterAccountCredentialRegistrar {
	if r != nil {
		r.proxyHealthReporter = reporter
	}
	return r
}

func (r *TwitterAccountCredentialRegistrar) Supports(platform string) bool {
	return isTwitterPlatform(platform)
}

func (r *TwitterAccountCredentialRegistrar) AcquireCredentials(ctx context.Context, req SocialAccountCredentialRequest) (*SocialAccountCredentialResult, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Password) == "" {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthMissing, "account and password are required", nil)
	}
	clientFactory := r.clientForProxy
	if clientFactory == nil {
		clientFactory = defaultTwitterHTTPClient
	}
	endpoints := r.endpoints
	if endpoints.guestToken == "" {
		endpoints = defaultTwitterLoginEndpoints()
	}
	// Resolve the device fingerprint before building the proxy client so a
	// fingerprint failure fails closed without starting the login.
	auth, err := r.resolveAuthHeaders(ctx)
	if err != nil {
		return nil, err
	}
	httpClient, err := clientFactory(strings.TrimSpace(req.ProxyEndpoint))
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailureProxyInvalid, "execution proxy is invalid", err)
	}
	session := &twitterLoginSession{
		httpClient:        httpClient,
		endpoints:         endpoints,
		auth:              auth,
		account:           req,
		now:               r.now,
		emailCodeResolver: r.emailCodeResolver,
		onHTTPResponse: func() {
			if r.proxyHealthReporter != nil && req.ProxyID > 0 {
				r.proxyHealthReporter(ctx, req.ProxyID)
			}
		},
	}
	return session.login(ctx)
}

// resolveAuthHeaders seeds the login session with a real device fingerprint when
// a provider is configured. A static fingerprint dramatically increases Twitter
// risk-control rejection, so when a provider is configured but fails the login
// fails closed rather than silently falling back to the static defaults.
func (r *TwitterAccountCredentialRegistrar) resolveAuthHeaders(ctx context.Context) (*twitterAuthHeaders, error) {
	if r == nil || r.deviceParamProv == nil {
		return defaultTwitterAuthHeaders(), nil
	}
	headers, err := r.deviceParamProv.Fetch(ctx)
	if err != nil {
		return nil, err
	}
	if headers == nil {
		return nil, newSocialExecutionError(SocialExecutionFailureNetwork, "twitter device fingerprint is unavailable", nil)
	}
	return headers, nil
}

type twitterLoginSession struct {
	httpClient        twitterHTTPClient
	endpoints         twitterLoginEndpoints
	auth              *twitterAuthHeaders
	account           SocialAccountCredentialRequest
	now               func() time.Time
	emailCodeResolver twitterEmailCodeResolver
	onHTTPResponse    func()
}

type twitterLoginResponse struct {
	FlowToken string                `json:"flow_token"`
	Status    string                `json:"status"`
	Subtasks  []twitterLoginSubtask `json:"subtasks"`
	Errors    []twitterLoginError   `json:"errors"`
	RawBody   string                `json:"-"`
}

type twitterLoginSubtask struct {
	SubtaskID     string                   `json:"subtask_id"`
	OpenAccount   *twitterLoginOpenAccount `json:"open_account"`
	EnterText     *twitterLoginTextInput   `json:"enter_text"`
	EnterPassword *twitterLoginTextInput   `json:"enter_password"`
}

type twitterLoginTextInput struct {
	HintText      string            `json:"hint_text"`
	ErrorMessage  string            `json:"error_message"`
	PrimaryText   *twitterLoginText `json:"primary_text"`
	SecondaryText *twitterLoginText `json:"secondary_text"`
}

type twitterLoginText struct {
	Text string `json:"text"`
}

type twitterLoginError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type twitterLoginOpenAccount struct {
	OauthToken       string `json:"oauth_token"`
	OauthTokenSecret string `json:"oauth_token_secret"`
	KnownDeviceToken string `json:"known_device_token"`
}

func (s *twitterLoginSession) login(ctx context.Context) (*SocialAccountCredentialResult, error) {
	if err := s.fetchGuestToken(ctx); err != nil {
		return nil, err
	}
	if err := s.fetchOnboardingTasks(ctx, nil, 0, ""); err != nil {
		return nil, err
	}
	return s.credentialResult(ctx)
}

func (s *twitterLoginSession) fetchOnboardingTasks(ctx context.Context, formData []byte, depth int, lastSubmittedSubtask string) error {
	if depth >= twitterLoginMaxFlowDepth {
		return newSocialExecutionError(SocialExecutionFailurePlatform, "twitter login flow did not complete", nil)
	}

	firstRequest := len(formData) == 0
	if firstRequest {
		var err error
		formData, err = buildTwitterLoginInitialForm()
		if err != nil {
			return err
		}
	}

	resp, header, err := s.fetchOnboardingTask(ctx, formData, firstRequest)
	if err != nil {
		return err
	}
	if resp == nil {
		return newSocialExecutionError(SocialExecutionFailurePlatform, "twitter login response is empty", nil)
	}
	if len(resp.Subtasks) == 0 {
		if err := twitterLoginResponseFailure(resp, lastSubmittedSubtask); err != nil {
			return err
		}
		if s.auth != nil && s.auth.AccessToken != "" && s.auth.TokenSecret != "" && resp.Status == "success" {
			s.auth.ScreenName = normalizeTwitterExecutionScreenName(s.account.Name)
			return nil
		}
		return newSocialExecutionError(SocialExecutionFailurePlatform, "twitter login flow returned no next step", nil)
	}

	subtask := resp.Subtasks[0]
	if err := twitterLoginSubtaskFailure(resp, subtask, lastSubmittedSubtask); err != nil {
		return err
	}
	var nextForm []byte
	switch subtask.SubtaskID {
	case "LoginEnterUserIdentifier":
		if att := strings.TrimSpace(header.Get("att")); att != "" {
			s.auth.Att = att
		}
		nextForm, err = buildTwitterLoginTextForm(resp.FlowToken, "LoginEnterUserIdentifier", strings.TrimSpace(s.account.Name))
	case "LoginEnterPassword":
		nextForm, err = buildTwitterLoginPasswordForm(resp.FlowToken, strings.TrimSpace(s.account.Password))
	case "LoginEnterAlternateIdentifierSubtask":
		email := strings.TrimSpace(s.account.Email)
		if email == "" {
			return newSocialExecutionError(SocialExecutionFailureChallengeRequired, "twitter requested alternate identifier; email is required", nil)
		}
		nextForm, err = buildTwitterLoginTextForm(resp.FlowToken, "LoginEnterAlternateIdentifierSubtask", email)
	case "LoginSuccessSubtask":
		if subtask.OpenAccount == nil || strings.TrimSpace(subtask.OpenAccount.OauthToken) == "" || strings.TrimSpace(subtask.OpenAccount.OauthTokenSecret) == "" {
			return newSocialExecutionError(SocialExecutionFailureAuthInvalid, "twitter login did not return OAuth credentials", nil)
		}
		s.auth.AccessToken = strings.TrimSpace(subtask.OpenAccount.OauthToken)
		s.auth.TokenSecret = strings.TrimSpace(subtask.OpenAccount.OauthTokenSecret)
		s.auth.Kdt = strings.TrimSpace(subtask.OpenAccount.KnownDeviceToken)
		nextForm, err = buildTwitterLoginSuccessForm(resp.FlowToken)
	case "LoginTwoFactorAuthChallenge":
		code, codeErr := s.generateTwoFactorCode()
		if codeErr != nil {
			return codeErr
		}
		nextForm, err = buildTwitterLoginTextForm(resp.FlowToken, "LoginTwoFactorAuthChallenge", code)
	case "LoginAcid":
		code, codeErr := s.resolveEmailVerificationCode(ctx)
		if codeErr != nil {
			return codeErr
		}
		nextForm, err = buildTwitterLoginTextForm(resp.FlowToken, "LoginAcid", code)
	default:
		slog.Warn("unknown twitter login subtask", "subtask_id", subtask.SubtaskID)
		return newSocialExecutionError(SocialExecutionFailurePlatform, fmt.Sprintf("unknown twitter login step: %s", subtask.SubtaskID), nil)
	}
	if err != nil {
		return err
	}
	return s.fetchOnboardingTasks(ctx, nextForm, depth+1, subtask.SubtaskID)
}

func twitterLoginResponseFailure(resp *twitterLoginResponse, lastSubmittedSubtask string) error {
	if resp == nil {
		return nil
	}
	message := twitterLoginResponseErrorMessage(resp, twitterLoginSubtask{})
	if kind, safeMessage, ok := twitterLoginFailureFromMessage(message); ok {
		return newSocialExecutionError(kind, safeMessage, nil)
	}
	if strings.EqualFold(lastSubmittedSubtask, "LoginEnterPassword") && strings.TrimSpace(resp.RawBody) != "" {
		if twitterLoginMessageIndicatesPasswordInvalid(resp.RawBody) {
			return newSocialExecutionError(SocialExecutionFailurePasswordInvalid, "twitter password is incorrect", nil)
		}
	}
	return nil
}

func twitterLoginSubtaskFailure(resp *twitterLoginResponse, subtask twitterLoginSubtask, lastSubmittedSubtask string) error {
	if strings.EqualFold(subtask.SubtaskID, "LoginEnterPassword") && strings.EqualFold(lastSubmittedSubtask, "LoginEnterPassword") {
		return newSocialExecutionError(SocialExecutionFailurePasswordInvalid, "twitter password is incorrect", nil)
	}
	message := twitterLoginResponseErrorMessage(resp, subtask)
	if kind, safeMessage, ok := twitterLoginFailureFromMessage(message); ok {
		return newSocialExecutionError(kind, safeMessage, nil)
	}
	if resp != nil && twitterLoginMessageIndicatesPasswordInvalid(resp.RawBody) {
		return newSocialExecutionError(SocialExecutionFailurePasswordInvalid, "twitter password is incorrect", nil)
	}
	return nil
}

func twitterLoginResponseErrorMessage(resp *twitterLoginResponse, subtask twitterLoginSubtask) string {
	parts := make([]string, 0, 6)
	if resp != nil {
		for _, item := range resp.Errors {
			if trimmed := strings.TrimSpace(item.Message); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
	}
	appendInputText := func(input *twitterLoginTextInput) {
		if input == nil {
			return
		}
		for _, value := range []string{input.ErrorMessage} {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		for _, text := range []*twitterLoginText{input.PrimaryText, input.SecondaryText} {
			if text != nil {
				if trimmed := strings.TrimSpace(text.Text); trimmed != "" {
					parts = append(parts, trimmed)
				}
			}
		}
	}
	appendInputText(subtask.EnterText)
	appendInputText(subtask.EnterPassword)
	return strings.Join(parts, " ")
}

func twitterLoginFailureFromMessage(message string) (SocialExecutionFailureKind, string, bool) {
	if strings.TrimSpace(message) == "" {
		return "", "", false
	}
	if failure, ok := knownTwitterFailureDetail(message); ok {
		return failure.kind, "twitter login error: " + shortBusinessMessage(message), true
	}
	return "", "", false
}

func twitterLoginMessageIndicatesPasswordInvalid(message string) bool {
	failure, ok := knownTwitterFailureDetail(message)
	return ok && failure.kind == SocialExecutionFailurePasswordInvalid
}

func (s *twitterLoginSession) credentialResult(ctx context.Context) (*SocialAccountCredentialResult, error) {
	accountID := s.fetchViewerUserID(ctx)
	if s.auth != nil && strings.TrimSpace(s.auth.ScreenName) == "" {
		s.auth.ScreenName = normalizeTwitterExecutionScreenName(s.account.Name)
	}
	authCookie, err := json.Marshal(s.auth)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to store auth credentials", err)
	}
	// auth_cookie keeps the full login backup. execution_auth is built only from
	// access_token, token_secret, and screen_name before storage encryption.
	executionAuth, err := twitterExecutionAuthFromHeaders(s.auth)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to store execution credentials", err)
	}
	return &SocialAccountCredentialResult{
		ExecutionAuth:  executionAuth,
		AuthCookie:     string(authCookie),
		PlatformUserID: accountID,
		Message:        "login succeeded",
	}, nil
}

func (s *twitterLoginSession) fetchGuestToken(ctx context.Context) error {
	createRequest := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoints.guestToken, nil)
		if err != nil {
			return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter login request", err)
		}
		s.setGuestTokenHeaders(req)
		return req, nil
	}
	resp, body, err := s.doWithRetry(ctx, createRequest)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return newSocialExecutionError(twitterFailureKind(&twitterActionResult{StatusCode: resp.StatusCode, Message: twitterErrorMessage(body, resp.StatusCode)}), twitterErrorMessage(body, resp.StatusCode), nil)
	}
	var parsed struct {
		GuestToken string `json:"guest_token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || strings.TrimSpace(parsed.GuestToken) == "" {
		return newSocialExecutionError(SocialExecutionFailurePlatform, "failed to get twitter guest token", err)
	}
	s.auth.GuestToken = parsed.GuestToken
	return nil
}

func (s *twitterLoginSession) fetchOnboardingTask(ctx context.Context, formData []byte, first bool) (*twitterLoginResponse, http.Header, error) {
	endpoint := s.endpoints.onboarding
	if first {
		endpoint = s.endpoints.onboardingFirst
	}
	createRequest := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(formData))
		if err != nil {
			return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter login request", err)
		}
		s.setOnboardingHeaders(req)
		return req, nil
	}
	resp, body, err := s.doWithRetry(ctx, createRequest)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		message := twitterErrorMessage(body, resp.StatusCode)
		return nil, nil, newSocialExecutionError(twitterFailureKind(&twitterActionResult{StatusCode: resp.StatusCode, Message: message}), message, nil)
	}
	var parsed twitterLoginResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to parse twitter login response", err)
	}
	parsed.RawBody = string(body)
	return &parsed, resp.Header, nil
}

func (s *twitterLoginSession) generateTwoFactorCode() (string, error) {
	secret := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s.account.TwoFactor), " ", ""))
	if secret == "" {
		return "", newSocialExecutionError(SocialExecutionFailureChallengeRequired, "twitter two-factor challenge requires a 2FA secret", nil)
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		decoded, err = base32.StdEncoding.DecodeString(secret)
	}
	if err != nil {
		return "", newSocialExecutionError(SocialExecutionFailureAuthInvalid, "twitter two-factor secret is invalid", err)
	}
	now := time.Now
	if s != nil && s.now != nil {
		now = s.now
	}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(now().Unix()/30))
	hash := hmac.New(sha1.New, decoded)
	_, _ = hash.Write(counter[:])
	sum := hash.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)
	return fmt.Sprintf("%06d", code%1000000), nil
}

func (s *twitterLoginSession) resolveEmailVerificationCode(ctx context.Context) (string, error) {
	if s == nil || s.emailCodeResolver == nil {
		return "", newSocialExecutionError(SocialExecutionFailureChallengeRequired, "twitter email verification challenge requires an email code resolver", nil)
	}
	code, err := s.emailCodeResolver(ctx, s.account)
	if err != nil {
		return "", newSocialExecutionError(SocialExecutionFailureChallengeRequired, "twitter email verification challenge failed", err)
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return "", newSocialExecutionError(SocialExecutionFailureChallengeRequired, "twitter email verification code is empty", nil)
	}
	return code, nil
}

func (s *twitterLoginSession) fetchViewerUserID(ctx context.Context) *string {
	createRequest := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoints.viewerUser, nil)
		if err != nil {
			return nil, err
		}
		s.setOnboardingHeaders(req)
		return req, nil
	}
	resp, body, err := s.doWithRetry(ctx, createRequest)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	var parsed struct {
		Data struct {
			Viewer struct {
				UserResult struct {
					Result struct {
						RestID string `json:"rest_id"`
					} `json:"result"`
				} `json:"userResult"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}
	restID := strings.TrimSpace(parsed.Data.Viewer.UserResult.Result.RestID)
	if restID == "" {
		return nil
	}
	return &restID
}

func (s *twitterLoginSession) doWithRetry(ctx context.Context, createRequest func() (*http.Request, error)) (*http.Response, []byte, error) {
	var lastErr error
	for attempt := 0; attempt <= twitterLoginNetworkMaxRetries; attempt++ {
		if attempt > 0 {
			if err := waitTwitterLoginRetryBackoff(ctx, attempt); err != nil {
				return nil, nil, newSocialExecutionError(SocialExecutionFailureNetwork, "twitter login network request cancelled", err)
			}
		}
		req, err := createRequest()
		if err != nil {
			return nil, nil, err
		}
		resp, body, err := s.doOnce(req)
		if err == nil {
			return resp, body, nil
		}
		lastErr = err
		if kind, ok := socialExecutionFailureKind(err); ok && kind != SocialExecutionFailureNetwork {
			return nil, nil, err
		}
		if !isRetryableTwitterNetworkError(err) || attempt == twitterLoginNetworkMaxRetries {
			break
		}
		slog.Warn(
			"twitter login network request retrying",
			"attempt", attempt+1,
			"max_retries", twitterLoginNetworkMaxRetries,
			"method", req.Method,
			"host", twitterRequestHost(req),
			"error", err,
		)
	}
	return nil, nil, twitterLoginNetworkExecutionError(lastErr)
}

func (s *twitterLoginSession) doOnce(req *http.Request) (*http.Response, []byte, error) {
	if s == nil || s.httpClient == nil {
		return nil, nil, newSocialExecutionError(SocialExecutionFailurePlatform, "twitter http client is unavailable", nil)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if s.onHTTPResponse != nil {
		s.onHTTPResponse()
	}
	defer resp.Body.Close()
	body, err := readTwitterResponseBody(resp)
	if err != nil {
		return nil, nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to read twitter response", err)
	}
	return resp, body, nil
}

func waitTwitterLoginRetryBackoff(ctx context.Context, attempt int) error {
	backoff := twitterNetworkRetryBackoffDuration(attempt)
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryableTwitterNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.IsTemporary || dnsErr.IsTimeout
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func twitterLoginNetworkExecutionError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return newSocialExecutionError(SocialExecutionFailureNetwork, "twitter login network request timed out", err)
	}
	return newSocialExecutionError(SocialExecutionFailureNetwork, "twitter login network request failed", err)
}

func (s *twitterLoginSession) setGuestTokenHeaders(req *http.Request) {
	s.setBaseHeaders(req, false)
	req.Header.Set("content-type", "text/plain; charset=ISO-8859-1")
}

func (s *twitterLoginSession) setOnboardingHeaders(req *http.Request) {
	s.setBaseHeaders(req, true)
	req.Header.Set("content-type", "application/json")
	if s.auth.Att != "" {
		req.Header.Set("att", s.auth.Att)
	}
	if s.auth.AccessToken != "" && s.auth.TokenSecret != "" {
		req.Header.Set("authorization", generateTwitterOAuthHeader(req.Method, req.URL.String(), nil, s.auth.AccessToken, s.auth.TokenSecret))
		if s.auth.Kdt != "" {
			req.Header.Set("kdt", s.auth.Kdt)
		}
		return
	}
	if s.auth.GuestToken != "" {
		req.Header.Set("x-guest-token", s.auth.GuestToken)
	}
}

func (s *twitterLoginSession) setBaseHeaders(req *http.Request, active bool) {
	h := s.auth
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-encoding", safeTwitterAcceptEncoding(h.AcceptEncoding))
	req.Header.Set("accept-language", h.AcceptLanguage)
	req.Header.Set("authorization", h.Authorization)
	req.Header.Set("cache-control", "no-store")
	req.Header.Set("optimize-body", "true")
	req.Header.Set("os-version", h.OsVersion)
	req.Header.Set("os-security-patch-level", h.OsSecurityPatchLevel)
	req.Header.Set("timezone", h.Timezone)
	req.Header.Set("user-agent", h.UserAgent)
	h.TraceId = generateTwitterTraceID()
	req.Header.Set("x-b3-traceid", h.TraceId)
	req.Header.Set("x-client-uuid", h.ClientUuid)
	req.Header.Set("x-twitter-active-user", twitterActiveUserHeader(active))
	req.Header.Set("x-twitter-api-version", h.TwitterApiVersion)
	req.Header.Set("x-twitter-client", h.TwitterClient)
	req.Header.Set("x-twitter-client-deviceid", h.ClientDeviceId)
	req.Header.Set("x-twitter-client-flavor", "")
	req.Header.Set("x-twitter-client-language", h.ClientLanguage)
	req.Header.Set("x-twitter-client-limit-ad-tracking", h.ClientLimitAdTracking)
	req.Header.Set("x-twitter-client-version", h.ClientVersion)
	req.Header.Set("twitter-display-size", h.TwitterDisplaySize)
	if h.AttestToken != "" {
		req.Header.Set("x-attest-token", h.AttestToken)
	}
	if h.ClientAppsetId != "" {
		req.Header.Set("x-twitter-client-appsetid", h.ClientAppsetId)
	}
	if h.ClientAdid != "" {
		req.Header.Set("x-twitter-client-adid", h.ClientAdid)
	}
	if h.SystemUserAgent != "" {
		req.Header.Set("system-user-agent", h.SystemUserAgent)
	}
}

func twitterActiveUserHeader(active bool) string {
	if active {
		return "yes"
	}
	return "no"
}

func buildTwitterLoginInitialForm() ([]byte, error) {
	return json.Marshal(map[string]any{
		"input_flow_data": map[string]any{
			"country_code": nil,
			"flow_context": map[string]any{
				"referrer_context": map[string]any{
					"referral_details": "utm_source=google-play&utm_medium=organic",
					"referrer_url":     "",
				},
				"start_location": map[string]any{"location": "deeplink"},
			},
			"requested_variant": nil,
			"target_user_id":    0,
		},
		"subtask_versions": twitterLoginSubtaskVersions(),
	})
}

func buildTwitterLoginTextForm(flowToken, subtaskID, text string) ([]byte, error) {
	return buildTwitterLoginSubtaskForm(flowToken, []map[string]any{{
		"subtask_id": subtaskID,
		"enter_text": map[string]any{
			"challenge_response": nil,
			"suggestion_id":      nil,
			"text":               text,
			"link":               "next_link",
		},
	}})
}

func buildTwitterLoginPasswordForm(flowToken, password string) ([]byte, error) {
	return buildTwitterLoginSubtaskForm(flowToken, []map[string]any{{
		"subtask_id": "LoginEnterPassword",
		"enter_password": map[string]any{
			"password": password,
			"link":     "next_link",
		},
	}})
}

func buildTwitterLoginSuccessForm(flowToken string) ([]byte, error) {
	return buildTwitterLoginSubtaskForm(flowToken, []map[string]any{
		{"subtask_id": "LoginSuccessSubtask", "open_account": map[string]any{"link": "next_link"}},
		{"subtask_id": "SuccessExit", "open_link": map[string]any{"link": "next_link"}},
		{"subtask_id": "LoginOpenHomeTimeline", "open_home_timeline": map[string]any{"link": "next_link"}},
	})
}

func buildTwitterLoginSubtaskForm(flowToken string, inputs []map[string]any) ([]byte, error) {
	return json.Marshal(map[string]any{
		"flow_token":       flowToken,
		"subtask_inputs":   inputs,
		"subtask_versions": twitterLoginSubtaskVersions(),
	})
}

func twitterLoginSubtaskVersions() map[string]int {
	return map[string]int{
		"generic_urt": 3, "standard": 1, "open_home_timeline": 1, "app_locale_update": 1,
		"enter_date": 1, "email_verification": 3, "deregister_device": 1, "enter_password": 5,
		"enter_text": 6, "one_tap": 2, "cta": 7, "single_sign_on": 1,
		"fetch_persisted_data": 1, "enter_username": 3, "web_modal": 2, "fetch_temporary_password": 1,
		"menu_dialog": 1, "sign_up_review": 5, "user_recommendations_urt": 3, "in_app_notification": 1,
		"sign_up": 2, "typeahead_search": 1, "app_attestation": 1, "user_recommendations_list": 4,
		"contacts_live_sync_permission_prompt": 3, "choice_selection": 5, "js_instrumentation": 1,
		"alert_dialog_suppress_client_events": 1, "privacy_options": 1, "topics_selector": 1,
		"wait_spinner": 3, "tweet_selection_urt": 1, "end_flow": 1, "settings_list": 7,
		"open_external_link": 1, "phone_verification": 5, "security_key": 3, "select_banner": 2,
		"upload_media": 1, "web": 2, "alert_dialog": 1, "open_account": 2,
		"passkey": 1, "action_list": 2, "enter_phone": 2, "open_link": 1,
		"show_code": 1, "update_users": 1, "check_logged_in_account": 1, "enter_email": 2,
		"select_avatar": 4, "location_permission_prompt": 2, "notifications_permission_prompt": 4,
	}
}
