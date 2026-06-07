//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPDeviceParamProviderFetchesFingerprint(t *testing.T) {
	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Contains(t, body, `"collection":"twitter_parameter"`)
		return twitterLoginJSONResponse(http.StatusOK, `{"code":200,"success":true,"data":{"client_uuid":"uuid-1","client_device_id":"dev-1","attest_token":"attest-1","user_agent":"UA-1"}}`), nil
	}
	provider := NewHTTPDeviceParamProvider(TwitterDeviceParamConfig{URL: "http://fingerprint.local/api"})
	provider.httpClient = fake

	headers, err := provider.Fetch(context.Background())
	require.NoError(t, err)
	require.NotNil(t, headers)
	require.Equal(t, "uuid-1", headers.ClientUuid)
	require.Equal(t, "dev-1", headers.ClientDeviceId)
	require.Equal(t, "attest-1", headers.AttestToken)
	require.Equal(t, "UA-1", headers.UserAgent)
}

func TestHTTPDeviceParamProviderFailsClosedWhenUnconfigured(t *testing.T) {
	provider := NewHTTPDeviceParamProvider(TwitterDeviceParamConfig{})
	_, err := provider.Fetch(context.Background())
	require.Error(t, err)
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureUnsupported, kind)
}

func TestHTTPDeviceParamProviderFailsClosedOnBusinessError(t *testing.T) {
	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		return twitterLoginJSONResponse(http.StatusOK, `{"code":500,"success":false,"message":"no params"}`), nil
	}
	provider := NewHTTPDeviceParamProvider(TwitterDeviceParamConfig{URL: "http://fingerprint.local/api", MaxRetries: 1})
	provider.httpClient = fake
	_, err := provider.Fetch(context.Background())
	require.Error(t, err)
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureNetwork, kind)
}

// TestRegistrarSeedsDeviceFingerprint verifies the acquired execution auth
// carries the real device fingerprint supplied by the provider, not the static
// defaults.
func TestRegistrarSeedsDeviceFingerprint(t *testing.T) {
	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		switch idx {
		case 0:
			return twitterLoginJSONResponse(http.StatusOK, `{"guest_token":"guest-token"}`), nil
		case 1:
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-1","subtasks":[{"subtask_id":"LoginEnterPassword"}]}`), nil
		case 2:
			return twitterLoginJSONResponse(http.StatusOK, `{"flow_token":"flow-2","subtasks":[{"subtask_id":"LoginSuccessSubtask","open_account":{"oauth_token":"oauth-token","oauth_token_secret":"oauth-secret"}}]}`), nil
		case 3:
			return twitterLoginJSONResponse(http.StatusOK, `{"status":"success"}`), nil
		case 4:
			return twitterLoginJSONResponse(http.StatusOK, `{"data":{"viewer":{"userResult":{"result":{"rest_id":"99"}}}}}`), nil
		default:
			t.Fatalf("unexpected request #%d", idx)
			return nil, nil
		}
	}
	var fetched atomic.Int32
	registrar := NewTwitterAccountCredentialRegistrar().
		WithDeviceParamProvider(deviceParamProviderFunc(func(ctx context.Context) (*twitterAuthHeaders, error) {
			fetched.Add(1)
			h := defaultTwitterAuthHeaders()
			h.ClientUuid = "real-uuid"
			h.ClientDeviceId = "real-device"
			return h, nil
		}))
	registrar.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) { return fake, nil }

	result, err := registrar.AcquireCredentials(context.Background(), SocialAccountCredentialRequest{
		Name:          "northwind_ops",
		Password:      "p@ssw0rd",
		ProxyEndpoint: "http://proxy.local:8080",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int32(1), fetched.Load())
	var auth twitterAuthHeaders
	require.NoError(t, json.Unmarshal([]byte(result.ExecutionAuth), &auth))
	require.Equal(t, "real-uuid", auth.ClientUuid)
	require.Equal(t, "real-device", auth.ClientDeviceId)
}

// TestRegistrarFailsClosedWhenDeviceProviderFails verifies that when a device
// provider is configured but fails, login fails closed instead of silently
// falling back to the static fingerprint.
func TestRegistrarFailsClosedWhenDeviceProviderFails(t *testing.T) {
	registrar := NewTwitterAccountCredentialRegistrar().
		WithDeviceParamProvider(deviceParamProviderFunc(func(ctx context.Context) (*twitterAuthHeaders, error) {
			return nil, newSocialExecutionError(SocialExecutionFailureNetwork, "fingerprint unavailable", nil)
		}))
	registrar.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		t.Fatal("login must not start without a device fingerprint")
		return nil, nil
	}
	_, err := registrar.AcquireCredentials(context.Background(), SocialAccountCredentialRequest{
		Name:          "northwind_ops",
		Password:      "p@ssw0rd",
		ProxyEndpoint: "http://proxy.local:8080",
	})
	require.Error(t, err)
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureNetwork, kind)
}

func TestHTTPEmailCodeResolverReturnsCode(t *testing.T) {
	var calls int32
	fake := &twitterLoginFakeClient{t: t}
	fake.handler = func(req *http.Request, body string, idx int) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		require.Contains(t, body, `"email":"northwind@example.com"`)
		require.Contains(t, body, `"operation":"login"`)
		return twitterLoginJSONResponse(http.StatusOK, `{"success":true,"verify_code":"246810"}`), nil
	}
	concrete := &httpEmailCodeResolver{
		cfg:        TwitterEmailCodeConfig{URL: "http://mail.local/code", MaxAttempts: 3},
		httpClient: fake,
		sleep:      func(d time.Duration) {},
	}
	code, err := concrete.resolve(context.Background(), SocialAccountCredentialRequest{
		Email:         "northwind@example.com",
		EmailPassword: "secret",
		EmailToken:    "refresh",
		EmailClientID: "client",
	})
	require.NoError(t, err)
	require.Equal(t, "246810", code)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestHTTPEmailCodeResolverFailsClosedWhenUnconfigured(t *testing.T) {
	resolver := NewHTTPEmailCodeResolver(TwitterEmailCodeConfig{})
	_, err := resolver(context.Background(), SocialAccountCredentialRequest{Email: "a@b.com"})
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "not configured")
}

type deviceParamProviderFunc func(ctx context.Context) (*twitterAuthHeaders, error)

func (f deviceParamProviderFunc) Fetch(ctx context.Context) (*twitterAuthHeaders, error) {
	return f(ctx)
}
