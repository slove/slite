package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"slite/internal/global"
	"slite/internal/models"
)

// ServeWebSocket 处理管理端WebSocket连接
func ServeWebSocket(c *gin.Context) {
	conn, err := global.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] 管理端升级协议失败: %v", err)
		return
	}

	global.ClientsMu.Lock()
	global.Clients[conn] = true
	global.ClientsMu.Unlock()

	defer func() {
		global.ClientsMu.Lock()
		delete(global.Clients, conn)
		global.ClientsMu.Unlock()
		conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// ServeWS 处理监控页面WebSocket连接
func ServeWS(c *gin.Context) {
	var settings models.GlobalSetting
	err := global.DB.First(&settings, 1).Error

	isAdmin := false
	adminToken, _ := c.Cookie("slite_admin_token")
	if adminToken != "" && strings.TrimSpace(adminToken) == strings.TrimSpace(settings.AdminAuthKey) {
		isAdmin = true
	}

	if err == nil {
		if settings.ViewAuthEnabled && !isAdmin {
			token, err := c.Cookie("slite_view_token")
			if err != nil || token == "" {
				token = c.Query("token")
			}
			if token != strings.TrimSpace(settings.ViewAuthKey) {
				return
			}
		}
	}

	conn, err := global.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	sendUpdate := func() error {
		global.NodesMu.RLock()
		defer global.NodesMu.RUnlock()

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

		thirtyDaysAgo := time.Now().AddDate(0, 0, -31).Format("2006-01-02")
		var historyRecords []models.UptimeDaily
		global.DB.Where("date >= ?", thirtyDaysAgo).Order("date ASC").Find(&historyRecords)

		historyMap := make(map[string][]models.HeatMapItem)
		for _, h := range historyRecords {
			item := models.HeatMapItem{
				Date:            h.Date,
				Status:          h.Status,
				UptimeRate:      h.Rate,
				OfflineDuration: h.OfflineDuration,
				DisplayInfo:     fmt.Sprintf("在线率: %.1f%%", h.Rate),
			}
			historyMap[h.NodeID] = append(historyMap[h.NodeID], item)
		}

		var list []*models.Node
		for _, id := range sortedIDs {
			cfg, cfgExists := configMap[id]

			if cfgExists && !cfg.IsVisible && !isAdmin {
				continue
			}

			v, exists := global.Nodes[id]
			var displayNode models.Node

			if exists {
				displayNode = *v
			} else {
				displayNode = models.Node{
					ID:     id,
					Online: false,
				}
			}

			if cfgExists {
				displayNode.GroupID = cfg.GroupID
				if cfg.CustomName != "" {
					displayNode.Name = cfg.CustomName
				} else if !exists {
					displayNode.Name = "Unknown Node"
				}
				if cfg.Location != "" {
					displayNode.Location = cfg.Location
				}
				if cfg.IP != "" && cfg.IP != "0.0.0.0" {
					displayNode.IP = cfg.IP
				}
				displayNode.SortIndex = cfg.SortIndex
				displayNode.IsVisible = cfg.IsVisible
				displayNode.MonthUp = cfg.MonthUp
				displayNode.MonthDown = cfg.MonthDown
				displayNode.UptimeRate = cfg.UptimeRate
				displayNode.OfflineTotal = cfg.OfflineTotal
				displayNode.TotalStatsDays = cfg.TotalStatsDays
			}

			if h, ok := historyMap[id]; ok {
				displayNode.UptimeHeatMap = h
			} else {
				displayNode.UptimeHeatMap = make([]models.HeatMapItem, 0)
			}

			displayNode.Tasks = loadNodeTasks(displayNode.ID)

			if displayNode.Load == nil {
				displayNode.Load = []float64{0.0, 0.0, 0.0}
			}
			if displayNode.Docker == nil {
				displayNode.Docker = &models.DockerStatus{ContainerList: []models.ContainerDetail{}}
			}
			list = append(list, &displayNode)
		}

		payload := gin.H{
			"groups": groups,
			"nodes":  list,
			"settings": gin.H{
				"admin_theme":   settings.AdminTheme,
				"monitor_theme": settings.MonitorTheme,
				"theme_mode":    settings.ThemeMode,
			},
		}

		return conn.WriteJSON(payload)
	}

	if err := sendUpdate(); err != nil {
		return
	}

	refreshSec := settings.NodeRefreshInterval
	if refreshSec < 1 {
		refreshSec = 2
	}
	ticker := time.NewTicker(time.Duration(refreshSec) * time.Second)
	defer ticker.Stop()

	// 使用 for range 替代 for { select {} }
	for range ticker.C {
		if err := sendUpdate(); err != nil {
			return
		}
	}
}

// HandleMonitorPage 处理监控页面请求
func HandleMonitorPage(c *gin.Context) {
	var s models.GlobalSetting
	global.DB.First(&s, 1)

	if s.MonitorTheme != "default" && s.MonitorTheme != "dark" {
		s.MonitorTheme = "default"
	}

	if !s.ViewAuthEnabled {
		htmlBytes, err := os.ReadFile("./static/monitor/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "页面文件不存在")
			return
		}

		htmlContent := string(htmlBytes)
		themeScript := fmt.Sprintf(`
    <script>
        (function() {
            var themeConfig = {
                adminTheme: "%s",
                monitorTheme: "%s",
                themeMode: "%s",
                viewAuthEnabled: %v
            };
            window.__SLITE_CONFIG = themeConfig;
            
            function applyTheme() {
                var themeName = 'default';
                var cssBasePath = '/static/monitor/assets/css/';
                
                if (themeConfig.themeMode === 'system') {
                    var prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
                    themeName = prefersDark ? 'dark' : 'default';
                } else {
                    themeName = themeConfig.monitorTheme;
                    if (themeName !== 'default' && themeName !== 'dark') {
                        themeName = 'default';
                    }
                }
                
                document.documentElement.setAttribute('data-theme', themeName);
                
                var themeLink = document.getElementById('theme-style');
                if (!themeLink) {
                    themeLink = document.createElement('link');
                    themeLink.id = 'theme-style';
                    themeLink.rel = 'stylesheet';
                    document.head.appendChild(themeLink);
                }
                themeLink.href = cssBasePath + themeName + '.css';
            }
            
            applyTheme();
            
            if (themeConfig.themeMode === 'system') {
                window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', applyTheme);
            }
        })();
    </script>
`, s.AdminTheme, s.MonitorTheme, s.ThemeMode, s.ViewAuthEnabled)

		htmlContent = strings.Replace(htmlContent, "<head>", "<head>"+themeScript, 1)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, htmlContent)
		return
	}

	token, err := c.Cookie("slite_view_token")
	if err != nil || strings.TrimSpace(token) != strings.TrimSpace(s.ViewAuthKey) {
		c.Redirect(http.StatusFound, "/monitor/login.html")
		return
	}

	htmlBytes, err := os.ReadFile("./static/monitor/index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "页面文件不存在")
		return
	}

	htmlContent := string(htmlBytes)
	themeScript := fmt.Sprintf(`
    <script>
        (function() {
            var themeConfig = {
                adminTheme: "%s",
                monitorTheme: "%s",
                themeMode: "%s",
                viewAuthEnabled: %v
            };
            window.__SLITE_CONFIG = themeConfig;
            
            function applyTheme() {
                var themeName = 'default';
                var cssBasePath = '/static/monitor/assets/css/';
                
                if (themeConfig.themeMode === 'system') {
                    var prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
                    themeName = prefersDark ? 'dark' : 'default';
                } else {
                    themeName = themeConfig.monitorTheme;
                    if (themeName !== 'default' && themeName !== 'dark') {
                        themeName = 'default';
                    }
                }
                
                document.documentElement.setAttribute('data-theme', themeName);
                
                var themeLink = document.getElementById('theme-style');
                if (!themeLink) {
                    themeLink = document.createElement('link');
                    themeLink.id = 'theme-style';
                    themeLink.rel = 'stylesheet';
                    document.head.appendChild(themeLink);
                }
                themeLink.href = cssBasePath + themeName + '.css';
            }
            
            applyTheme();
            
            if (themeConfig.themeMode === 'system') {
                window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', applyTheme);
            }
        })();
    </script>
`, s.AdminTheme, s.MonitorTheme, s.ThemeMode, s.ViewAuthEnabled)

	htmlContent = strings.Replace(htmlContent, "<head>", "<head>"+themeScript, 1)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, htmlContent)
}

