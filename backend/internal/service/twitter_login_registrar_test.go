//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type twitterLoginFakeClient struct {
	t       *testing.T
	mu      sync.Mutex
	calls   []string
	bodies  []string
	handler func(*http.Request, string, int) (*http.Response, error)
}

func (c *twitterLoginFakeClient) Do(req *http.Request) (*http.Response, error) {
	body := ""
	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		require.NoError(c.t, err)
		body = string(data)
	}
	c.mu.Lock()
	idx := len(c.calls)
	c.calls = append(c.calls, req.Method+" "+req.URL.String())
	c.bodies = append(c.bodies, body)
	c.mu.Unlock()
	return c.handler(req, body, idx)
}

func TestTwitterExecutionAuthBuildsOAuthPayload(t *testing.T) {
	headers := defaultTwitterAuthHeaders()
	headers.AccessToken = "123456-oauth-token"
	headers.TokenSecret = "oauth-secret"
	headers.ScreenName = "@northwind_ops"
	headers.GuestToken = "guest-token"
	headers.Kdt = "kdt-token"
	headers.Att = "att-token"
	headers.ClientUuid = "client-uuid"
	headers.ClientDeviceId = "device-id"

	executionAuth, err := twitterExecutionAuthFromHeaders(headers)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(executionAuth)))
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(executionAuth), &payload))
	require.Equal(t, "123456-oauth-token", payload["access_token"])
	require.Equal(t, "oauth-secret", payload["token_secret"])
	require.Equal(t, "northwind_ops", payload["screen_name"])
	require.Len(t, payload, 3)
	require.NotContains(t, payload, "guest_token")
	require.NotContains(t, payload, "kdt")
	require.NotContains(t, payload, "att")
	require.NotContains(t, payload, "client_uuid")
	require.NotContains(t, payload, "client_device_id")

	auth, err := twitterAuthHeadersFromExecutionAuth(executionAuth)
	require.NoError(t, err)
	require.Equal(t, "123456-oauth-token", auth.AccessToken)
	require.Equal(t, "oauth-secret", auth.TokenSecret)
	require.Equal(t, "northwind_ops", auth.ScreenName)
}

func twitterLoginJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestTwitterAccountCredentialRegistrarCompletesCredentialLoginWith2FAAndAcid(t *testing.T) {
	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		switch idx {
		case 0:
			require.Equal(t, "no", req.Header.Get("x-twitter-active-user"))
			return twitterLoginJSONResponse(http.StatusOK, `{"guest_token":"guest-token"}`), nil
		case 1:
			require.Equal(t, "yes", req.Header.Get("x-twitter-active-user"))
			require.Equal(t, "guest-token", req.Header.Get("x-guest-token"))
			require.Contains(t, req.Header.Get("authorization"), "Bearer ")
			require.Equal(t, defaultTwitterAuthHeaders().OsVersion, req.Header.Get("os-version"))
			require.Equal(t, defaultTwitterAuthHeaders().TwitterDisplaySize, req.Header.Get("twitter-display-size"))
			require.Contains(t, req.Header, "X-Twitter-Client-Flavor")
			resp := twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-1","subtasks":[{"subtask_id":"LoginEnterUserIdentifier"}]}`)
			resp.Header.Set("att", "att-token")
			return resp, nil
		case 2:
			require.Contains(t, body, `"subtask_id":"LoginEnterUserIdentifier"`)
			require.Contains(t, body, `"text":"northwind_ops"`)
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-2","subtasks":[{"subtask_id":"LoginEnterPassword"}]}`), nil
		case 3:
			require.Contains(t, body, `"subtask_id":"LoginEnterPassword"`)
			require.Contains(t, body, `"password":"p@ssw0rd"`)
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-3","subtasks":[{"subtask_id":"LoginTwoFactorAuthChallenge","enter_text":{"hint_text":"Enter code"}}]}`), nil
		case 4:
			require.Contains(t, body, `"subtask_id":"LoginTwoFactorAuthChallenge"`)
			require.Contains(t, body, `"text":"996554"`)
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-4","subtasks":[{"subtask_id":"LoginAcid","enter_text":{"hint_text":"Confirmation code"}}]}`), nil
		case 5:
			require.Contains(t, body, `"subtask_id":"LoginAcid"`)
			require.Contains(t, body, `"text":"654321"`)
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-5","subtasks":[{"subtask_id":"LoginSuccessSubtask","open_account":{"oauth_token":"oauth-token","oauth_token_secret":"oauth-secret","known_device_token":"kdt-token"}}]}`), nil
		case 6:
			require.Contains(t, body, `"subtask_id":"LoginSuccessSubtask"`)
			require.Empty(t, req.Header.Get("x-guest-token"))
			require.Equal(t, "kdt-token", req.Header.Get("kdt"))
			require.Contains(t, req.Header.Get("authorization"), `oauth_token="oauth-token"`)
			require.NotContains(t, req.Header.Get("authorization"), "Bearer ")
			return twitterLoginJSONResponse(http.StatusOK, `{"status":"success"}`), nil
		case 7:
			require.Equal(t, http.MethodGet, req.Method)
			require.Empty(t, req.Header.Get("x-guest-token"))
			require.Equal(t, "kdt-token", req.Header.Get("kdt"))
			require.Contains(t, req.Header.Get("authorization"), `oauth_token="oauth-token"`)
			return twitterLoginJSONResponse(http.StatusOK, `{"data":{"viewer":{"userResult":{"result":{"rest_id":"12345"}}}}}`), nil
		default:
			t.Fatalf("unexpected twitter login request #%d: %s %s", idx, req.Method, req.URL.String())
			return nil, nil
		}
	}

	registrar := NewTwitterAccountCredentialRegistrar()
	registrar.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://proxy.local:8080", proxyURL)
		return fake, nil
	}
	registrar.now = func() time.Time { return time.Unix(59, 0).UTC() }
	registrar.emailCodeResolver = func(ctx context.Context, req SocialAccountCredentialRequest) (string, error) {
		require.Equal(t, "northwind@example.com", req.Email)
		require.Equal(t, "email-secret", req.EmailPassword)
		require.Equal(t, "refresh-token", req.EmailToken)
		require.Equal(t, "client-id", req.EmailClientID)
		return "654321", nil
	}

	result, err := registrar.AcquireCredentials(context.Background(), SocialAccountCredentialRequest{
		Name:          "northwind_ops",
		Password:      "p@ssw0rd",
		Email:         "northwind@example.com",
		EmailPassword: "email-secret",
		TwoFactor:     "JBSWY3DPEHPK3PXP",
		EmailToken:    "refresh-token",
		EmailClientID: "client-id",
		ProxyEndpoint: "http://proxy.local:8080",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "login succeeded", result.Message)
	require.Equal(t, "12345", requireStringPtr(t, result.PlatformUserID))

	require.NotEqual(t, result.AuthCookie, result.ExecutionAuth)
	require.True(t, json.Valid([]byte(result.ExecutionAuth)))
	var stored map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.ExecutionAuth), &stored))
	require.Equal(t, "oauth-token", stored["access_token"])
	require.Equal(t, "oauth-secret", stored["token_secret"])
	require.Equal(t, "northwind_ops", stored["screen_name"])
	require.Len(t, stored, 3)
	require.NotContains(t, stored, "guest_token")
	require.NotContains(t, stored, "kdt")
	require.NotContains(t, stored, "att")
	require.NotContains(t, stored, "twitter_client")
	require.NotContains(t, stored, "client_version")
	require.NotContains(t, stored, "user_agent")
	require.NotContains(t, stored, "authorization")

	var authCookie twitterAuthHeaders
	require.NoError(t, json.Unmarshal([]byte(result.AuthCookie), &authCookie))
	require.Equal(t, "guest-token", authCookie.GuestToken)
	require.Equal(t, "kdt-token", authCookie.Kdt)
	require.Equal(t, "att-token", authCookie.Att)
	require.Equal(t, "northwind_ops", authCookie.ScreenName)

	auth, err := twitterAuthHeadersFromExecutionAuth(result.ExecutionAuth)
	require.NoError(t, err)
	require.Equal(t, "oauth-token", auth.AccessToken)
	require.Equal(t, "oauth-secret", auth.TokenSecret)
	require.Equal(t, "northwind_ops", auth.ScreenName)
}

