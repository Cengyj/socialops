package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	cryptorand "crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/pkg/httpclient"
	"github.com/Wei-Shaw/socialops/internal/pkg/proxyurl"
)

const (
	twitterOAuthConsumerKey    = "3nVuSoBZnx6U4vzUxf5w"
	twitterOAuthConsumerSecret = "Bcs59EFbbsdF6Sl9Ng71smgStWEGwXXKSjYvPVt7qys"
)

type twitterHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type twitterEndpoints struct {
	createFriendship string
	favoriteTweet    string
	createTweet      string
	createRetweet    string
	viewerUser       string
}

func defaultTwitterEndpoints() twitterEndpoints {
	return twitterEndpoints{
		createFriendship: "https://api.twitter.com/1.1/friendships/create.json",
		favoriteTweet:    "https://api.twitter.com/graphql/lI07N6Otwv1PhnEgXILM7A/FavoriteTweet",
		createTweet:      "https://api.twitter.com/graphql/zjas7pv0yD5Kfn4iUEJU3w/CreateTweet",
		createRetweet:    "https://api.twitter.com/graphql/yG9lllv3WyBS63TA3t0D7Q/CreateRetweet",
		viewerUser:       "https://api.twitter.com/graphql/-gEcxCUhgJqG0YcSq-ztbg/ViewerUserQuery?variables=%7B%22includeTweetImpression%22%3Atrue%2C%22include_profile_info%22%3Atrue%2C%22includeHasBirdwatchNotes%22%3Afalse%2C%22includeEditPerspective%22%3Afalse%2C%22includeEditControl%22%3Atrue%7D&features=%7B%22profile_label_improvements_pcf_label_in_profile_enabled%22%3Atrue%2C%22super_follow_badge_privacy_enabled%22%3Atrue%2C%22graduated_access_invisible_treatment_enabled%22%3Atrue%2C%22subscriptions_verification_info_enabled%22%3Atrue%2C%22super_follow_user_api_enabled%22%3Atrue%2C%22blue_business_profile_image_shape_enabled%22%3Atrue%2C%22immersive_video_status_linkable_timestamps%22%3Atrue%2C%22super_follow_exclusive_tweet_notifications_enabled%22%3Atrue%7D",
	}
}

// TwitterExecutor executes SocialOps Twitter/X task actions.
type TwitterExecutor struct {
	clientForProxy func(proxyURL string) (twitterHTTPClient, error)
	endpoints      twitterEndpoints
}

func NewTwitterExecutor() *TwitterExecutor {
	return &TwitterExecutor{
		clientForProxy: defaultTwitterHTTPClient,
		endpoints:      defaultTwitterEndpoints(),
	}
}

func defaultTwitterHTTPClient(proxyURL string) (twitterHTTPClient, error) {
	return httpclient.GetClient(httpclient.Options{
		ProxyURL:              proxyURL,
		Timeout:               30 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
	})
}

