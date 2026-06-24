package middleware

import (
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/ip"
	"github.com/Wei-Shaw/socialops/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger 请求日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 请求路径
		path := c.Request.URL.Path

		// 处理请求
		c.Next()

		// 跳过健康检查等高频探针路径的日志
		if path == "/health" || path == "/setup/status" {
			return
		}

		endTime := time.Now()
		latency := endTime.Sub(startTime)

		method := c.Request.Method
		statusCode := c.Writer.Status()
		clientIP := ip.GetClientIP(c)
		protocol := c.Request.Proto

		fields := []zap.Field{
			zap.String("component", "http.access"),
			zap.Int("status_code", statusCode),
			zap.Int64("latency_ms", latency.Milliseconds()),
			zap.String("client_ip", clientIP),
			zap.String("protocol", protocol),
			zap.String("method", method),
			zap.String("path", path),
		}

		l := logger.FromContext(c.Request.Context()).With(fields...)
		l.Info("http request completed", zap.Time("completed_at", endTime))

		if len(c.Errors) > 0 {
			l.Warn("http request contains gin errors", zap.String("errors", c.Errors.String()))
		}
	}
}