func TestTwitterAccountCredentialRegistrarRetriesRetryableNetworkErrors(t *testing.T) {
	originalBackoff := twitterNetworkRetryBackoffDuration
	twitterNetworkRetryBackoffDuration = func(int) time.Duration { return 0 }
	t.Cleanup(func() { twitterNetworkRetryBackoffDuration = originalBackoff })

	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		switch idx {
		case 0, 1:
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("temporary connection reset")}
		case 2:
			return twitterLoginJSONResponse(http.StatusOK, `{"guest_token":"guest-token"}`), nil
		case 3:
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-1","subtasks":[{"subtask_id":"LoginEnterPassword"}]}`), nil
		case 4:
			require.Contains(t, body, `"password":"p@ssw0rd"`)
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-2","subtasks":[{"subtask_id":"LoginSuccessSubtask","open_account":{"oauth_token":"oauth-token","oauth_token_secret":"oauth-secret"}}]}`), nil
		case 5:
			return twitterLoginJSONResponse(http.StatusOK, `{"status":"success"}`), nil
		case 6:
			return twitterLoginJSONResponse(http.StatusOK, `{"data":{"viewer":{"userResult":{"result":{"rest_id":"12345"}}}}}`), nil
		default:
			t.Fatalf("unexpected twitter login request #%d: %s %s", idx, req.Method, req.URL.String())
			return nil, nil
		}
	}

	registrar := NewTwitterAccountCredentialRegistrar()
	registrar.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://proxy.local:8080", proxyURL)
		return fake, nil
	}

	result, err := registrar.AcquireCredentials(context.Background(), SocialAccountCredentialRequest{
		Name:          "northwind_ops",
		Password:      "p@ssw0rd",
		ProxyEndpoint: "http://proxy.local:8080",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "login succeeded", result.Message)
	require.Len(t, fake.calls, 7)
}

func TestTwitterAccountCredentialRegistrarNetworkFailureUsesNetworkMessage(t *testing.T) {
	originalBackoff := twitterNetworkRetryBackoffDuration
	twitterNetworkRetryBackoffDuration = func(int) time.Duration { return 0 }
	t.Cleanup(func() { twitterNetworkRetryBackoffDuration = originalBackoff })

	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("temporary connection reset")}
	}

	registrar := NewTwitterAccountCredentialRegistrar()
	registrar.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://proxy.local:8080", proxyURL)
		return fake, nil
	}

	_, err := registrar.AcquireCredentials(context.Background(), SocialAccountCredentialRequest{
		Name:          "northwind_ops",
		Password:      "p@ssw0rd",
		ProxyEndpoint: "http://proxy.local:8080",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "twitter login network request failed")
	require.NotContains(t, err.Error(), "check execution proxy")
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureNetwork, kind)
	require.Len(t, fake.calls, twitterLoginNetworkMaxRetries+1)
}

func TestTwitterAccountCredentialRegistrarPasswordFailureUsesPasswordMessage(t *testing.T) {
	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		switch idx {
		case 0:
			return twitterLoginJSONResponse(http.StatusOK, `{"guest_token":"guest-token"}`), nil
		case 1:
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-1","subtasks":[{"subtask_id":"LoginEnterPassword"}]}`), nil
		case 2:
			require.Contains(t, body, `"subtask_id":"LoginEnterPassword"`)
			require.Contains(t, body, `"password":"wrong-password"`)
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-2","subtasks":[{"subtask_id":"LoginEnterPassword","enter_password":{"error_message":"The password you entered is incorrect."}}]}`), nil
		default:
			t.Fatalf("unexpected twitter login request #%d: %s %s", idx, req.Method, req.URL.String())
			return nil, nil
		}
	}

	registrar := NewTwitterAccountCredentialRegistrar()
	registrar.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://proxy.local:8080", proxyURL)
		return fake, nil
	}

	_, err := registrar.AcquireCredentials(context.Background(), SocialAccountCredentialRequest{
		Name:          "northwind_ops",
		Password:      "wrong-password",
		ProxyEndpoint: "http://proxy.local:8080",
	})

	require.Error(t, err)
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailurePasswordInvalid, kind)
	require.Equal(t, "密码错误，本次未扣费", safeSocialTaskFailureMessage(err))
	require.Len(t, fake.calls, 3)
}

