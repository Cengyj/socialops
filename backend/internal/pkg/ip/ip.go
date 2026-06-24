// Package ip 提供客户端 IP 地址提取工具。
package ip

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetClientIP 从 Gin Context 中提取客户端真实 IP 地址。
// 按以下优先级检查 Header：
// 1. CF-Connecting-IP (Cloudflare)
// 2. X-Real-IP (Nginx)
// 3. X-Forwarded-For (取第一个非私有 IP)
// 4. c.ClientIP() (Gin 内置方法)
func GetClientIP(c *gin.Context) string {
	// 1. Cloudflare
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return normalizeIP(ip)
	}

	// 2. Nginx X-Real-IP
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return normalizeIP(ip)
	}

	// 3. X-Forwarded-For (多个 IP 时取第一个公网 IP)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" && !isPrivateIP(ip) {
				return normalizeIP(ip)
			}
		}
		// 如果都是私有 IP，返回第一个
		if len(ips) > 0 {
			return normalizeIP(strings.TrimSpace(ips[0]))
		}
	}

	// 4. Gin 内置方法
	return normalizeIP(c.ClientIP())
}

// normalizeIP 规范化 IP 地址，去除端口号和空格。
func normalizeIP(ip string) string {
	ip = strings.TrimSpace(ip)
	// 移除端口号（如 "192.168.1.1:8080" -> "192.168.1.1"）
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

// privateNets 预编译私有 IP CIDR 块，避免每次调用 isPrivateIP 时重复解析
var privateNets []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("invalid CIDR: " + cidr)
		}
		privateNets = append(privateNets, block)
	}
}

// isPrivateIP 检查 IP 是否为私有地址。
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, block := range privateNets {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
