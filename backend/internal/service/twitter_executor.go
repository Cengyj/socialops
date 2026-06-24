package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/md5"
	cryptorand "crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/pkg/httpclient"
)

const (
	twitterOAuthConsumerKey    = "3nVuSoBZnx6U4vzUxf5w"
	twitterOAuthConsumerSecret = "Bcs59EFbbsdF6Sl9Ng71smgStWEGwXXKSjYvPVt7qys"
	twitterVideoChunkBytes     = 5 * 1024 * 1024
	twitterMediaStatusMaxPolls = 30
	twitterNetworkMaxRetries   = 3
)

var twitterNetworkRetryBackoffDuration = func(attempt int) time.Duration {
	return time.Duration(1<<uint(attempt-1)) * time.Second
}

type twitterHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type twitterEndpoints struct {
	createFriendship    string
	favoriteTweet       string
	createTweet         string
	createRetweet       string
	updateProfile       string
	updateProfileImage  string
	updateProfileBanner string
	mediaUpload         string
	viewerUser          string
}

func defaultTwitterEndpoints() twitterEndpoints {
	return twitterEndpoints{
		createFriendship:    "https://api.twitter.com/1.1/friendships/create.json",
		favoriteTweet:       "https://api.twitter.com/graphql/lI07N6Otwv1PhnEgXILM7A/FavoriteTweet",
		createTweet:         "https://api.twitter.com/graphql/zjas7pv0yD5Kfn4iUEJU3w/CreateTweet",
		createRetweet:       "https://api.twitter.com/graphql/yG9lllv3WyBS63TA3t0D7Q/CreateRetweet",
		updateProfile:       "https://api.twitter.com/1.1/account/update_profile.json",
		updateProfileImage:  "https://api.twitter.com/1.1/account/update_profile_image.json",
		updateProfileBanner: "https://api.twitter.com/1.1/account/update_profile_banner.json",
		mediaUpload:         "https://upload.twitter.com/1.1/media/upload.json",
		viewerUser:          "https://api.twitter.com/graphql/-gEcxCUhgJqG0YcSq-ztbg/ViewerUserQuery?variables=%7B%22includeTweetImpression%22%3Atrue%2C%22include_profile_info%22%3Atrue%2C%22includeHasBirdwatchNotes%22%3Afalse%2C%22includeEditPerspective%22%3Afalse%2C%22includeEditControl%22%3Atrue%7D&features=%7B%22profile_label_improvements_pcf_label_in_profile_enabled%22%3Atrue%2C%22super_follow_badge_privacy_enabled%22%3Atrue%2C%22graduated_access_invisible_treatment_enabled%22%3Atrue%2C%22subscriptions_verification_info_enabled%22%3Atrue%2C%22super_follow_user_api_enabled%22%3Atrue%2C%22blue_business_profile_image_shape_enabled%22%3Atrue%2C%22immersive_video_status_linkable_timestamps%22%3Atrue%2C%22super_follow_exclusive_tweet_notifications_enabled%22%3Atrue%7D",
	}
}

func twitterEndpointsWithDefaults(overrides twitterEndpoints) twitterEndpoints {
	defaults := defaultTwitterEndpoints()
	if overrides.createFriendship != "" {
		defaults.createFriendship = overrides.createFriendship
	}
	if overrides.favoriteTweet != "" {
		defaults.favoriteTweet = overrides.favoriteTweet
	}
	if overrides.createTweet != "" {
		defaults.createTweet = overrides.createTweet
	}
	if overrides.createRetweet != "" {
		defaults.createRetweet = overrides.createRetweet
	}
	if overrides.updateProfile != "" {
		defaults.updateProfile = overrides.updateProfile
	}
	if overrides.updateProfileImage != "" {
		defaults.updateProfileImage = overrides.updateProfileImage
	}
	if overrides.updateProfileBanner != "" {
		defaults.updateProfileBanner = overrides.updateProfileBanner
	}
	if overrides.mediaUpload != "" {
		defaults.mediaUpload = overrides.mediaUpload
	}
	if overrides.viewerUser != "" {
		defaults.viewerUser = overrides.viewerUser
	}
	return defaults
}

// TwitterExecutor executes SocialOps Twitter/X task actions.
type TwitterExecutor struct {
	clientForProxy      func(proxyURL string) (twitterHTTPClient, error)
	endpoints           twitterEndpoints
	mediaResolver       SocialTaskMediaResolver
	loginRegistrar      SocialAccountCredentialRegistrar
	proxyHealthReporter func(context.Context, int64)
	credentialEncryptor ExecutionAuthEncryptor
}

func NewTwitterExecutor() *TwitterExecutor {
	return &TwitterExecutor{
		clientForProxy: defaultTwitterHTTPClient,
		endpoints:      defaultTwitterEndpoints(),
	}
}

func (e *TwitterExecutor) WithMediaResolver(resolver SocialTaskMediaResolver) *TwitterExecutor {
	if e == nil {
		return nil
	}
	e.mediaResolver = resolver
	return e
}

