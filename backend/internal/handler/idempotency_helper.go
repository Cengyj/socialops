package handler

import (
	"context"
	"strconv"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/logger"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// executeUserIdempotentJSON executes a user-facing operation with idempotency support.
func executeUserIdempotentJSON(c *gin.Context, scope string, payload any, ttl time.Duration, fn func(ctx context.Context) (any, error)) {
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		result, err := fn(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, result)
		return
	}

	actorScope := "user:0"
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		actorScope = "user:" + strconv.FormatInt(subject.UserID, 10)
	}

	opts := service.IdempotencyExecuteOptions{
		Scope:          scope,
		ActorScope:     actorScope,
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Payload:        payload,
		TTL:            ttl,
		Method:         c.Request.Method,
		Route:          c.FullPath(),
		RequireKey:     true,
	}

	execResult, err := coordinator.Execute(c.Request.Context(), opts, fn)
	if err != nil {
		if infraerrors.Code(err) == infraerrors.Code(service.ErrIdempotencyStoreUnavail) {
			service.RecordIdempotencyStoreUnavailable(c.FullPath(), scope, "handler_fail_close")
			logger.LegacyPrintf("handler.idempotency", "[Idempotency] store unavailable: method=%s route=%s scope=%s strategy=fail_close", c.Request.Method, c.FullPath(), scope)
		}
		if retryAfter := service.RetryAfterSecondsFromError(err); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		response.ErrorFrom(c, err)
		return
	}
	if execResult != nil && execResult.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	response.Success(c, execResult.Data)
}
