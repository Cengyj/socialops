package service

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/socialip"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/proxyurl"
	"golang.org/x/net/proxy"
)

// SocialIPStatus constants
const (
	SocialIPStatusOnline  = "online"
	SocialIPStatusOffline = "offline"
	SocialIPStatusUnknown = "unknown"
)

// SocialIPCheckResult holds the result of a connectivity test.
type SocialIPCheckResult struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	LatencyMs int    `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// SocialIPChecker handles IP/proxy connectivity testing.
type SocialIPChecker struct {
	entClient   *dbent.Client
	testTimeout time.Duration
	testTarget  string // target host:port to test connectivity against
}

// NewSocialIPChecker creates a new IP checker.
func NewSocialIPChecker(entClient *dbent.Client) *SocialIPChecker {
	return &SocialIPChecker{
		entClient:   entClient,
		testTimeout: 10 * time.Second,
		testTarget:  "www.google.com:443",
	}
}

// TestIP tests connectivity of a single IP/proxy and updates its status.
func (c *SocialIPChecker) TestIP(ctx context.Context, ipID int64) (*SocialIPCheckResult, error) {
	ipEnt, err := c.entClient.SocialIP.Get(ctx, int64(ipID))
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
		}
		return nil, err
	}

	result := c.checkConnectivity(ipEnt)

	if err := c.updateCheckResult(ctx, ipEnt.ID, result); err != nil {
		slog.Error("failed to update IP status", "ip_id", ipID, "error", err)
		return nil, err
	}

	return result, nil
}

// TestAllByUser tests all IPs belonging to a user.
func (c *SocialIPChecker) TestAllByUser(ctx context.Context, userID int64) ([]*SocialIPCheckResult, error) {
	ips, err := c.entClient.SocialIP.Query().
		Where(socialip.UserIDEQ(int64(userID))).
		All(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]*SocialIPCheckResult, len(ips))
	for i, ip := range ips {
		results[i] = c.checkConnectivity(ip)

		// Update DB
		if err := c.updateCheckResult(ctx, ip.ID, results[i]); err != nil {
			slog.Error("failed to update IP status", "ip_id", ip.ID, "error", err)
		}
	}

	return results, nil
}

func (c *SocialIPChecker) updateCheckResult(ctx context.Context, id int64, result *SocialIPCheckResult) error {
	update := c.entClient.SocialIP.UpdateOneID(id).
		SetStatus(result.Status).
		SetLastCheckAt(time.Now())
	if result.LatencyMs > 0 {
		update.SetLatencyMs(result.LatencyMs)
	} else {
		update.ClearLatencyMs()
	}
	_, err := update.Save(ctx)
	return err
}

func (c *SocialIPChecker) checkConnectivity(ipEnt *dbent.SocialIP) *SocialIPCheckResult {
	result := &SocialIPCheckResult{
		ID: int64(ipEnt.ID),
	}

	if ipEnt.Endpoint == nil || *ipEnt.Endpoint == "" {
		result.Status = SocialIPStatusUnknown
		result.Error = "no endpoint configured"
		return result
	}

	endpoint := *ipEnt.Endpoint
	_, parsed, err := proxyurl.Parse(endpoint)
	if err != nil {
		result.Status = SocialIPStatusOffline
		result.Error = fmt.Sprintf("invalid endpoint URL: %v", err)
		return result
	}
	if err := validateProxyEndpoint(parsed); err != nil {
		result.Status = SocialIPStatusOffline
		result.Error = err.Error()
		return result
	}

	start := time.Now()
	var connErr error

	switch parsed.Scheme {
	case "socks5", "socks5h":
		connErr = c.testSOCKS5(parsed)
	case "http", "https":
		connErr = c.testHTTPProxy(parsed)
	default:
		result.Status = SocialIPStatusUnknown
		result.Error = fmt.Sprintf("unsupported scheme: %s", parsed.Scheme)
		return result
	}

	latency := time.Since(start)
	result.LatencyMs = int(latency.Milliseconds())

	if connErr != nil {
		result.Status = SocialIPStatusOffline
		result.Error = connErr.Error()
	} else {
		result.Status = SocialIPStatusOnline
	}

	return result
}

func (c *SocialIPChecker) testSOCKS5(proxyURL *url.URL) error {
	var auth *proxy.Auth
	if proxyURL.User != nil {
		auth = &proxy.Auth{
			User: proxyURL.User.Username(),
		}
		auth.Password, _ = proxyURL.User.Password()
	}

	dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, &net.Dialer{
		Timeout: c.testTimeout,
	})
	if err != nil {
		return fmt.Errorf("SOCKS5 dialer creation failed: %w", err)
	}

	conn, err := dialer.Dial("tcp", c.testTarget)
	if err != nil {
		return fmt.Errorf("SOCKS5 connection failed: %w", err)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("SOCKS5 connection close failed: %w", err)
	}
	return nil
}

func (c *SocialIPChecker) testHTTPProxy(proxyURL *url.URL) error {
	// For HTTP proxies, test by connecting to the proxy itself
	conn, err := net.DialTimeout("tcp", proxyURL.Host, c.testTimeout)
	if err != nil {
		return fmt.Errorf("HTTP proxy connection failed: %w", err)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("HTTP proxy connection close failed: %w", err)
	}
	return nil
}

func validateProxyEndpoint(proxyURL *url.URL) error {
	if proxyURL == nil {
		return fmt.Errorf("proxy endpoint is required")
	}
	switch proxyURL.Scheme {
	case "http", "https", "socks5", "socks5h":
	default:
		return fmt.Errorf("unsupported scheme: %s", proxyURL.Scheme)
	}
	host := strings.TrimSpace(proxyURL.Hostname())
	if host == "" {
		return fmt.Errorf("proxy host is required")
	}
	port := proxyURL.Port()
	if port == "" {
		return fmt.Errorf("proxy port is required")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum <= 0 || portNum > 65535 {
		return fmt.Errorf("proxy port is invalid")
	}
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("proxy host points to a local address")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedProxyIP(ip) {
			return fmt.Errorf("proxy host points to a private or local address")
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("proxy host could not be resolved: %w", err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("proxy host did not resolve to an IP address")
	}
	for _, addr := range addrs {
		if isBlockedProxyIP(addr.IP) {
			return fmt.Errorf("proxy host resolves to a private or local address")
		}
	}
	return nil
}

func isBlockedProxyIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}