func (e *TwitterExecutor) WithLoginRegistrar(registrar SocialAccountCredentialRegistrar) *TwitterExecutor {
	if e == nil {
		return nil
	}
	e.loginRegistrar = registrar
	return e
}

func (e *TwitterExecutor) WithCredentialEncryptor(encryptor ExecutionAuthEncryptor) *TwitterExecutor {
	if e == nil {
		return nil
	}
	e.credentialEncryptor = encryptor
	return e
}

func (e *TwitterExecutor) WithProxyHealthReporter(reporter func(context.Context, int64)) *TwitterExecutor {
	if e == nil {
		return nil
	}
	e.proxyHealthReporter = reporter
	return e
}

func (e *TwitterExecutor) reportTaskProxyReachable(ctx context.Context, taskLog *dbent.SocialTaskLog) {
	if e == nil || e.proxyHealthReporter == nil {
		return
	}
	proxyID := socialTaskProxyID(taskLog)
	if proxyID <= 0 {
		return
	}
	e.proxyHealthReporter(ctx, proxyID)
}

func socialTaskProxyID(taskLog *dbent.SocialTaskLog) int64 {
	if taskLog == nil || taskLog.ProxyID == nil {
		return 0
	}
	return int64(*taskLog.ProxyID)
}

// Login performs a password login for the account, acquiring fresh execution
// credentials (OAuth token/secret + device fingerprint) without relying on any
// existing cookie. It forces execution through the task's bound proxy snapshot.
func (e *TwitterExecutor) Login(ctx context.Context, taskLog *dbent.SocialTaskLog, account *dbent.SocialAccount) (*SocialAccountCredentialResult, error) {
	if e == nil || taskLog == nil || account == nil {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "twitter login input is unavailable", nil)
	}
	if e.loginRegistrar == nil {
		return nil, newSocialExecutionError(SocialExecutionFailureConfiguration, "twitter login is not configured", nil)
	}
	password := strings.TrimSpace(trimPtr(account.Password))
	if password == "" {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthMissing, "account password is required to log in", nil)
	}
	proxyEndpoint, err := twitterProxyEndpointFromTask(ctx, taskLog)
	if err != nil {
		return nil, err
	}
	result, err := e.loginRegistrar.AcquireCredentials(ctx, SocialAccountCredentialRequest{
		Name:          strings.TrimSpace(account.Name),
		Password:      password,
		Email:         strings.TrimSpace(trimPtr(account.Email)),
		EmailPassword: trimPtr(account.EmailPassword),
		TwoFactor:     strings.TrimSpace(trimPtr(account.TwoFactor)),
		EmailToken:    strings.TrimSpace(trimPtr(account.EmailToken)),
		EmailClientID: strings.TrimSpace(trimPtr(account.EmailClientID)),
		ProxyID:       socialTaskProxyID(taskLog),
		ProxyEndpoint: proxyEndpoint,
	})
	if err != nil {
		return nil, err
	}
	if result == nil || strings.TrimSpace(result.ExecutionAuth) == "" {
		return nil, newSocialExecutionError(SocialExecutionFailureAuthInvalid, "twitter login did not return usable credentials", nil)
	}
	return result, nil
}

func defaultTwitterHTTPClient(proxyURL string) (twitterHTTPClient, error) {
	return httpclient.GetClient(httpclient.Options{
		ProxyURL:              proxyURL,
		Timeout:               30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DialTimeout:           30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
	})
}

