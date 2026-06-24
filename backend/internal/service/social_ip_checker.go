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
	return c.testIP(ctx, ipID, 0)
}

// TestIPForUser tests connectivity of a single current-user proxy.
func (c *SocialIPChecker) TestIPForUser(ctx context.Context, ipID, userID int64) (*SocialIPCheckResult, error) {
	return c.testIP(ctx, ipID, userID)
}

func (c *SocialIPChecker) testIP(ctx context.Context, ipID, userID int64) (*SocialIPCheckResult, error) {
	ipQuery := c.entClient.SocialIP.Query().Where(socialip.IDEQ(ipID))
	if userID > 0 {
		ipQuery = ipQuery.Where(socialip.UserIDEQ(userID))
	}
	ipEnt, err := ipQuery.Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
		}
		return nil, err
	}

	result := c.checkConnectivity(ctx, ipEnt)

	if err := c.updateCheckResult(ctx, ipEnt.ID, userID, result); err != nil {
		slog.Error("failed to update IP status", "ip_id", ipID, "error", err)
		return nil, err
	}

	return result, nil
}

// TestAllByUser tests all IPs belonging to a user.
func (c *SocialIPChecker) TestAllByUser(ctx context.Context, userID int64) ([]*SocialIPCheckResult, error) {
	ips, err := c.entClient.SocialIP.Query().
		Where(socialip.UserIDEQ(int64(userID))).
		Order(dbent.Asc(socialip.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]*SocialIPCheckResult, len(ips))
	for i, ip := range ips {
		results[i] = c.checkConnectivity(ctx, ip)
	}

	tx, err := c.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txClient := tx.Client()
	for i, ip := range ips {
		ent, err := updateCheckResultWithClient(ctx, txClient, ip.ID, userID, results[i])
		if err != nil {
			slog.Error("failed to update IP status", "ip_id", ip.ID, "error", err)
			return nil, err
		}
		if err := syncDefaultProxySnapshotsForIP(ctx, txClient, socialIPFromEnt(ent)); err != nil {
			slog.Error("failed to sync default proxy snapshots", "ip_id", ip.ID, "error", err)
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return results, nil
}

func (c *SocialIPChecker) updateCheckResult(ctx context.Context, id, userID int64, result *SocialIPCheckResult) error {
	tx, err := c.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ent, err := updateCheckResultWithClient(ctx, tx.Client(), id, userID, result)
	if err != nil {
		return err
	}
	if err := syncDefaultProxySnapshotsForIP(ctx, tx.Client(), socialIPFromEnt(ent)); err != nil {
		return err
	}
	return tx.Commit()
}

func updateCheckResultWithClient(ctx context.Context, client *dbent.Client, id, userID int64, result *SocialIPCheckResult) (*dbent.SocialIP, error) {
	update := client.SocialIP.UpdateOneID(id).
		SetStatus(result.Status).
		SetLastCheckAt(time.Now())
	if userID > 0 {
		update.Where(socialip.UserIDEQ(userID))
	}
	if result.LatencyMs > 0 {
		update.SetLatencyMs(result.LatencyMs)
	} else {
		update.ClearLatencyMs()
	}
	ent, err := update.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_IP_NOT_FOUND", "social IP not found")
		}
		return nil, err
	}
	return ent, nil
}

func (c *SocialIPChecker) checkConnectivity(ctx context.Context, ipEnt *dbent.SocialIP) *SocialIPCheckResult {
	result := &SocialIPCheckResult{
		ID: int64(ipEnt.ID),
	}

	if ipEnt.Endpoint == nil || *ipEnt.Endpoint == "" {
		result.Status = SocialIPStatusUnknown
		result.Error = safeSocialIPCheckErrorMessage(result.Status)
		return result
	}

	endpoint, err := ResolveSocialIPExecutionEndpoint(ctx, *ipEnt.Endpoint)
	if err != nil {
		result.Status = SocialIPStatusOffline
		result.Error = safeSocialIPCheckErrorMessage(result.Status)
		return result
	}
	_, parsed, _ := proxyurl.Parse(endpoint)

	start := time.Now()
	var connErr error

	switch parsed.Scheme {
	case "socks5", "socks5h":
		connErr = c.testSOCKS5(parsed)
	case "http", "https":
		connErr = c.testHTTPProxy(parsed)
	default:
		result.Status = SocialIPStatusUnknown
		result.Error = safeSocialIPCheckErrorMessage(result.Status)
		return result
	}

	latency := time.Since(start)
	result.LatencyMs = int(latency.Milliseconds())

	if connErr != nil {
		result.Status = SocialIPStatusOffline
		result.Error = safeSocialIPCheckErrorMessage(result.Status)
	} else {
		result.Status = SocialIPStatusOnline
	}

	return result
}

func safeSocialIPCheckErrorMessage(status string) string {
	switch status {
	case SocialIPStatusOffline:
		return "proxy connectivity check failed"
	case SocialIPStatusUnknown:
		return "proxy endpoint is not ready for connectivity check"
	default:
		return "proxy connectivity check could not be completed"
	}
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
	port := proxyURL.Port()
	if port == "" {
		return fmt.Errorf("proxy port is required")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum <= 0 || portNum > 65535 {
		return fmt.Errorf("proxy port is invalid")
	}
	return validatePublicProxyHost(proxyURL)
}

func validatePublicProxyHost(proxyURL *url.URL) error {
	if proxyURL == nil {
		return fmt.Errorf("proxy host is required")
	}
	host := strings.TrimSpace(proxyURL.Hostname())
	if host == "" {
		return fmt.Errorf("proxy host is required")
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