func (e *TwitterExecutor) Execute(ctx context.Context, taskLog *dbent.SocialTaskLog, account *dbent.SocialAccount) (string, error) {
	if taskLog == nil || account == nil {
		return "", newSocialExecutionError(SocialExecutionFailureActionInput, "twitter task input is unavailable", nil)
	}
	auth, err := twitterAuthHeadersFromAccount(account)
	if err != nil {
		slog.Warn("twitter task auth unavailable", "task_log_id", taskLog.ID, "account_id", account.ID, "error", err)
		return "", err
	}
	proxyEndpoint, err := twitterProxyEndpointFromTask(taskLog)
	if err != nil {
		slog.Warn("twitter task proxy unavailable", "task_log_id", taskLog.ID, "account_id", account.ID, "proxy_id", taskLog.ProxyID, "error", err)
		return "", err
	}

	clientFactory := e.clientForProxy
	if clientFactory == nil {
		clientFactory = defaultTwitterHTTPClient
	}
	httpClient, err := clientFactory(proxyEndpoint)
	if err != nil {
		slog.Warn("twitter task http client build failed", "task_log_id", taskLog.ID, "account_id", account.ID, "proxy_id", taskLog.ProxyID, "error", err)
		return "", newSocialExecutionError(SocialExecutionFailureProxyInvalid, "execution proxy is invalid", err)
	}

	endpoints := e.endpoints
	if endpoints.createFriendship == "" {
		endpoints = defaultTwitterEndpoints()
	}
	api := &twitterAPIClient{httpClient: httpClient, auth: auth, endpoints: endpoints}

	slog.Info("twitter task execution started", "task_log_id", taskLog.ID, "account_id", account.ID, "action", taskLog.Action, "proxy_id", taskLog.ProxyID)
	var result *twitterActionResult
	switch taskLog.Action {
	case SocialTaskActionLoginCheck:
		result, err = api.loginCheck(ctx)
	case SocialTaskActionFollow:
		userID, parseErr := twitterNumericIDFromTarget(stringPtrValue(taskLog.Target), "user")
		if parseErr != nil {
			return "", parseErr
		}
		result, err = api.followUser(ctx, userID)
	case SocialTaskActionLike:
		tweetID, parseErr := twitterNumericIDFromTarget(stringPtrValue(taskLog.Target), "tweet")
		if parseErr != nil {
			return "", parseErr
		}
		result, err = api.favoriteTweet(ctx, tweetID)
	case SocialTaskActionPost:
		content := strings.TrimSpace(stringPtrValue(taskLog.Content))
		if content == "" {
			return "", newSocialExecutionError(SocialExecutionFailureActionInput, "post content is required", nil)
		}
		result, err = api.createTweet(ctx, content)
	case SocialTaskActionRetweet:
		tweetID, parseErr := twitterNumericIDFromTarget(stringPtrValue(taskLog.Target), "tweet")
		if parseErr != nil {
			return "", parseErr
		}
		result, err = api.retweet(ctx, tweetID)
	default:
		return "", newSocialExecutionError(SocialExecutionFailureUnsupported, fmt.Sprintf("%s is not configured: twitter executor does not support this action", taskLog.Action), nil)
	}
	if err != nil {
		slog.Warn("twitter task request failed", "task_log_id", taskLog.ID, "account_id", account.ID, "action", taskLog.Action, "error", err)
		return "", err
	}
	if result == nil {
		return "", newSocialExecutionError(SocialExecutionFailurePlatform, "twitter action returned no result", nil)
	}
	if !result.Success {
		slog.Warn(
			"twitter task api failed",
			"task_log_id", taskLog.ID,
			"account_id", account.ID,
			"action", taskLog.Action,
			"status_code", result.StatusCode,
			"message", result.Message,
			"response_preview", truncateForLog(result.RawBody, 512),
		)
		return "", newSocialExecutionError(twitterFailureKind(result), result.Message, nil)
	}
	slog.Info("twitter task execution succeeded", "task_log_id", taskLog.ID, "account_id", account.ID, "action", taskLog.Action, "status_code", result.StatusCode)
	return result.Message, nil
}

type twitterAuthHeaders struct {
	TwitterApiVersion     string `json:"twitter_api_version"`
	TwitterClient         string `json:"twitter_client"`
	ClientVersion         string `json:"client_version"`
	ClientLanguage        string `json:"client_language"`
	ClientUuid            string `json:"client_uuid"`
	ClientDeviceId        string `json:"client_device_id"`
	ClientAppsetId        string `json:"client_appset_id"`
	ClientAdid            string `json:"client_adid"`
	ClientLimitAdTracking string `json:"client_limit_ad_tracking"`
	TwitterDisplaySize    string `json:"twitter_display_size"`
	UserAgent             string `json:"user_agent"`
	SystemUserAgent       string `json:"system_user_agent"`
	OsVersion             string `json:"os_version"`
	OsSecurityPatchLevel  string `json:"os_security_patch_level"`
	Timezone              string `json:"timezone"`
	AcceptEncoding        string `json:"accept_encoding"`
	AcceptLanguage        string `json:"accept_language"`
	TraceId               string `json:"trace_id"`
	Authorization         string `json:"authorization"`
	GuestToken            string `json:"guest_token"`
	AccessToken           string `json:"access_token"`
	TokenSecret           string `json:"token_secret"`
	AttestToken           string `json:"attest_token"`
	Att                   string `json:"att"`
	Kdt                   string `json:"kdt"`
	ScreenName            string `json:"screen_name"`
}

