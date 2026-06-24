//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type twitterRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f twitterRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTwitterAPIClientRetriesRetryableNetworkErrors(t *testing.T) {
	originalBackoff := twitterNetworkRetryBackoffDuration
	twitterNetworkRetryBackoffDuration = func(int) time.Duration { return 0 }
	t.Cleanup(func() { twitterNetworkRetryBackoffDuration = originalBackoff })

	calls := 0
	client := &twitterAPIClient{
		httpClient: &http.Client{Transport: twitterRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), `"variables"`)
			_ = req.Body.Close()

			calls++
			if calls < 3 {
				return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("temporary connection reset")}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"data":{"ok":true}}`)),
				Request:    req,
			}, nil
		})},
		auth: &twitterAuthHeaders{
			AccessToken:    "access-token",
			TokenSecret:    "token-secret",
			ClientUuid:     "client-uuid",
			ClientDeviceId: "device-id",
		},
		endpoints: twitterEndpoints{
			favoriteTweet: "https://twitter.test/graphql/FavoriteTweet",
		},
	}

	result, err := client.favoriteTweet(context.Background(), "123456789")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)
	require.Equal(t, "like succeeded", result.Message)
	require.Equal(t, 3, calls)
}

func TestTwitterAPIClientNetworkFailureUsesNetworkMessage(t *testing.T) {
	originalBackoff := twitterNetworkRetryBackoffDuration
	twitterNetworkRetryBackoffDuration = func(int) time.Duration { return 0 }
	t.Cleanup(func() { twitterNetworkRetryBackoffDuration = originalBackoff })

	calls := 0
	client := &twitterAPIClient{
		httpClient: &http.Client{Transport: twitterRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("temporary connection reset")}
		})},
		auth: &twitterAuthHeaders{
			AccessToken:    "access-token",
			TokenSecret:    "token-secret",
			ClientUuid:     "client-uuid",
			ClientDeviceId: "device-id",
		},
		endpoints: twitterEndpoints{
			favoriteTweet: "https://twitter.test/graphql/FavoriteTweet",
		},
	}

	_, err := client.favoriteTweet(context.Background(), "123456789")

	require.Error(t, err)
	require.ErrorContains(t, err, "twitter network request failed")
	require.NotContains(t, err.Error(), "check execution proxy")
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureNetwork, kind)
	require.Equal(t, twitterNetworkMaxRetries+1, calls)
}