func (e *TwitterExecutor) Execute(ctx context.Context, taskLog *dbent.SocialTaskLog, account *dbent.SocialAccount) (string, error) {
	if taskLog == nil || account == nil {
		return "", newSocialExecutionError(SocialExecutionFailureActionInput, "twitter task input is unavailable", nil)
	}
	prepared, err := e.prepareTwitterTask(ctx, taskLog)
	if err != nil {
		slog.Warn("twitter task payload validation failed", "task_log_id", taskLog.ID, "account_id", account.ID, "action", taskLog.Action, "error", err)
		return "", err
	}
	auth, err := e.twitterAuthHeadersFromAccount(account)
	if err != nil {
		slog.Warn("twitter task auth unavailable", "task_log_id", taskLog.ID, "account_id", account.ID, "error", err)
		return "", err
	}
	proxyEndpoint, err := twitterProxyEndpointFromTask(ctx, taskLog)
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

	endpoints := twitterEndpointsWithDefaults(e.endpoints)
	api := &twitterAPIClient{
		httpClient: httpClient,
		auth:       auth,
		endpoints:  endpoints,
		onHTTPResponse: func() {
			e.reportTaskProxyReachable(ctx, taskLog)
		},
	}

	slog.Info("twitter task execution started", "task_log_id", taskLog.ID, "account_id", account.ID, "action", taskLog.Action, "proxy_id", taskLog.ProxyID)
	var result *twitterActionResult
	switch taskLog.Action {
	case SocialTaskActionLoginCheck:
		result, err = api.loginCheck(ctx)
	case SocialTaskActionFollow:
		userID, parseErr := twitterNumericIDFromTarget(prepared.target, "user")
		if parseErr != nil {
			return "", parseErr
		}
		result, err = api.followUser(ctx, userID)
	case SocialTaskActionLike:
		tweetID, parseErr := twitterNumericIDFromTarget(prepared.target, "tweet")
		if parseErr != nil {
			return "", parseErr
		}
		result, err = api.favoriteTweet(ctx, tweetID)
	case SocialTaskActionPost:
		result, err = api.createTweet(ctx, prepared.post)
	case SocialTaskActionRetweet:
		tweetID, parseErr := twitterNumericIDFromTarget(prepared.target, "tweet")
		if parseErr != nil {
			return "", parseErr
		}
		result, err = api.retweet(ctx, tweetID)
	case SocialTaskActionUpdateProfile:
		result, err = api.updateProfile(ctx, prepared.profile)
	case SocialTaskActionUpdateAvatar:
		result, err = api.updateProfileImage(ctx, prepared.avatar)
	case SocialTaskActionUpdateBanner:
		result, err = api.updateProfileBanner(ctx, prepared.banner)
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

func (e *TwitterExecutor) twitterAuthHeadersFromAccount(account *dbent.SocialAccount) (*twitterAuthHeaders, error) {
	raw := ""
	if account != nil {
		raw = trimPtr(account.ExecutionAuth)
	}
	var encryptor ExecutionAuthEncryptor
	if e != nil {
		encryptor = e.credentialEncryptor
	}
	return twitterAuthHeadersFromStoredExecutionAuth(raw, encryptor)
}

func twitterProxyEndpointFromTask(ctx context.Context, taskLog *dbent.SocialTaskLog) (string, error) {
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
	trimmed, err := ResolveSocialIPExecutionEndpoint(ctx, snapshot.Endpoint)
	if err != nil {
		return "", newSocialExecutionError(SocialExecutionFailureProxyInvalid, "execution proxy is invalid", err)
	}
	return trimmed, nil
}

type twitterAPIClient struct {
	httpClient     twitterHTTPClient
	auth           *twitterAuthHeaders
	endpoints      twitterEndpoints
	onHTTPResponse func()
}

type preparedTwitterTask struct {
	target  string
	post    *preparedTwitterPost
	profile *SocialProfileUpdateParams
	avatar  *twitterPreparedMedia
	banner  *twitterPreparedMedia
}

type preparedTwitterPost struct {
	text         string
	quotePostURL string
	media        []*twitterPreparedMedia
}

type twitterPreparedMedia struct {
	fieldName   string
	contentType string
	fileName    string
	body        []byte
	md5Hex      string
}

type twitterMediaUploadFinalizeResponse struct {
	MediaIDString  string                      `json:"media_id_string"`
	ProcessingInfo *twitterMediaProcessingInfo `json:"processing_info"`
}

type twitterMediaProcessingInfo struct {
	State           string                       `json:"state"`
	CheckAfterSecs  int                          `json:"check_after_secs"`
	ProgressPercent int                          `json:"progress_percent"`
	Error           *twitterMediaProcessingError `json:"error"`
}

type twitterMediaProcessingError struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
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

func (c *twitterAPIClient) createTweet(ctx context.Context, post *preparedTwitterPost) (*twitterActionResult, error) {
	if post == nil {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "post content or media is required", nil)
	}
	text := strings.TrimSpace(post.text)
	attachmentURL := strings.TrimSpace(post.quotePostURL)
	if text == "" && len(post.media) == 0 {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "post content or media is required", nil)
	}
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
	if attachmentURL != "" {
		variables["attachment_url"] = attachmentURL
	}
	if len(post.media) > 0 {
		mediaEntities := make([]map[string]any, 0, len(post.media))
		for _, media := range post.media {
			mediaID, err := c.uploadMedia(ctx, media)
			if err != nil {
				return nil, err
			}
			parsedID, err := strconv.ParseInt(mediaID, 10, 64)
			if err != nil {
				return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned invalid media id", err)
			}
			mediaEntities = append(mediaEntities, map[string]any{
				"media_id":     parsedID,
				"tagged_users": []any{},
			})
		}
		variables["media"] = map[string]any{
			"media_entities":     mediaEntities,
			"possibly_sensitive": false,
		}
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

func (c *twitterAPIClient) updateProfile(ctx context.Context, profile *SocialProfileUpdateParams) (*twitterActionResult, error) {
	if profile == nil || profile.IsZero() {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "profile update params are required", nil)
	}
	form := twitterProfileUpdateForm(profile)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoints.updateProfile, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter request", err)
	}
	c.setFormHeaders(req, form)
	return c.sendSimple(req, "profile updated")
}

func (c *twitterAPIClient) updateProfileImage(ctx context.Context, media *twitterPreparedMedia) (*twitterActionResult, error) {
	return c.sendMultipartMedia(ctx, c.endpoints.updateProfileImage, media, "avatar updated", "")
}