func defaultTwitterAuthHeaders() *twitterAuthHeaders {
	return &twitterAuthHeaders{
		TwitterApiVersion:     "5",
		UserAgent:             "TwitterAndroid/11.46.0-release.0 (311460000-r-0) Pixel+6/15 (Google;Pixel+6;google;oriole;0;;1;2016)",
		Authorization:         "Bearer AAAAAAAAAAAAAAAAAAAAAFXzAwAAAAAAMHCxpeSDG1gLNLghVe8d74hl6k4%3DRUMF4xAQLsbeBhTSRrCiQpJtxoGWeyHrDb5te2jpGskWDFW82F",
		TwitterClient:         "TwitterAndroid",
		ClientLanguage:        "en-US",
		ClientLimitAdTracking: "0",
		ClientVersion:         "11.46.0-release.0",
		ClientDeviceId:        "ab97d51879dcfdbd",
		AcceptEncoding:        "gzip",
		AcceptLanguage:        "en-US",
		OsVersion:             "35",
		OsSecurityPatchLevel:  "2024-10-05",
		Timezone:              "Pacific/Honolulu",
		SystemUserAgent:       "Dalvik/2.1.0 (Linux; U; Android 15; Pixel 6 Build/BP1A.250505.005)",
		TwitterDisplaySize:    "1080x2400x420",
	}
}

func twitterAuthHeadersFromAccount(account *dbent.SocialAccount) (*twitterAuthHeaders, error) {
	raw := ""
	if account != nil {
		raw = trimPtr(account.ExecutionAuth)
	}
	if raw == "" {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthMissing, "account execution auth is required", nil)
	}
	if !strings.HasPrefix(raw, "{") {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err == nil && strings.HasPrefix(strings.TrimSpace(string(decoded)), "{") {
			raw = strings.TrimSpace(string(decoded))
		}
	}
	headers := defaultTwitterAuthHeaders()
	if err := json.Unmarshal([]byte(raw), headers); err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthInvalid, "account execution auth is invalid", err)
	}
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

