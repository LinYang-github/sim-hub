package resource

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/modules/resource/core"
)

// AuthMiddleware 鉴权中间件 (支持 JWT 与 Trusted Proxy)
func AuthMiddleware(authMgr *core.AuthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 检查是否来源于信任代理 (Sidecar Auth)
		// 注意：生产环境下需校验 c.ClientIP() 是否在 TrustedIPs 列表中
		trustedUser := c.GetHeader("X-SimHub-User")
		if trustedUser != "" {
			c.Set("user_id", trustedUser)
			c.Set("auth_mode", "proxy")
			c.Next()
			return
		}

		// 2. 备选方案：标准 Bearer Token 验证
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

		c.Set("user_id", userID)
		c.Set("auth_mode", "jwt")
		c.Next()
	}
}

// RBACMiddleware 权限控制中间件
func RBACMiddleware(authMgr *core.AuthManager, requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		uid := userID.(string)

		// 获取用户及其角色权限
		user, err := authMgr.GetUserWithRole(c.Request.Context(), uid)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Failed to fetch user permissions"})
			c.Abort()
			return
		}

		// 校验权限
		hasPermission := false
		for _, p := range user.Role.Permissions {
			if p == requiredPermission || p == "*" { // 支持 * 通配符
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Missing required permission: %s", requiredPermission)})
			c.Abort()
			return
		}

		c.Next()
	}
}