func (c *twitterAPIClient) updateProfileBanner(ctx context.Context, media *twitterPreparedMedia) (*twitterActionResult, error) {
	query := ""
	if media != nil && media.md5Hex != "" {
		query = "?return_user=true&original_md5=" + media.md5Hex
	}
	return c.sendMultipartMedia(ctx, c.endpoints.updateProfileBanner, media, "banner updated", query)
}

func (c *twitterAPIClient) uploadMedia(ctx context.Context, media *twitterPreparedMedia) (string, error) {
	if media != nil && strings.HasPrefix(strings.ToLower(strings.TrimSpace(media.contentType)), "video/") {
		return c.uploadChunkedVideoMedia(ctx, media)
	}
	req, err := c.newMultipartMediaRequest(ctx, c.endpoints.mediaUpload, media, queryWithOriginalMD5(media))
	if err != nil {
		return "", err
	}
	resp, body, err := c.do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		message := twitterErrorMessage(body, resp.StatusCode)
		return "", newSocialExecutionError(twitterFailureKind(&twitterActionResult{
			StatusCode: resp.StatusCode,
			Message:    message,
			RawBody:    string(body),
		}), message, nil)
	}
	var parsed struct {
		MediaIDString string `json:"media_id_string"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned invalid response", err)
	}
	if strings.TrimSpace(parsed.MediaIDString) == "" {
		return "", newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned no media id", nil)
	}
	return parsed.MediaIDString, nil
}

func (c *twitterAPIClient) uploadChunkedVideoMedia(ctx context.Context, media *twitterPreparedMedia) (string, error) {
	mediaID, err := c.initChunkedVideoUpload(ctx, media)
	if err != nil {
		return "", err
	}
	if err := c.appendChunkedVideoUpload(ctx, mediaID, media); err != nil {
		return "", err
	}
	processing, err := c.finalizeChunkedVideoUpload(ctx, mediaID)
	if err != nil {
		return "", err
	}
	if err := c.waitForChunkedVideoProcessing(ctx, mediaID, processing); err != nil {
		return "", err
	}
	return mediaID, nil
}

func (c *twitterAPIClient) initChunkedVideoUpload(ctx context.Context, media *twitterPreparedMedia) (string, error) {
	form := url.Values{}
	form.Set("command", "INIT")
	form.Set("total_bytes", strconv.Itoa(len(media.body)))
	form.Set("media_type", media.contentType)
	form.Set("media_category", "tweet_video")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoints.mediaUpload, strings.NewReader(form.Encode()))
	if err != nil {
		return "", newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter request", err)
	}
	c.setFormHeaders(req, form)
	resp, body, err := c.do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		message := twitterErrorMessage(body, resp.StatusCode)
		return "", newSocialExecutionError(twitterFailureKind(&twitterActionResult{
			StatusCode: resp.StatusCode,
			Message:    message,
			RawBody:    string(body),
		}), message, nil)
	}
	var parsed twitterMediaUploadFinalizeResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned invalid response", err)
	}
	if strings.TrimSpace(parsed.MediaIDString) == "" {
		return "", newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned no media id", nil)
	}
	return parsed.MediaIDString, nil
}

func (c *twitterAPIClient) appendChunkedVideoUpload(ctx context.Context, mediaID string, media *twitterPreparedMedia) error {
	if media == nil || len(media.body) == 0 {
		return newSocialExecutionError(SocialExecutionFailureActionInput, "media is required", nil)
	}
	for segmentIndex, offset := 0, 0; offset < len(media.body); segmentIndex, offset = segmentIndex+1, offset+twitterVideoChunkBytes {
		end := offset + twitterVideoChunkBytes
		if end > len(media.body) {
			end = len(media.body)
		}
		chunk := &twitterPreparedMedia{
			fieldName:   "media",
			contentType: media.contentType,
			fileName:    media.fileName,
			body:        media.body[offset:end],
		}
		fields := url.Values{}
		fields.Set("command", "APPEND")
		fields.Set("media_id", mediaID)
		fields.Set("segment_index", strconv.Itoa(segmentIndex))
		req, err := c.newMultipartMediaRequestWithFields(ctx, c.endpoints.mediaUpload, chunk, "", fields)
		if err != nil {
			return err
		}
		resp, body, err := c.do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			message := twitterErrorMessage(body, resp.StatusCode)
			return newSocialExecutionError(twitterFailureKind(&twitterActionResult{
				StatusCode: resp.StatusCode,
				Message:    message,
				RawBody:    string(body),
			}), message, nil)
		}
	}
	return nil
}

func (c *twitterAPIClient) finalizeChunkedVideoUpload(ctx context.Context, mediaID string) (*twitterMediaProcessingInfo, error) {
	form := url.Values{}
	form.Set("command", "FINALIZE")
	form.Set("media_id", mediaID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoints.mediaUpload, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter request", err)
	}
	c.setFormHeaders(req, form)
	resp, body, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		message := twitterErrorMessage(body, resp.StatusCode)
		return nil, newSocialExecutionError(twitterFailureKind(&twitterActionResult{
			StatusCode: resp.StatusCode,
			Message:    message,
			RawBody:    string(body),
		}), message, nil)
	}
	var parsed twitterMediaUploadFinalizeResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned invalid response", err)
	}
	if strings.TrimSpace(parsed.MediaIDString) == "" {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned no media id", nil)
	}
	return parsed.ProcessingInfo, nil
}

func (c *twitterAPIClient) waitForChunkedVideoProcessing(ctx context.Context, mediaID string, processing *twitterMediaProcessingInfo) error {
	current := processing
	for polls := 0; current != nil && polls < twitterMediaStatusMaxPolls; polls++ {
		state := strings.ToLower(strings.TrimSpace(current.State))
		switch state {
		case "", "succeeded":
			return nil
		case "failed":
			return newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned processing failed", twitterMediaProcessingFailureError(current))
		case "pending", "in_progress":
			if err := waitForTwitterMediaProcessing(ctx, current.CheckAfterSecs); err != nil {
				return err
			}
			next, err := c.fetchChunkedVideoUploadStatus(ctx, mediaID)
			if err != nil {
				return err
			}
			current = next
		default:
			return newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned processing failed", fmt.Errorf("unknown processing state %q", current.State))
		}
	}
	if current == nil {
		return nil
	}
	return newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned processing timeout", nil)
}

func (c *twitterAPIClient) fetchChunkedVideoUploadStatus(ctx context.Context, mediaID string) (*twitterMediaProcessingInfo, error) {
	statusURL := c.endpoints.mediaUpload + "?command=STATUS&media_id=" + url.QueryEscape(mediaID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter request", err)
	}
	c.setUploadStatusHeaders(req)
	resp, body, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		message := twitterErrorMessage(body, resp.StatusCode)
		return nil, newSocialExecutionError(twitterFailureKind(&twitterActionResult{
			StatusCode: resp.StatusCode,
			Message:    message,
			RawBody:    string(body),
		}), message, nil)
	}
	var parsed twitterMediaUploadFinalizeResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned invalid response", err)
	}
	if strings.TrimSpace(parsed.MediaIDString) == "" {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned no media id", nil)
	}
	return parsed.ProcessingInfo, nil
}

func twitterMediaProcessingFailureError(processing *twitterMediaProcessingInfo) error {
	if processing == nil || processing.Error == nil {
		return nil
	}
	parts := make([]string, 0, 3)
	if processing.Error.Name != "" {
		parts = append(parts, processing.Error.Name)
	}
	if processing.Error.Message != "" {
		parts = append(parts, processing.Error.Message)
	}
	if processing.Error.Code != 0 {
		parts = append(parts, fmt.Sprintf("code=%d", processing.Error.Code))
	}
	if len(parts) == 0 {
		return nil
	}
	return errors.New(strings.Join(parts, ": "))
}

func waitForTwitterMediaProcessing(ctx context.Context, checkAfterSecs int) error {
	if checkAfterSecs <= 0 {
		return nil
	}
	timer := time.NewTimer(time.Duration(checkAfterSecs) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return newSocialExecutionError(SocialExecutionFailurePlatform, "twitter media upload returned processing timeout", ctx.Err())
	case <-timer.C:
		return nil
	}
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
	if req == nil {
		return nil, nil, newSocialExecutionError(SocialExecutionFailurePlatform, "twitter request is unavailable", nil)
	}
	var lastErr error
	for attempt := 0; attempt <= twitterNetworkMaxRetries; attempt++ {
		attemptReq := req
		if attempt > 0 {
			if err := waitTwitterNetworkRetryBackoff(req.Context(), attempt); err != nil {
				return nil, nil, newSocialExecutionError(SocialExecutionFailureNetwork, "twitter network request cancelled", err)
			}
			cloned, err := twitterRequestForRetry(req)
			if err != nil {
				break
			}
			attemptReq = cloned
		}

		resp, body, err := c.doOnce(attemptReq)
		if err == nil {
			return resp, body, nil
		}
		lastErr = err
		if kind, ok := socialExecutionFailureKind(err); ok && kind != SocialExecutionFailureNetwork {
			return nil, nil, err
		}
		if !isRetryableTwitterNetworkError(err) || attempt == twitterNetworkMaxRetries {
			break
		}
		slog.Warn(
			"twitter network request retrying",
			"attempt", attempt+1,
			"max_retries", twitterNetworkMaxRetries,
			"method", attemptReq.Method,
			"host", twitterRequestHost(attemptReq),
			"error", err,
		)
	}
	return nil, nil, twitterNetworkExecutionError(lastErr)
}

func (c *twitterAPIClient) doOnce(req *http.Request) (*http.Response, []byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if c.onHTTPResponse != nil {
		c.onHTTPResponse()
	}
	defer resp.Body.Close()
	body, err := readTwitterResponseBody(resp)
	if err != nil {
		return nil, nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to read twitter response", err)
	}
	return resp, body, nil
}

func waitTwitterNetworkRetryBackoff(ctx context.Context, attempt int) error {
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

func twitterRequestForRetry(req *http.Request) (*http.Request, error) {
	if req == nil {
		return nil, fmt.Errorf("twitter request is unavailable")
	}
	cloned := req.Clone(req.Context())
	if req.Body == nil {
		return cloned, nil
	}
	if req.GetBody == nil {
		return nil, fmt.Errorf("twitter request body is not replayable")
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	cloned.Body = body
	cloned.GetBody = req.GetBody
	cloned.ContentLength = req.ContentLength
	return cloned, nil
}

func twitterNetworkExecutionError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return newSocialExecutionError(SocialExecutionFailureNetwork, "twitter network request timed out", err)
	}
	return newSocialExecutionError(SocialExecutionFailureNetwork, "twitter network request failed", err)
}

func twitterRequestHost(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}
	return req.URL.Host
}

func (c *twitterAPIClient) setFormHeaders(req *http.Request, form url.Values) {
	c.setCommonHeaders(req, "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", generateTwitterOAuthHeader(req.Method, req.URL.String(), form, c.auth.AccessToken, c.auth.TokenSecret))
}

func (c *twitterAPIClient) setGraphQLHeaders(req *http.Request) {
	c.setCommonHeaders(req, "application/json")
	req.Header.Set("Authorization", generateTwitterOAuthHeader(req.Method, req.URL.String(), nil, c.auth.AccessToken, c.auth.TokenSecret))
}

func (c *twitterAPIClient) setMultipartHeaders(req *http.Request, contentType string) {
	c.setCommonHeaders(req, contentType)
	req.Header.Set("Authorization", generateTwitterOAuthHeader(req.Method, req.URL.String(), nil, c.auth.AccessToken, c.auth.TokenSecret))
}

func (c *twitterAPIClient) setUploadStatusHeaders(req *http.Request) {
	c.setCommonHeaders(req, "application/x-www-form-urlencoded")
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
		message := strings.TrimSpace(errResp.Errors[0].Message)
		if message == "" {
			return fmt.Sprintf("twitter error %d", code)
		}
		return fmt.Sprintf("twitter error %d: %s", code, shortBusinessMessage(message))
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
	if kind, ok := twitterTaskFailureKind(result.Message); ok {
		return kind
	}
	if IsTwitterPlatformFailureMessage(result.Message) {
		return SocialExecutionFailurePlatform
	}
	message := strings.ToLower(strings.TrimSpace(result.Message))
	if failure, ok := knownTwitterFailureDetail(message); ok {
		return failure.kind
	}
	switch result.StatusCode {
	case http.StatusUnauthorized:
		return SocialExecutionFailureAuthInvalid
	case http.StatusForbidden:
		if twitterMessageIndicatesChallenge(message) {
			return SocialExecutionFailureChallengeRequired
		}
		if strings.Contains(message, "suspended") ||
			strings.Contains(message, "locked") ||
			strings.Contains(message, "limit") ||
			strings.Contains(message, "automated") {
			return SocialExecutionFailureAccountLimited
		}
		return SocialExecutionFailurePlatform
	case http.StatusTooManyRequests:
		return SocialExecutionFailureAccountLimited
	}
	if strings.Contains(message, "authentication failed") {
		return SocialExecutionFailureAuthInvalid
	}
	if twitterMessageIndicatesChallenge(message) {
		return SocialExecutionFailureChallengeRequired
	}
	if strings.Contains(message, "suspended") ||
		strings.Contains(message, "locked") ||
		strings.Contains(message, "automated") ||
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

func twitterMessageIndicatesChallenge(message string) bool {
	return strings.Contains(message, "challenge required") ||
		strings.Contains(message, "captcha challenge") ||
		strings.Contains(message, "verification required") ||
		strings.Contains(message, "verify login") ||
		strings.Contains(message, "additional verification") ||
		strings.Contains(message, "confirm your identity")
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

func (e *TwitterExecutor) prepareTwitterTask(ctx context.Context, taskLog *dbent.SocialTaskLog) (*preparedTwitterTask, error) {
	if taskLog == nil {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "twitter task input is unavailable", nil)
	}

	prepared := &preparedTwitterTask{
		target: strings.TrimSpace(stringPtrValue(taskLog.Target)),
	}
	if payloadTarget := strings.TrimSpace(taskLog.Payload.Target); payloadTarget != "" {
		prepared.target = payloadTarget
	}

	switch taskLog.Action {
	case SocialTaskActionPost:
		postPayload := taskLog.Payload.Post
		if postPayload == nil {
			postPayload = &SocialPostPayload{Text: strings.TrimSpace(stringPtrValue(taskLog.Content))}
		}
		if strings.TrimSpace(postPayload.Text) == "" {
			postPayload.Text = strings.TrimSpace(stringPtrValue(taskLog.Content))
		}
		postMedia, err := e.prepareTwitterPostMedia(ctx, taskLog.UserID, postPayload.Media)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(postPayload.Text) == "" && len(postMedia) == 0 {
			return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "post content or media is required", nil)
		}
		prepared.post = &preparedTwitterPost{
			text:         strings.TrimSpace(postPayload.Text),
			quotePostURL: strings.TrimSpace(postPayload.QuotePostURL),
			media:        postMedia,
		}
	case SocialTaskActionUpdateProfile:
		if taskLog.Payload.Profile == nil || taskLog.Payload.Profile.IsZero() {
			return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "profile update params are required", nil)
		}
		prepared.profile = taskLog.Payload.Profile
	case SocialTaskActionUpdateAvatar:
		if taskLog.Payload.Avatar == nil {
			return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "avatar media is required", nil)
		}
		if err := validateSocialTaskImageMedia("avatar", taskLog.Payload.Avatar); err != nil {
			return nil, newSocialExecutionError(SocialExecutionFailureActionInput, err.Error(), nil)
		}
		media, err := e.prepareTwitterImageMedia(ctx, taskLog.UserID, taskLog.Payload.Avatar, "avatar", "image")
		if err != nil {
			return nil, err
		}
		media, err = normalizeTwitterProfileMedia(media, "avatar", socialTaskAvatarImageWidth, socialTaskAvatarImageHeight)
		if err != nil {
			return nil, err
		}
		prepared.avatar = media
	case SocialTaskActionUpdateBanner:
		if taskLog.Payload.Banner == nil {
			return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "banner media is required", nil)
		}
		if err := validateSocialTaskImageMedia("banner", taskLog.Payload.Banner); err != nil {
			return nil, newSocialExecutionError(SocialExecutionFailureActionInput, err.Error(), nil)
		}
		media, err := e.prepareTwitterImageMedia(ctx, taskLog.UserID, taskLog.Payload.Banner, "banner", "banner")
		if err != nil {
			return nil, err
		}
		media, err = normalizeTwitterProfileMedia(media, "banner", socialTaskBannerImageWidth, socialTaskBannerImageHeight)
		if err != nil {
			return nil, err
		}
		prepared.banner = media
	}

	return prepared, nil
}

func twitterProfileUpdateForm(profile *SocialProfileUpdateParams) url.Values {
	form := url.Values{}
	if screenName := strings.TrimSpace(profile.ScreenName); screenName != "" {
		form.Set("screen_name", screenName)
	}
	if displayName := strings.TrimSpace(profile.DisplayName); displayName != "" {
		form.Set("name", displayName)
	}
	if description := strings.TrimSpace(profile.Description); description != "" {
		form.Set("description", description)
	}
	if location := strings.TrimSpace(profile.Location); location != "" {
		form.Set("location", location)
	}
	if website := strings.TrimSpace(profile.URL); website != "" {
		form.Set("url", website)
	}
	return form
}

func prepareTwitterInlineMedia(ref *SocialTaskMediaRef, label, fieldName string) (*twitterPreparedMedia, error) {
	if ref == nil || ref.IsZero() {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media is required", label), nil)
	}
	if strings.TrimSpace(ref.Source) != "" && strings.TrimSpace(ref.Source) != "inline" {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf(socialTaskMediaSourceUnsupportedMessage, label), nil)
	}
	contentType, body, err := parseTwitterDataURL(strings.TrimSpace(ref.URL))
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media is invalid", label), err)
	}
	if declaredType := strings.TrimSpace(ref.ContentType); declaredType != "" {
		contentType = declaredType
	}
	fileName := strings.TrimSpace(ref.FileName)
	if fileName == "" {
		fileName = label + twitterFileExtensionFromContentType(contentType)
	}
	md5Sum := md5.Sum(body)
	return &twitterPreparedMedia{
		fieldName:   fieldName,
		contentType: contentType,
		fileName:    fileName,
		body:        body,
		md5Hex:      fmt.Sprintf("%x", md5Sum[:]),
	}, nil
}

func (e *TwitterExecutor) prepareTwitterImageMedia(ctx context.Context, userID int64, ref *SocialTaskMediaRef, label, fieldName string) (*twitterPreparedMedia, error) {
	media, err := e.prepareTwitterMedia(ctx, userID, ref, label, fieldName)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(media.contentType)), "image/") {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media must be an image", label), nil)
	}
	return media, nil
}

func (e *TwitterExecutor) prepareTwitterPostMedia(ctx context.Context, userID int64, refs []SocialTaskMediaRef) ([]*twitterPreparedMedia, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	prepared := make([]*twitterPreparedMedia, 0, len(refs))
	for i, ref := range refs {
		label := fmt.Sprintf("post media #%d", i+1)
		media, err := e.prepareTwitterMedia(ctx, userID, &ref, label, "media")
		if err != nil {
			return nil, err
		}
		contentType := strings.ToLower(strings.TrimSpace(media.contentType))
		switch {
		case strings.HasPrefix(contentType, "video/"):
			return nil, newSocialExecutionError(SocialExecutionFailureActionInput, socialTaskVideoUnsupportedMessage, nil)
		case !strings.HasPrefix(contentType, "image/"):
			return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "post media content type is not supported", nil)
		}
		prepared = append(prepared, media)
	}
	return prepared, nil
}

func (e *TwitterExecutor) prepareTwitterMedia(ctx context.Context, userID int64, ref *SocialTaskMediaRef, label, fieldName string) (*twitterPreparedMedia, error) {
	if ref == nil || ref.IsZero() {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media is required", label), nil)
	}
	source := strings.TrimSpace(ref.Source)
	if source == "" || source == "inline" {
		return prepareTwitterInlineMedia(ref, label, fieldName)
	}
	if source != "library" {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf(socialTaskMediaSourceUnsupportedMessage, label), nil)
	}
	if e == nil || e.mediaResolver == nil {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf(socialTaskMediaSourceUnsupportedMessage, label), nil)
	}
	resolved, err := e.mediaResolver.Resolve(ctx, userID, ref)
	if err != nil {
		if errors.Is(err, errSocialTaskMediaAssetUnavailable) || errors.Is(err, errSocialTaskMediaAssetInvalid) {
			return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media asset is unavailable", label), err)
		}
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media is invalid", label), err)
	}
	contentType := strings.TrimSpace(resolved.ContentType)
	if contentType == "" {
		contentType = strings.TrimSpace(ref.ContentType)
	}
	fileName := strings.TrimSpace(resolved.FileName)
	if fileName == "" {
		fileName = strings.TrimSpace(ref.FileName)
	}
	if fileName == "" {
		fileName = label + twitterFileExtensionFromContentType(contentType)
	}
	md5Sum := md5.Sum(resolved.Body)
	return &twitterPreparedMedia{
		fieldName:   fieldName,
		contentType: contentType,
		fileName:    fileName,
		body:        resolved.Body,
		md5Hex:      fmt.Sprintf("%x", md5Sum[:]),
	}, nil
}

func parseTwitterDataURL(raw string) (string, []byte, error) {
	if raw == "" {
		return "", nil, fmt.Errorf("data url is required")
	}
	if !strings.HasPrefix(raw, "data:") {
		return "", nil, fmt.Errorf("only data url media is supported")
	}
	comma := strings.Index(raw, ",")
	if comma <= len("data:") {
		return "", nil, fmt.Errorf("data url is malformed")
	}
	meta := raw[len("data:"):comma]
	dataPart := raw[comma+1:]
	if !strings.HasSuffix(strings.ToLower(meta), ";base64") {
		return "", nil, fmt.Errorf("data url must be base64 encoded")
	}
	contentType := strings.TrimSuffix(meta, ";base64")
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	body, err := base64.StdEncoding.DecodeString(dataPart)
	if err != nil {
		return "", nil, err
	}
	if len(body) == 0 {
		return "", nil, fmt.Errorf("media body is empty")
	}
	return contentType, body, nil
}

func twitterFileExtensionFromContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}

func (c *twitterAPIClient) sendMultipartMedia(ctx context.Context, endpoint string, media *twitterPreparedMedia, successMessage, rawQuery string) (*twitterActionResult, error) {
	req, err := c.newMultipartMediaRequest(ctx, endpoint, media, rawQuery)
	if err != nil {
		return nil, err
	}
	return c.sendSimple(req, successMessage)
}

func (c *twitterAPIClient) newMultipartMediaRequest(ctx context.Context, endpoint string, media *twitterPreparedMedia, rawQuery string) (*http.Request, error) {
	return c.newMultipartMediaRequestWithFields(ctx, endpoint, media, rawQuery, nil)
}

func (c *twitterAPIClient) newMultipartMediaRequestWithFields(ctx context.Context, endpoint string, media *twitterPreparedMedia, rawQuery string, fields url.Values) (*http.Request, error) {
	if media == nil || len(media.body) == 0 {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, "media is required", nil)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.SetBoundary("twitter")

	for key, values := range fields {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to build twitter media request", err)
			}
		}
	}

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, media.fieldName, media.fileName))
	partHeader.Set("Content-Type", media.contentType)
	partHeader.Set("Content-Transfer-Encoding", "binary")
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to build twitter media request", err)
	}
	if _, err := part.Write(media.body); err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to build twitter media request", err)
	}
	if err := writer.Close(); err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to build twitter media request", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+rawQuery, &body)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailurePlatform, "failed to create twitter request", err)
	}
	c.setMultipartHeaders(req, writer.FormDataContentType())
	return req, nil
}

func queryWithOriginalMD5(media *twitterPreparedMedia) string {
	if media == nil || strings.TrimSpace(media.md5Hex) == "" {
		return ""
	}
	return "?original_md5=" + media.md5Hex
}
