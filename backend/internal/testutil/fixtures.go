//go:build unit

package testutil

import (
	"time"

	"github.com/Wei-Shaw/socialops/internal/service"
)

// NewTestUser 创建一个可用的测试用户，可通过 opts 覆盖默认值。
func NewTestUser(opts ...func(*service.User)) *service.User {
	u := &service.User{
		ID:          1,
		Email:       "test@example.com",
		Username:    "testuser",
		Role:        "user",
		Balance:     100.0,
		Concurrency: 5,
		Status:      service.StatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// NewTestGroup 创建一个可用的测试分组，可通过 opts 覆盖默认值。
func NewTestGroup(opts ...func(*service.Group)) *service.Group {
	g := &service.Group{
		ID:       1,
		Platform: "social",
		Status:   service.StatusActive,
		Hydrated: true,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}
