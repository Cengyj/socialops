//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

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
			return twitterLoginJSONResponse(http.StatusOK, `{"status":"success"}`), nil
		case 7:
			require.Equal(t, http.MethodGet, req.Method)
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

	var auth twitterAuthHeaders
	require.NoError(t, json.Unmarshal([]byte(result.ExecutionAuth), &auth))
	require.Equal(t, "guest-token", auth.GuestToken)
	require.Equal(t, "oauth-token", auth.AccessToken)
	require.Equal(t, "oauth-secret", auth.TokenSecret)
	require.Equal(t, "kdt-token", auth.Kdt)
	require.Equal(t, "att-token", auth.Att)
	require.Equal(t, "northwind_ops", auth.ScreenName)
}

func TestTwitterAccountCredentialRegistrarFailsClosedWhen2FASecretMissing(t *testing.T) {
	session := &twitterLoginSession{account: SocialAccountCredentialRequest{Name: "northwind"}}
	_, _, err := session.nextLoginForm(&twitterLoginResponse{
		FlowToken: "flow",
		Subtasks:  []twitterLoginSubtask{{SubtaskID: "LoginTwoFactorAuthChallenge"}},
	}, http.Header{})
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
	svc := NewSocialAccountService(client)
	password := "p@ssw0rd"
	email := "northwind@example.com"
	emailPassword := "email-secret"
	twoFactor := "JBSWY3DPEHPK3PXP"
	emailToken := "refresh-token"
	emailClientID := "client-id"
	platformUserID := "12345"
	registrar := &fakeSocialAccountCredentialRegistrar{res: &SocialAccountCredentialResult{
		ExecutionAuth:  `{"access_token":"oauth-token","token_secret":"oauth-secret"}`,
		AuthCookie:     `{"access_token":"oauth-token","token_secret":"oauth-secret"}`,
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
	require.Equal(t, `{"access_token":"oauth-token","token_secret":"oauth-secret"}`, requireStringPtr(t, account.ExecutionAuth))
	require.Equal(t, `{"access_token":"oauth-token","token_secret":"oauth-secret"}`, requireStringPtr(t, account.AuthCookie))
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
	require.Equal(t, `{"access_token":"oauth-token","token_secret":"oauth-secret"}`, requireStringPtr(t, stored.ExecutionAuth))
	require.Equal(t, `{"access_token":"oauth-token","token_secret":"oauth-secret"}`, requireStringPtr(t, stored.AuthCookie))
	require.Equal(t, twoFactor, requireStringPtr(t, stored.TwoFactor))
	require.Equal(t, emailToken, requireStringPtr(t, stored.EmailToken))
	require.Equal(t, emailClientID, requireStringPtr(t, stored.EmailClientID))
}