func TestTwitterAccountCredentialRegistrarHTTP399WrongPasswordUsesPasswordMessage(t *testing.T) {
	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		switch idx {
		case 0:
			return twitterLoginJSONResponse(http.StatusOK, `{"guest_token":"guest-token"}`), nil
		case 1:
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-1","subtasks":[{"subtask_id":"LoginEnterPassword"}]}`), nil
		case 2:
			require.Contains(t, body, `"subtask_id":"LoginEnterPassword"`)
			require.Contains(t, body, `"password":"wrong-password"`)
			return twitterLoginJSONResponse(http.StatusForbidden, `{"errors":[{"code":399,"message":"Wrong password!"}]}`), nil
		default:
			t.Fatalf("unexpected twitter login request #%d: %s %s", idx, req.Method, req.URL.String())
			return nil, nil
		}
	}

	registrar := NewTwitterAccountCredentialRegistrar()
	registrar.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://proxy.local:8080", proxyURL)
		return fake, nil
	}

	_, err := registrar.AcquireCredentials(context.Background(), SocialAccountCredentialRequest{
		Name:          "northwind_ops",
		Password:      "wrong-password",
		ProxyEndpoint: "http://proxy.local:8080",
	})

	require.Error(t, err)
	require.Equal(t, "twitter error 399: Wrong password!", err.Error())
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailurePasswordInvalid, kind)
	require.Equal(t, "密码错误，本次未扣费", safeSocialTaskFailureMessage(err))
	require.Len(t, fake.calls, 3)
}

func TestTwitterAccountCredentialRegistrarMissingAccountUsesAccountNotFoundMessage(t *testing.T) {
	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		switch idx {
		case 0:
			return twitterLoginJSONResponse(http.StatusOK, `{"guest_token":"guest-token"}`), nil
		case 1:
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-1","subtasks":[{"subtask_id":"LoginEnterUserIdentifier","enter_text":{"error_message":"Sorry, we could not find your account."}}]}`), nil
		default:
			t.Fatalf("unexpected twitter login request #%d: %s %s", idx, req.Method, req.URL.String())
			return nil, nil
		}
	}

	registrar := NewTwitterAccountCredentialRegistrar()
	registrar.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://proxy.local:8080", proxyURL)
		return fake, nil
	}

	_, err := registrar.AcquireCredentials(context.Background(), SocialAccountCredentialRequest{
		Name:          "missing_account",
		Password:      "pass",
		ProxyEndpoint: "http://proxy.local:8080",
	})

	require.Error(t, err)
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureAuthInvalid, kind)
	require.Equal(t, "账号不存在，本次未扣费", safeSocialTaskFailureMessage(err))
	require.Len(t, fake.calls, 2)
}

func TestTwitterAccountCredentialRegistrarFailsClosedWhen2FASecretMissing(t *testing.T) {
	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		require.Equal(t, 0, idx)
		return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow","subtasks":[{"subtask_id":"LoginTwoFactorAuthChallenge"}]}`), nil
	}
	session := &twitterLoginSession{
		httpClient: fake,
		endpoints:  defaultTwitterLoginEndpoints(),
		auth:       defaultTwitterAuthHeaders(),
		account:    SocialAccountCredentialRequest{Name: "northwind"},
	}

	err := session.fetchOnboardingTasks(context.Background(), nil, 0, "")

	require.Error(t, err)
	require.ErrorContains(t, err, "requires a 2FA secret")
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureChallengeRequired, kind)
}

type fakeSocialAccountCredentialRegistrar struct {
	req SocialAccountCredentialRequest
	res *SocialAccountCredentialResult
	err error
}

func (r *fakeSocialAccountCredentialRegistrar) Supports(platform string) bool {
	return isTwitterPlatform(platform)
}

func (r *fakeSocialAccountCredentialRegistrar) AcquireCredentials(ctx context.Context, req SocialAccountCredentialRequest) (*SocialAccountCredentialResult, error) {
	r.req = req
	return r.res, r.err
}

