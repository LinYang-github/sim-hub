package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/liny/sim-hub/pkg/logger"
)

// RequestIDMiddleware 注入请求 ID 到 Context 和响应头
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取请求头中的 RequestID，没有则生成新的
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 2. 将 RequestID 注入 Gin Context (用于路由处理器)
		c.Set(string(logger.TraceIDKey), requestID)

		// 3. 将 RequestID 注入 Go Context (用于 slog.Handler)
		// 注意：gin.Context.Request.Context() 返回的是子 context
		ctx := logger.WithTraceID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// 4. 设置响应头
		c.Header("X-Request-ID", requestID)

		slog.Log(ctx, slog.LevelInfo, "开始处理请求", "method", c.Request.Method, "path", c.Request.URL.Path, "ip", c.ClientIP())

		c.Next()

		slog.Log(ctx, slog.LevelInfo, "请求处理完成", "status", c.Writer.Status())
	}
}
