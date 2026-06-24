// Package ctxkey 定义用于 context.Value 的类型安全 key
package ctxkey

// Key 定义 context key 的类型，避免使用内置 string 类型（staticcheck SA1029）
type Key string

const (
	// RequestID 为服务端生成/透传的请求 ID。
	RequestID Key = "ctx_request_id"

	// ClientRequestID 客户端请求的唯一标识，用于追踪请求全生命周期。
	ClientRequestID Key = "ctx_client_request_id"
)
