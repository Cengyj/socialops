package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/proxyurl"
)

var dynamicProxySourceHTTPClient = &http.Client{Timeout: 10 * time.Second}

type dynamicProxySourceResponse struct {
	Data  []dynamicProxySourceItem `json:"data"`
	Final []dynamicProxySourceItem `json:"final"`
}

type dynamicProxySourceItem struct {
	IP       string          `json:"ip"`
	Port     json.RawMessage `json:"port"`
	Username string          `json:"username"`
	Password string          `json:"password"`
	Protocol string          `json:"protocol"`
	Scheme   string          `json:"scheme"`
	Type     string          `json:"type"`
}

func isDynamicProxySourceURL(proxyURL *url.URL) bool {
	if proxyURL == nil {
		return false
	}
	scheme := strings.ToLower(strings.TrimSpace(proxyURL.Scheme))
	if scheme != "http" && scheme != "https" {
		return false
	}
	path := strings.TrimSpace(proxyURL.EscapedPath())
	return proxyURL.Port() == "" || (path != "" && path != "/") || strings.TrimSpace(proxyURL.RawQuery) != ""
}

func validateDynamicProxySourceURL(sourceURL *url.URL) error {
	if sourceURL == nil {
		return fmt.Errorf("proxy source URL is required")
	}
	scheme := strings.ToLower(strings.TrimSpace(sourceURL.Scheme))
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("proxy source URL must use http or https")
	}
	if strings.TrimSpace(sourceURL.Hostname()) == "" {
		return fmt.Errorf("proxy source URL host is required")
	}
	return validatePublicProxyHost(sourceURL)
}

func ResolveSocialIPExecutionEndpoint(ctx context.Context, endpoint string) (string, error) {
	normalized, parsed, err := proxyurl.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if parsed == nil {
		return "", fmt.Errorf("proxy endpoint is required")
	}
	if isDynamicProxySourceURL(parsed) {
		if err := validateDynamicProxySourceURL(parsed); err != nil {
			return "", err
		}
		return fetchDynamicProxyEndpoint(ctx, normalized)
	}
	if err := validateProxyEndpoint(parsed); err != nil {
		return "", err
	}
	return normalized, nil
}

func fetchDynamicProxyEndpoint(ctx context.Context, sourceURL string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", fmt.Errorf("proxy source request is invalid: %w", err)
	}
	client := dynamicProxySourceHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("proxy source request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("proxy source returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("proxy source response could not be read: %w", err)
	}
	return dynamicProxyEndpointFromPayload(body)
}

func dynamicProxyEndpointFromPayload(body []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var payload dynamicProxySourceResponse
	if err := decoder.Decode(&payload); err != nil {
		return "", fmt.Errorf("proxy source returned invalid JSON: %w", err)
	}
	items := payload.Data
	if len(items) == 0 {
		items = payload.Final
	}
	if len(items) == 0 {
		return "", fmt.Errorf("proxy source returned no proxy data")
	}
	return dynamicProxyEndpointFromItem(items[0])
}

func dynamicProxyEndpointFromItem(item dynamicProxySourceItem) (string, error) {
	host := strings.TrimSpace(item.IP)
	if host == "" {
		return "", fmt.Errorf("proxy source returned an empty proxy host")
	}
	port, err := dynamicProxyPortString(item.Port)
	if err != nil {
		return "", err
	}
	scheme := dynamicProxyScheme(item)
	proxyURL := &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
	}
	username := strings.TrimSpace(item.Username)
	password := strings.TrimSpace(item.Password)
	if username != "" || password != "" {
		if password != "" {
			proxyURL.User = url.UserPassword(username, password)
		} else {
			proxyURL.User = url.User(username)
		}
	}
	normalized, parsed, err := proxyurl.Parse(proxyURL.String())
	if err != nil {
		return "", err
	}
	if err := validateProxyEndpoint(parsed); err != nil {
		return "", err
	}
	return normalized, nil
}

func dynamicProxyPortString(raw json.RawMessage) (string, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return "", fmt.Errorf("proxy source returned an empty proxy port")
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		value = strings.TrimSpace(asString)
	}
	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 || port > 65535 {
		return "", fmt.Errorf("proxy source returned an invalid proxy port")
	}
	return strconv.Itoa(port), nil
}

func dynamicProxyScheme(item dynamicProxySourceItem) string {
	for _, raw := range []string{item.Scheme, item.Protocol, item.Type} {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "http", "https", "socks5", "socks5h":
			return strings.ToLower(strings.TrimSpace(raw))
		}
	}
	return "http"
}