// HandleMonitorLoginPage 处理监控登录页面请求
func HandleMonitorLoginPage(c *gin.Context) {
	var s models.GlobalSetting
	if err := global.DB.First(&s, 1).Error; err != nil {
		s.MonitorTheme = "default"
		s.ThemeMode = "fixed"
	}

	if s.MonitorTheme != "default" && s.MonitorTheme != "dark" {
		s.MonitorTheme = "default"
	}

	htmlBytes, err := os.ReadFile("./static/monitor/login.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "页面文件不存在")
		return
	}

	htmlContent := string(htmlBytes)

	themeScript := fmt.Sprintf(`
    <script>
        (function() {
            var themeConfig = {
                adminTheme: "%s",
                monitorTheme: "%s",
                themeMode: "%s"
            };
            window.__SLITE_CONFIG = themeConfig;
            
            function applyTheme() {
                var themeName = 'default';
                
                if (themeConfig.themeMode === 'system') {
                    var prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
                    themeName = prefersDark ? 'dark' : 'default';
                } else {
                    themeName = themeConfig.monitorTheme;
                    if (themeName !== 'default' && themeName !== 'dark') {
                        themeName = 'default';
                    }
                }
                
                document.documentElement.setAttribute('data-theme', themeName);
                
                var themeLink = document.getElementById('theme-style');
                if (themeLink) {
                    themeLink.href = '/static/monitor/assets/css/' + themeName + '.css';
                }
            }
            
            applyTheme();
            
            if (themeConfig.themeMode === 'system') {
                window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', applyTheme);
            }
        })();
    </script>
`, s.AdminTheme, s.MonitorTheme, s.ThemeMode)

	htmlContent = strings.Replace(htmlContent, "<head>", "<head>"+themeScript, 1)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, htmlContent)
}