func twitterProxyEndpointFromTask(taskLog *dbent.SocialTaskLog) (string, error) {
	if taskLog == nil || taskLog.ProxySnapshot == nil || strings.TrimSpace(*taskLog.ProxySnapshot) == "" {
		return "", newSocialExecutionError(SocialExecutionFailureProxyMissing, "execution proxy is required", nil)
	}
	var snapshot struct {
		Endpoint string `json:"endpoint"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal([]byte(*taskLog.ProxySnapshot), &snapshot); err != nil {
		return "", newSocialExecutionError(SocialExecutionFailureProxyInvalid, "execution proxy snapshot is invalid", err)
	}
	if strings.TrimSpace(snapshot.Endpoint) == "" {
		return "", newSocialExecutionError(SocialExecutionFailureProxyMissing, "execution proxy endpoint is required", nil)
	}
	if snapshot.Status != "" && snapshot.Status != SocialIPStatusOnline {
		return "", newSocialExecutionError(SocialExecutionFailureProxyUnavailable, "execution proxy is not online", nil)
	}
	trimmed, _, err := proxyurl.Parse(snapshot.Endpoint)
	if err != nil {
		return "", newSocialExecutionError(SocialExecutionFailureProxyInvalid, "execution proxy is invalid", err)
	}
	return trimmed, nil
}

type twitterAPIClient struct {
	httpClient twitterHTTPClient
	auth       *twitterAuthHeaders
	endpoints  twitterEndpoints
}

type twitterActionResult struct {
	Success    bool
	Message    string
	StatusCode int
	RawBody    string
}

func (c *twitterAPIClient) loginCheck(ctx context.Context) (*twitterActionResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoints.viewerUser, nil)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter request", err)
	}
	c.setGraphQLHeaders(req)
	resp, body, err := c.do(req)
	if err != nil {
		return nil, err
	}
	result := &twitterActionResult{StatusCode: resp.StatusCode, RawBody: string(body)}
	if resp.StatusCode != http.StatusOK {
		result.Message = twitterErrorMessage(body, resp.StatusCode)
		return result, nil
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
	if err := json.Unmarshal(body, &parsed); err != nil || strings.TrimSpace(parsed.Data.Viewer.UserResult.Result.RestID) == "" {
		result.Message = "authentication failed"
		return result, nil
	}
	result.Success = true
	result.Message = "login check succeeded"
	return result, nil
}

func (c *twitterAPIClient) followUser(ctx context.Context, userID string) (*twitterActionResult, error) {
	form := url.Values{}
	form.Set("ext", "mediaRestrictions,altText,mediaStats,mediaColor,info360,highlightedLabel,unmentionInfo,editControl,previousCounts,limitedActionResults,superFollowMetadata")
	form.Set("send_error_codes", "true")
	form.Set("user_id", userID)
	form.Set("handles_challenges", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoints.createFriendship, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter request", err)
	}
	c.setFormHeaders(req, form)
	return c.sendSimple(req, "follow succeeded")
}

func (c *twitterAPIClient) favoriteTweet(ctx context.Context, tweetID string) (*twitterActionResult, error) {
	variables := fmt.Sprintf(`{"includeTweetImpression":true,"includeHasBirdwatchNotes":false,"includeEditPerspective":false,"tweet_id":%s,"includeEditControl":true}`, tweetID)
	return c.sendGraphQL(ctx, c.endpoints.favoriteTweet, map[string]string{"variables": variables}, "like succeeded")
}

func (c *twitterAPIClient) createTweet(ctx context.Context, text string) (*twitterActionResult, error) {
	variables := map[string]any{
		"nullcast":                          false,
		"includeTweetImpression":            true,
		"includeHasBirdwatchNotes":          false,
		"includeEditPerspective":            false,
		"includeEditControl":                true,
		"includeCommunityTweetRelationship": false,
		"includeTweetVisibilityNudge":       true,
		"tweet_text":                        text,
	}
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "post content is invalid", err)
	}
	return c.sendGraphQL(ctx, c.endpoints.createTweet, map[string]string{
		"features":  twitterCreateTweetFeatures,
		"variables": string(variablesJSON),
	}, "post succeeded")
}

func (c *twitterAPIClient) retweet(ctx context.Context, tweetID string) (*twitterActionResult, error) {
	variables := fmt.Sprintf(`{"includeTweetImpression":true,"includeHasBirdwatchNotes":false,"includeEditPerspective":false,"tweet_id":"%s","includeEditControl":true,"includeTweetVisibilityNudge":true}`, tweetID)
	return c.sendGraphQL(ctx, c.endpoints.createRetweet, map[string]string{
		"features":  twitterCreateTweetFeatures,
		"variables": variables,
	}, "retweet succeeded")
}

func (c *twitterAPIClient) sendGraphQL(ctx context.Context, endpoint string, payload map[string]string, successMessage string) (*twitterActionResult, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to build twitter request", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter request", err)
	}
	c.setGraphQLHeaders(req)
	return c.sendSimple(req, successMessage)
}

func (c *twitterAPIClient) sendSimple(req *http.Request, successMessage string) (*twitterActionResult, error) {
	resp, body, err := c.do(req)
	if err != nil {
		return nil, err
	}
	result := &twitterActionResult{StatusCode: resp.StatusCode, RawBody: string(body)}
	if resp.StatusCode == http.StatusOK {
		result.Success = true
		result.Message = successMessage
		return result, nil
	}
	result.Message = twitterErrorMessage(body, resp.StatusCode)
	return result, nil
}

func (c *twitterAPIClient) do(req *http.Request) (*http.Response, []byte, error) {
	if c == nil || c.httpClient == nil {
		return nil, nil, newSocialExecutionError(SocialExecutionFailurePlatform, "twitter http client is unavailable", nil)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, newSocialExecutionError(SocialExecutionFailureNetwork, "network request failed; check execution proxy", err)
	}
	defer resp.Body.Close()
	body, err := readTwitterResponseBody(resp)
	if err != nil {
		return nil, nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to read twitter response", err)
	}
	return resp, body, nil
}

func (c *twitterAPIClient) setFormHeaders(req *http.Request, form url.Values) {
	c.setCommonHeaders(req, "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", generateTwitterOAuthHeader(req.Method, req.URL.String(), form, c.auth.AccessToken, c.auth.TokenSecret))
}

func (c *twitterAPIClient) setGraphQLHeaders(req *http.Request) {
	c.setCommonHeaders(req, "application/json")
	req.Header.Set("Authorization", generateTwitterOAuthHeader(req.Method, req.URL.String(), nil, c.auth.AccessToken, c.auth.TokenSecret))
}

func (c *twitterAPIClient) setCommonHeaders(req *http.Request, contentType string) {
	h := c.auth
	if h.TraceId == "" {
		h.TraceId = generateTwitterTraceID()
	}
	req.Host = req.URL.Host
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", h.UserAgent)
	req.Header.Set("X-Twitter-Client", h.TwitterClient)
	req.Header.Set("X-Twitter-Client-Version", h.ClientVersion)
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-API-Version", h.TwitterApiVersion)
	req.Header.Set("X-Twitter-Client-Language", h.ClientLanguage)
	req.Header.Set("X-Twitter-Client-DeviceId", h.ClientDeviceId)
	req.Header.Set("X-Client-UUID", h.ClientUuid)
	req.Header.Set("Timezone", h.Timezone)
	req.Header.Set("OS-Security-Patch-Level", h.OsSecurityPatchLevel)
	req.Header.Set("Accept-Encoding", safeTwitterAcceptEncoding(h.AcceptEncoding))
	req.Header.Set("Accept-Language", h.AcceptLanguage)
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set("Optimize-Body", "true")
	req.Header.Set("X-B3-TraceId", h.TraceId)
	req.Header.Set("X-Twitter-Client-Limit-Ad-Tracking", h.ClientLimitAdTracking)
	if h.AttestToken != "" {
		req.Header.Set("X-Attest-Token", h.AttestToken)
	}
	if h.Kdt != "" {
		req.Header.Set("Kdt", h.Kdt)
	}
	if h.ClientAdid != "" {
		req.Header.Set("X-Twitter-Client-Adid", h.ClientAdid)
	}
	if h.ClientAppsetId != "" {
		req.Header.Set("X-Twitter-Client-Appsetid", h.ClientAppsetId)
	}
	if h.SystemUserAgent != "" {
		req.Header.Set("System-User-Agent", h.SystemUserAgent)
	}
	if h.TwitterDisplaySize != "" {
		req.Header.Set("Twitter-Display-Size", h.TwitterDisplaySize)
	}
	if h.Att != "" {
		req.Header.Set("Att", h.Att)
	}
}

func readTwitterResponseBody(resp *http.Response) ([]byte, error) {
	var reader io.ReadCloser
	switch strings.ToLower(resp.Header.Get("Content-Encoding")) {
	case "gzip":
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		reader = gzipReader
		defer reader.Close()
	default:
		reader = resp.Body
	}
	return io.ReadAll(reader)
}

func generateTwitterOAuthHeader(method, urlStr string, extra url.Values, token, tokenSecret string) string {
	nonce := generateTwitterOAuthNonce()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	oauthParams := map[string]string{
		"oauth_consumer_key":     twitterOAuthConsumerKey,
		"oauth_nonce":            nonce,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_token":            token,
		"oauth_version":          "1.0",
	}
	signatureBase := twitterOAuthSignatureBase(method, urlStr, extra, oauthParams)
	signature := twitterOAuthSignature(signatureBase, twitterOAuthConsumerSecret+"&"+tokenSecret)
	return fmt.Sprintf(
		`OAuth realm="http://api.twitter.com/", oauth_version="1.0", oauth_token="%s", oauth_nonce="%s", oauth_timestamp="%s", oauth_signature="%s", oauth_consumer_key="%s", oauth_signature_method="HMAC-SHA1"`,
		token,
		nonce,
		timestamp,
		twitterPercentEncode(signature),
		twitterOAuthConsumerKey,
	)
}

func twitterOAuthSignatureBase(method, urlStr string, extra url.Values, oauthParams map[string]string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return strings.ToUpper(method) + "&&"
	}
	baseURL := parsed.Scheme + "://" + parsed.Host + parsed.Path
	params := make([][2]string, 0, len(parsed.Query())+len(extra)+len(oauthParams))
	for key, values := range parsed.Query() {
		for _, value := range values {
			params = append(params, [2]string{key, value})
		}
	}
	for key, values := range extra {
		for _, value := range values {
			params = append(params, [2]string{key, value})
		}
	}
	for key, value := range oauthParams {
		params = append(params, [2]string{key, value})
	}
	sort.Slice(params, func(i, j int) bool {
		ik, jk := twitterPercentEncode(params[i][0]), twitterPercentEncode(params[j][0])
		if ik == jk {
			return twitterPercentEncode(params[i][1]) < twitterPercentEncode(params[j][1])
		}
		return ik < jk
	})
	encoded := make([]string, 0, len(params))
	for _, param := range params {
		encoded = append(encoded, twitterPercentEncode(param[0])+"="+twitterPercentEncode(param[1]))
	}
	return strings.ToUpper(method) + "&" + twitterPercentEncode(baseURL) + "&" + twitterPercentEncode(strings.Join(encoded, "&"))
}

func twitterOAuthSignature(base, signingKey string) string {
	h := hmac.New(sha1.New, []byte(signingKey))
	_, _ = h.Write([]byte(base))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func twitterPercentEncode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
		} else {
			b.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return b.String()
}

func generateTwitterOAuthNonce() string {
	var buf [8]byte
	if _, err := cryptorand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	random := int64(binary.BigEndian.Uint64(buf[:]) & 0x7fffffffffffffff)
	return fmt.Sprintf("%d%d", time.Now().UnixNano()%10000000000000, random)
}

func generateTwitterTraceID() string {
	var buf [8]byte
	if _, err := cryptorand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%016x", binary.BigEndian.Uint64(buf[:]))
}

func generateTwitterClientUUID() string {
	var b [16]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		now := uint64(time.Now().UnixNano())
		binary.BigEndian.PutUint64(b[:8], now)
		binary.BigEndian.PutUint64(b[8:], ^now)
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func twitterErrorMessage(body []byte, statusCode int) string {
	var errResp struct {
		Errors []struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && len(errResp.Errors) > 0 {
		code := errResp.Errors[0].Code
		switch code {
		case 32, 89:
			return "authentication failed"
		case 64:
			return "account is suspended"
		case 88:
			return "rate limit exceeded"
		case 131:
			return "action is too frequent"
		case 139:
			return "tweet is already liked"
		case 160:
			return "user is already followed"
		case 161:
			return "follow limit reached"
		case 186:
			return "post content is too long"
		case 187:
			return "post content is duplicate"
		case 326:
			return "account is locked"
		case 327:
			return "tweet is already retweeted"
		case 385:
			return "post is restricted"
		default:
			message := strings.TrimSpace(errResp.Errors[0].Message)
			if message == "" {
				return fmt.Sprintf("twitter error %d", code)
			}
			return fmt.Sprintf("twitter error %d: %s", code, shortBusinessMessage(message))
		}
	}
	switch statusCode {
	case http.StatusUnauthorized:
		return "authentication failed"
	case http.StatusForbidden:
		return "access denied"
	case http.StatusNotFound:
		return "target not found"
	case http.StatusTooManyRequests:
		return "rate limit exceeded"
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return "twitter service is temporarily unavailable"
	default:
		return fmt.Sprintf("twitter request failed (HTTP %d)", statusCode)
	}
}

func twitterFailureKind(result *twitterActionResult) SocialExecutionFailureKind {
	if result == nil {
		return SocialExecutionFailurePlatform
	}
	message := strings.ToLower(strings.TrimSpace(result.Message))
	switch result.StatusCode {
	case http.StatusUnauthorized:
		return SocialExecutionFailureAuthInvalid
	case http.StatusForbidden:
		if strings.Contains(message, "suspended") || strings.Contains(message, "locked") || strings.Contains(message, "limit") {
			return SocialExecutionFailureAccountLimited
		}
		return SocialExecutionFailurePlatform
	case http.StatusTooManyRequests:
		return SocialExecutionFailureAccountLimited
	}
	if strings.Contains(message, "authentication failed") {
		return SocialExecutionFailureAuthInvalid
	}
	if strings.Contains(message, "suspended") ||
		strings.Contains(message, "locked") ||
		strings.Contains(message, "rate limit") ||
		strings.Contains(message, "too frequent") ||
		strings.Contains(message, "follow limit") {
		return SocialExecutionFailureAccountLimited
	}
	if strings.Contains(message, "target") ||
		strings.Contains(message, "content") ||
		strings.Contains(message, "duplicate") ||
		strings.Contains(message, "already") ||
		strings.Contains(message, "restricted") {
		return SocialExecutionFailureActionInput
	}
	return SocialExecutionFailurePlatform
}

func twitterNumericIDFromTarget(target, kind string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s target is required", kind), nil)
	}
	matches := twitterNumericIDPattern.FindAllString(target, -1)
	if len(matches) == 0 {
		if kind == "user" {
			return "", newSocialExecutionError(SocialExecutionFailureActionInput, "target must be a Twitter/X numeric user ID", nil)
		}
		return "", newSocialExecutionError(SocialExecutionFailureActionInput, "target must contain a Twitter/X tweet ID", nil)
	}
	return matches[len(matches)-1], nil
}

var twitterNumericIDPattern = regexp.MustCompile(`\d{5,}`)

func safeTwitterAcceptEncoding(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || strings.Contains(raw, "br") {
		return "gzip"
	}
	return raw
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func shortBusinessMessage(message string) string {
	message = strings.TrimSpace(message)
	if idx := strings.Index(message, " g;"); idx > 0 {
		message = strings.TrimSpace(message[:idx])
	}
	if len(message) > 160 {
		return message[:160]
	}
	return message
}

func truncateForLog(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

const twitterCreateTweetFeatures = `{"grok_translations_community_note_translation_is_enabled":false,"longform_notetweets_inline_media_enabled":true,"grok_android_analyze_trend_fetch_enabled":false,"super_follow_badge_privacy_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"super_follow_user_api_enabled":true,"super_follow_tweet_api_enabled":true,"articles_api_enabled":true,"profile_label_improvements_pcf_label_in_profile_enabled":true,"premium_content_api_read_enabled":false,"grok_translations_community_note_auto_translation_is_enabled":false,"android_graphql_skip_api_media_color_palette":true,"creator_subscriptions_tweet_preview_api_enabled":true,"freedom_of_speech_not_reach_fetch_enabled":true,"tweetypie_unmention_optimization_enabled":true,"longform_notetweets_consumption_enabled":true,"subscriptions_verification_info_enabled":true,"grok_translations_post_auto_translation_is_enabled":false,"blue_business_profile_image_shape_enabled":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"immersive_video_status_linkable_timestamps":true,"profile_label_improvements_pcf_label_in_post_enabled":true,"super_follow_exclusive_tweet_notifications_enabled":true}`
