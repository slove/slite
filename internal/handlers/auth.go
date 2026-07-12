package handlers

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"slite/internal/global"
	"slite/internal/models"
)

// HandleAdminLogin 处理管理员登录
func HandleAdminLogin(c *gin.Context) {
	var input struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"message": "INVALID FORMAT"})
		return
	}
	var s models.GlobalSetting
	if err := global.DB.First(&s, 1).Error; err != nil {
		c.JSON(500, gin.H{"message": "DATABASE ERROR"})
		return
	}

	inputKey := strings.TrimSpace(input.Key)
	dbKey := strings.TrimSpace(s.AdminAuthKey)

	if inputKey != "" && inputKey == dbKey {
		c.SetCookie("slite_admin_token", dbKey, 604800, "/", "", false, false)
		c.SetCookie("slite_admin_theme", s.AdminTheme, 604800, "/", "", false, false)
		c.SetCookie("slite_theme_mode", s.ThemeMode, 604800, "/", "", false, false)
		global.AddSystemLog(models.LogTypeAudit, "info", fmt.Sprintf("管理员成功登录 | IP: %s", c.ClientIP()))
		c.JSON(200, gin.H{
			"token":  dbKey,
			"status": "success",
			"theme": gin.H{
				"admin_theme":   s.AdminTheme,
				"monitor_theme": s.MonitorTheme,
				"theme_mode":    s.ThemeMode,
			},
		})
	} else {
		global.AddSystemLog(models.LogTypeSecurity, "alert", fmt.Sprintf("管理员登录失败：密钥错误 | IP: %s", c.ClientIP()))
		c.JSON(401, gin.H{"message": "INVALID ADMIN KEY"})
	}
}

// HandleViewLogin 处理查看者登录
func HandleViewLogin(c *gin.Context) {
	var input struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"message": "INVALID FORMAT"})
		return
	}
	var s models.GlobalSetting
	global.DB.First(&s, 1)

	inputKey := strings.TrimSpace(input.Key)
	dbKey := strings.TrimSpace(s.ViewAuthKey)

	if inputKey != "" && inputKey == dbKey {
		c.SetCookie("slite_view_token", dbKey, 604800, "/", "", false, false)
		c.SetCookie("slite_monitor_theme", s.MonitorTheme, 604800, "/", "", false, false)
		c.SetCookie("slite_theme_mode", s.ThemeMode, 604800, "/", "", false, false)
		c.JSON(200, gin.H{"token": dbKey, "status": "success"})
	} else {
		c.JSON(401, gin.H{"message": "INVALID ACCESS KEY"})
	}
}
