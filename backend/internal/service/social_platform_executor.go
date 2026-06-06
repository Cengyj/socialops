package service

import (
	"context"
	"strings"

	dbent "github.com/Wei-Shaw/socialops/ent"
)

// SocialPlatformExecutor is the adapter boundary between SocialOps tasks and a
// concrete social platform implementation.
type SocialPlatformExecutor interface {
	Execute(ctx context.Context, taskLog *dbent.SocialTaskLog, account *dbent.SocialAccount) (string, error)
}

func isTwitterPlatform(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "x_twitter", "twitter", "x":
		return true
	default:
		return false
	}
}
