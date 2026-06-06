package repository

import (
	"strings"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/redis/go-redis/v9"
)

type dashboardCache struct {
	rdb       *redis.Client
	keyPrefix string
}

func NewDashboardCache(rdb *redis.Client, cfg *config.Config) service.DashboardStatsCache {
	prefix := "socialops:"
	if cfg != nil {
		prefix = strings.TrimSpace(cfg.Dashboard.KeyPrefix)
	}
	if prefix != "" && !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}
	return &dashboardCache{rdb: rdb, keyPrefix: prefix}
}
