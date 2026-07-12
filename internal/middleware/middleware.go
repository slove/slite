package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"slite/internal/global"
	"slite/internal/models"
)

// AdminAuthMiddleware 管理后台专用鉴权中间件
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var settings models.GlobalSetting
		if err := global.DB.First(&settings, 1).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "系统配置初始化失败"})
			c.Abort()
			return
		}

		token, err := c.Cookie("slite_admin_token")
		if err != nil || token == "" {
			token = c.GetHeader("X-Admin-Token")
		}

		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" || strings.TrimSpace(token) != strings.TrimSpace(settings.AdminAuthKey) {
			global.AddSystemLog(models.LogTypeSecurity, "alert", fmt.Sprintf("非法后台访问尝试 | IP: %s", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "管理员权限验证失败",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ViewAuthMiddleware 前端访问鉴权中间件
func ViewAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var settings models.GlobalSetting
		if err := global.DB.First(&settings, 1).Error; err != nil {
			c.Next()
			return
		}

		if settings.ViewAuthEnabled {
			token, err := c.Cookie("slite_view_token")
			if err != nil || token == "" {
				token = c.GetHeader("X-View-Token")
			}
			if token == "" {
				token = c.Query("token")
			}

			if strings.TrimSpace(token) != strings.TrimSpace(settings.ViewAuthKey) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":         "Unauthorized Access",
					"auth_required": true,
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
