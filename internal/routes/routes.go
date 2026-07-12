package routes

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"slite/internal/database"
	"slite/internal/global"
	"slite/internal/handlers"
	"slite/internal/middleware"
	"slite/internal/models"
	"slite/internal/utils"
)

// RegisterHandlers 注册所有路由
func RegisterHandlers(r *gin.Engine) {
	r.Static("/static", "./static")
	r.Static("/dist", "./dist")

	r.POST("/api/auth/admin", handlers.HandleAdminLogin)
	r.POST("/api/auth/view", handlers.HandleViewLogin)

	r.GET("/ws", handlers.ServeWebSocket)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/monitor")
	})

	r.GET("/monitor", handlers.HandleMonitorPage)
	r.GET("/monitor/", handlers.HandleMonitorPage)
	r.GET("/monitor/login.html", handlers.HandleMonitorLoginPage)

	r.GET("/admin", handlers.HandleAdminPage)
	r.GET("/admin/", handlers.HandleAdminPage)
	r.GET("/admin/login", handlers.HandleAdminLoginPage)

	r.GET("/install.sh", handleInstallScript)

	r.GET("/api/config/public", func(c *gin.Context) {
		var config models.SiteConfig
		var settings models.GlobalSetting
		global.DB.First(&config, 1)
		global.DB.First(&settings, 1)
		c.JSON(http.StatusOK, gin.H{
			"site_name":         config.SiteName,
			"site_slogan":       config.SiteSlogan,
			"site_logo":         config.SiteLogo,
			"site_footer":       config.SiteFooter,
			"version":           config.Version,
			"view_auth_enabled": settings.ViewAuthEnabled,
		})
	})

	r.GET("/api/config/public/settings", func(c *gin.Context) {
		var settings models.GlobalSetting
		if err := global.DB.First(&settings, 1).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{
				"admin_theme":   "default",
				"monitor_theme": "default",
				"theme_mode":    "fixed",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"admin_theme":   settings.AdminTheme,
			"monitor_theme": settings.MonitorTheme,
			"theme_mode":    settings.ThemeMode,
		})
	})

	r.GET("/api/theme/config", handlers.GetThemeConfig)

	r.POST("/api/report", handlers.HandleAgentReport)

	r.GET("/api/agent/tasks", handlers.HandleAgentTasks)

	clientsGroup := r.Group("/api/clients")
	{
		clientsGroup.GET("/report", handlers.HandleAgentReport)
		clientsGroup.POST("/report", handlers.HandleAgentReport)
		clientsGroup.POST("/uploadBasicInfo", handlers.HandleAgentUploadBasicInfo)
		clientsGroup.GET("/tasks", handlers.HandleAgentTasks)
	}

	r.GET("/ws/stats", handlers.ServeWS)

	r.GET("/api/config/settings", func(c *gin.Context) {
		var settings models.GlobalSetting
		if err := global.DB.First(&settings, 1).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{
				"show_docker":           true,
				"show_net_delay":        true,
				"show_heatmap":          true,
				"node_refresh_interval": 5,
				"view_auth_enabled":     false,
			})
			return
		}
		c.JSON(http.StatusOK, settings)
	})

	protected := r.Group("/api")
	protected.Use(middleware.ViewAuthMiddleware())
	{
		protected.GET("/stats", func(c *gin.Context) {
			global.NodesMu.RLock()
			defer global.NodesMu.RUnlock()

			isAdmin := false
			adminToken := c.GetHeader("X-Admin-Token")

			if adminToken != "" {
				var settings models.GlobalSetting
				if err := global.DB.First(&settings, 1).Error; err == nil {
					if adminToken == strings.TrimSpace(settings.AdminAuthKey) {
						isAdmin = true
					}
				}
			}

			var groups []models.NodeGroup
			global.DB.Order("sort_index ASC, id ASC").Find(&groups)

			var configs []models.NodeConfig
			global.DB.Order("sort_index ASC, node_id ASC").Find(&configs)

			configMap := make(map[string]models.NodeConfig)
			sortedIDs := make([]string, 0)
			for _, cfg := range configs {
				configMap[cfg.ID] = cfg
				sortedIDs = append(sortedIDs, cfg.ID)
			}

			var list []*models.Node
			for _, id := range sortedIDs {
				v, exists := global.Nodes[id]
				var displayNode models.Node
				if exists {
					displayNode = *v
				} else {
					displayNode = models.Node{ID: id, Online: false}
				}

				if cfg, ok := configMap[id]; ok {
					if !isAdmin && !cfg.IsVisible {
						continue
					}

					displayNode.GroupID = cfg.GroupID
					if cfg.CustomName != "" {
						displayNode.Name = cfg.CustomName
					}
					if cfg.Location != "" {
						displayNode.Location = cfg.Location
					}
					if cfg.IP != "" && cfg.IP != "0.0.0.0" {
						displayNode.IP = cfg.IP
					}
					displayNode.SortIndex = cfg.SortIndex
					displayNode.MonthUp = cfg.MonthUp
					displayNode.MonthDown = cfg.MonthDown
					displayNode.TotalStatsDays = cfg.TotalStatsDays
					displayNode.IsVisible = cfg.IsVisible
				}
				list = append(list, &displayNode)
			}
			c.JSON(http.StatusOK, gin.H{
				"groups": groups,
				"nodes":  list,
			})
		})

		protected.GET("/task/history/timerange", handlers.GetTaskHistoryTimeRange)
		protected.GET("/task/history", handlers.GetTaskHistoryData)
	}

	admin := r.Group("/api/admin")
	admin.Use(middleware.AdminAuthMiddleware())
	{
		// 主题管理
		themeGroup := admin.Group("/theme")
		{
			themeGroup.GET("/config", handlers.GetThemeConfig)
			themeGroup.POST("/config", handlers.SaveThemeConfig)
			themeGroup.GET("/list", handlers.GetThemeList)
			themeGroup.POST("/switch", handlers.SwitchTheme)
			themeGroup.GET("/validate", handlers.ValidateTheme)
		}

		// 节点分组
		admin.GET("/groups", handlers.GetNodeGroups)
		admin.POST("/groups", handlers.CreateNodeGroup)
		admin.PUT("/groups/:id", handlers.UpdateNodeGroup)
		admin.DELETE("/groups/:id", handlers.DeleteNodeGroup)

		// 日志
		admin.GET("/logs", func(c *gin.Context) {
			logType := c.DefaultQuery("type", "all")
			query := c.Query("query")
			logs := database.GetSystemLogs(logType, query)
			c.JSON(http.StatusOK, logs)
		})

		admin.DELETE("/logs", func(c *gin.Context) {
			global.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.SystemLog{})
			global.AddSystemLog(models.LogTypeAudit, "warn", "管理员清空了所有系统历史日志")
			c.JSON(http.StatusOK, gin.H{"status": "success"})
		})

		// 站点配置
		admin.GET("/config", func(c *gin.Context) {
			var config models.SiteConfig
			var settings models.GlobalSetting
			global.DB.First(&config, 1)
			global.DB.First(&settings, 1)

			c.JSON(http.StatusOK, gin.H{
				"id":                config.ID,
				"site_name":         config.SiteName,
				"site_slogan":       config.SiteSlogan,
				"site_logo":         config.SiteLogo,
				"site_footer":       config.SiteFooter,
				"version":           config.Version,
				"admin_auth_key":    strings.TrimSpace(settings.AdminAuthKey),
				"view_auth_enabled": settings.ViewAuthEnabled,
				"view_auth_key":     strings.TrimSpace(settings.ViewAuthKey),
			})
		})

		admin.POST("/config", func(c *gin.Context) {
			var incoming map[string]interface{}
			if err := c.ShouldBindJSON(&incoming); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "格式错误"})
				return
			}

			var settings models.GlobalSetting
			if err := global.DB.First(&settings, 1).Error; err != nil {
				global.DB.Create(&models.GlobalSetting{ID: 1})
				global.DB.First(&settings, 1)
			}
			if val, ok := incoming["admin_auth_key"]; ok {
				newKey := strings.TrimSpace(val.(string))
				if newKey != "" {
					global.DB.Model(&settings).Update("AdminAuthKey", newKey)
					global.AddSystemLog(models.LogTypeAudit, "info", "管理员重置了后台访问密钥")
				}
			}
			if val, ok := incoming["view_auth_enabled"]; ok {
				global.DB.Model(&settings).Update("ViewAuthEnabled", val)
			}
			if val, ok := incoming["view_auth_key"]; ok {
				global.DB.Model(&settings).Update("ViewAuthKey", strings.TrimSpace(val.(string)))
			}

			var config models.SiteConfig
			if err := global.DB.First(&config, 1).Error; err != nil {
				global.DB.Create(&models.SiteConfig{ID: 1})
				global.DB.First(&config, 1)
			}

			siteConfigFields := []string{"site_name", "site_slogan", "site_logo", "site_footer", "version"}
			siteUpdates := make(map[string]interface{})

			for _, field := range siteConfigFields {
				if val, ok := incoming[field]; ok {
					siteUpdates[field] = val
				}
			}

			if len(siteUpdates) > 0 {
				global.DB.Model(&config).Updates(siteUpdates)
			}

			global.AddSystemLog(models.LogTypeAudit, "info", "管理员修改了站点全局外观配置")
			c.JSON(http.StatusOK, gin.H{"status": "success", "message": "配置已保存"})
		})

		admin.POST("/settings", func(c *gin.Context) {
			var incoming map[string]interface{}
			if err := c.ShouldBindJSON(&incoming); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "解析失败"})
				return
			}
			var settings models.GlobalSetting
			if err := global.DB.First(&settings, 1).Error; err != nil {
				global.DB.Create(&models.GlobalSetting{ID: 1})
				global.DB.First(&settings, 1)
			}

			globalSettingFields := []string{
				"show_docker", "show_net_delay", "show_heatmap",
				"node_refresh_interval", "task_show_count", "max_log_count",
				"view_auth_enabled", "view_auth_key", "admin_auth_key",
				"admin_theme", "monitor_theme", "theme_mode",
			}

			updates := make(map[string]interface{})
			for _, field := range globalSettingFields {
				if val, ok := incoming[field]; ok {
					updates[field] = val
				}
			}

			delete(updates, "site_name")
			delete(updates, "site_slogan")
			delete(updates, "site_logo")
			delete(updates, "site_footer")
			delete(updates, "version")

			if len(updates) > 0 {
				global.DB.Model(&settings).Updates(updates)
			}

			global.AddSystemLog(models.LogTypeAudit, "info", "管理员更新了系统全局显示设置")
			c.JSON(http.StatusOK, gin.H{"status": "success", "message": "设置已同步"})
		})

		// 告警
		alertGroup := admin.Group("/alert")
		{
			alertGroup.GET("/config", handlers.GetAlertConfig)
			alertGroup.POST("/config", handlers.SaveAlertConfig)
			alertGroup.POST("/test", handlers.TestAlertPush)
		}

		// 备份
		backupGroup := admin.Group("/backup")
		{
			backupGroup.GET("/config", handlers.GetBackupConfig)
			backupGroup.POST("/config", handlers.SaveBackupConfig)
			backupGroup.POST("/config/reset", handlers.ResetBackupConfig)
			backupGroup.GET("/files", handlers.GetBackupFiles)
			backupGroup.POST("/create", handlers.CreateBackup)
			backupGroup.POST("/verify/:id", handlers.VerifyBackup)
			backupGroup.GET("/download/:id", handlers.DownloadBackup)
			backupGroup.DELETE("/delete/:id", handlers.DeleteBackup)
			backupGroup.POST("/restore", handlers.RestoreDatabase)
			backupGroup.GET("/restore/progress/:id", handlers.GetRestoreProgress)
			backupGroup.GET("/export/:format", handlers.ExportData)
			backupGroup.GET("/status", handlers.GetDatabaseStatus)
			backupGroup.POST("/toggle", handlers.ToggleBackupEngine)
			backupGroup.GET("/logs", handlers.GetBackupLogs)
			backupGroup.POST("/cleanup", handlers.CleanupExpiredBackups)
		}

		// 节点管理
		admin.GET("/nodes/alert-list", handlers.GetNodesForAlert)

		admin.GET("/nodes/next-id", func(c *gin.Context) {
			newID := utils.GenerateSnowflakeID()
			c.JSON(http.StatusOK, gin.H{"id": newID})
		})

		admin.POST("/nodes/update", func(c *gin.Context) {
			var input struct {
				ID        string `json:"id"`
				GroupID   uint   `json:"group_id"`
				Name      string `json:"name"`
				Location  string `json:"location"`
				IP        string `json:"ip"`
				IsVisible bool   `json:"is_visible"`
			}
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "参数格式异常"})
				return
			}

			if input.GroupID == 0 {
				input.GroupID = 1
			}

			var conf models.NodeConfig
			if global.DB.Where("node_id = ?", input.ID).First(&conf).Error != nil {
				var maxSort int
				global.DB.Model(&models.NodeConfig{}).Select("COALESCE(MAX(sort_index), 0)").Scan(&maxSort)

				global.DB.Create(&models.NodeConfig{
					ID:             input.ID,
					GroupID:        input.GroupID,
					CustomName:     input.Name,
					Location:       input.Location,
					IP:             input.IP,
					IsVisible:      input.IsVisible,
					UptimeRate:     100.0,
					SortIndex:      maxSort + 1,
					TotalStatsDays: 1,
				})
				global.AddSystemLog(models.LogTypeAudit, "info", fmt.Sprintf("管理员手动添加新节点: %s", input.Name))
			} else {
				global.DB.Model(&models.NodeConfig{}).Where("node_id = ?", input.ID).Updates(map[string]interface{}{
					"group_id":    input.GroupID,
					"custom_name": input.Name,
					"location":    input.Location,
					"ip":          input.IP,
					"is_visible":  input.IsVisible,
				})
				global.AddSystemLog(models.LogTypeAudit, "info", fmt.Sprintf("管理员更新了节点 [%s] 的配置信息", input.Name))
			}

			global.NodesMu.Lock()
			if node, ok := global.Nodes[input.ID]; ok {
				node.GroupID = input.GroupID
				node.Name = input.Name
				node.Location = input.Location
				if input.IP != "" && input.IP != "0.0.0.0" {
					node.IP = input.IP
				}
			}
			global.NodesMu.Unlock()

			c.JSON(http.StatusOK, gin.H{"status": "success"})
		})

		admin.POST("/nodes/reorder", func(c *gin.Context) {
			var input struct {
				IDs []string `json:"ids"`
			}
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error"})
				return
			}
			for index, id := range input.IDs {
				global.DB.Model(&models.NodeConfig{}).Where("node_id = ?", id).Update("sort_index", index)
			}

			global.NodesMu.Lock()
			for index, id := range input.IDs {
				if n, ok := global.Nodes[id]; ok {
					n.SortIndex = index
				}
			}
			global.NodesMu.Unlock()

			c.JSON(http.StatusOK, gin.H{"status": "success"})
		})

		admin.DELETE("/nodes/delete/:id", func(c *gin.Context) {
			id := c.Param("id")
			global.NodesMu.Lock()
			delete(global.Nodes, id)
			global.NodesMu.Unlock()
			global.DB.Where("node_id = ?", id).Delete(&models.NodeConfig{})
			global.DB.Where("node_id = ?", id).Delete(&models.UptimeDaily{})
			global.DB.Where("node_id = ?", id).Delete(&models.TaskHistory{})
			global.AddSystemLog(models.LogTypeAudit, "warn", fmt.Sprintf("管理员删除了节点: %s", id))
			c.JSON(http.StatusOK, gin.H{"status": "success"})
		})

		// 任务管理
		admin.GET("/tasks", handlers.GetTasks)
		admin.POST("/tasks", handlers.CreateTask)
		admin.POST("/tasks/reorder", handlers.ReorderTasks)
		admin.PUT("/tasks/:id", handlers.UpdateTask)
		admin.DELETE("/tasks/:id", handlers.DeleteTask)
	}
}

// handleInstallScript 处理安装脚本请求
func handleInstallScript(c *gin.Context) {
	host := c.Request.Host
	protocol := "http"
	if c.Request.TLS != nil {
		protocol = "https"
	}
	serverURL := fmt.Sprintf("%s://%s", protocol, host)

	// 修改：从 ./scripts/install.sh 读取安装脚本
	scriptBytes, err := os.ReadFile("./scripts/install.sh")
	if err != nil {
		c.String(http.StatusInternalServerError, "服务器错误：安装脚本文件未找到")
		return
	}

	finalScript := strings.ReplaceAll(string(scriptBytes), "{{.MasterURL}}", serverURL)

	c.Header("Content-Type", "application/x-sh")
	c.String(http.StatusOK, finalScript)
}
