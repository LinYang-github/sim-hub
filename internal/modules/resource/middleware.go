package resource

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/modules/resource/core"
)

// AuthMiddleware 鉴权中间件
func AuthMiddleware(authMgr *core.AuthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be Bearer token"})
			c.Abort()
			return
		}

		token := parts[1]
		userID, err := authMgr.VerifyToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 将 UserID 注入 Context 供后续使用
		c.Set("user_id", userID)
		c.Next()
	}
}