func TestSocialAccountServiceRegisterWithCredentialsPersistsExecutionAuthAndAuthCookie(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountServiceWithCredentialEncryptor(client, executionAuthEncryptorStub{})
	password := "p@ssw0rd"
	email := "northwind@example.com"
	emailPassword := "email-secret"
	twoFactor := "JBSWY3DPEHPK3PXP"
	emailToken := "refresh-token"
	emailClientID := "client-id"
	platformUserID := "12345"
	registrar := &fakeSocialAccountCredentialRegistrar{res: &SocialAccountCredentialResult{
		ExecutionAuth:  `{"access_token":"oauth-token","token_secret":"oauth-secret","screen_name":"northwind_ops"}`,
		AuthCookie:     `{"access_token":"oauth-token","token_secret":"oauth-secret","screen_name":"northwind_ops","guest_token":"guest-token","client_uuid":"client-uuid"}`,
		PlatformUserID: &platformUserID,
		Message:        "login succeeded",
	}}

	account, err := svc.RegisterWithCredentials(ctx, &RegisterSocialAccountInput{
		Name:          "@northwind_ops",
		Platform:      "x_twitter",
		Password:      &password,
		Email:         &email,
		EmailPassword: &emailPassword,
		TwoFactor:     &twoFactor,
		EmailToken:    &emailToken,
		EmailClientID: &emailClientID,
	}, registrar, "http://proxy.local:8080")

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, SocialAccountStatusAvailable, account.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, account.TaskStatus)
	require.Equal(t, "login succeeded", requireStringPtr(t, account.TaskMessage))
	require.NotContains(t, requireStringPtr(t, account.ExecutionAuth), "oauth-token")
	require.NotContains(t, requireStringPtr(t, account.ExecutionAuth), "oauth-secret")
	require.Equal(t, `{"access_token":"oauth-token","token_secret":"oauth-secret","screen_name":"northwind_ops","guest_token":"guest-token","client_uuid":"client-uuid"}`, requireStringPtr(t, account.AuthCookie))
	require.NotEqual(t, requireStringPtr(t, account.AuthCookie), requireStringPtr(t, account.ExecutionAuth))
	require.Equal(t, platformUserID, requireStringPtr(t, account.PlatformUserID))

	require.Equal(t, "@northwind_ops", registrar.req.Name)
	require.Equal(t, password, registrar.req.Password)
	require.Equal(t, email, registrar.req.Email)
	require.Equal(t, emailPassword, registrar.req.EmailPassword)
	require.Equal(t, twoFactor, registrar.req.TwoFactor)
	require.Equal(t, emailToken, registrar.req.EmailToken)
	require.Equal(t, emailClientID, registrar.req.EmailClientID)
	require.Equal(t, "http://proxy.local:8080", registrar.req.ProxyEndpoint)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	decryptedExecutionAuth, err := decryptTwitterExecutionAuthCiphertext(requireStringPtr(t, stored.ExecutionAuth), executionAuthEncryptorStub{})
	require.NoError(t, err)
	require.Equal(t, `{"access_token":"oauth-token","token_secret":"oauth-secret","screen_name":"northwind_ops"}`, decryptedExecutionAuth)
	require.Equal(t, `{"access_token":"oauth-token","token_secret":"oauth-secret","screen_name":"northwind_ops","guest_token":"guest-token","client_uuid":"client-uuid"}`, requireStringPtr(t, stored.AuthCookie))
	require.Equal(t, twoFactor, requireStringPtr(t, stored.TwoFactor))
	require.Equal(t, emailToken, requireStringPtr(t, stored.EmailToken))
	require.Equal(t, emailClientID, requireStringPtr(t, stored.EmailClientID))
}

func TestTwitterExecutionAuthRejectsFullLoginBackupShape(t *testing.T) {
	fullLoginBackup := `{"access_token":"oauth-token","token_secret":"oauth-secret","screen_name":"northwind_ops","guest_token":"guest-token","client_uuid":"client-uuid","user_agent":"UA"}`

	_, err := normalizeTwitterExecutionAuthForStorage(fullLoginBackup, "northwind_ops")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSocialAccountExecutionAuthInvalid)
	require.Equal(t, "SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID", infraerrors.Reason(err))

	_, err = twitterAuthHeadersFromExecutionAuth(fullLoginBackup)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported shape")
}
