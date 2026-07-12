package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"slite/internal/global"
	"slite/internal/models"
)

// HandleAdminPage 处理管理员页面请求
func HandleAdminPage(c *gin.Context) {
	var s models.GlobalSetting
	global.DB.First(&s, 1)

	if s.AdminTheme != "default" && s.AdminTheme != "dark" {
		s.AdminTheme = "default"
	}

	token, err := c.Cookie("slite_admin_token")
	dbKey := strings.TrimSpace(s.AdminAuthKey)

	if err != nil || strings.TrimSpace(token) != dbKey {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}

	htmlBytes, err := os.ReadFile("./static/admin/admin.html")
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
                var cssBasePath = '/static/admin/assets/css/';
                
                if (themeConfig.themeMode === 'system') {
                    var prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
                    themeName = prefersDark ? 'dark' : 'default';
                } else {
                    themeName = themeConfig.adminTheme;
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

// HandleAdminLoginPage 处理管理员登录页面请求
func HandleAdminLoginPage(c *gin.Context) {
	var s models.GlobalSetting
	if err := global.DB.First(&s, 1).Error; err != nil {
		s.AdminTheme = "default"
		s.ThemeMode = "fixed"
	}

	if s.AdminTheme != "default" && s.AdminTheme != "dark" {
		s.AdminTheme = "default"
	}

	htmlBytes, err := os.ReadFile("./static/admin/admin_login.html")
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
                    themeName = themeConfig.adminTheme;
                    if (themeName !== 'default' && themeName !== 'dark') {
                        themeName = 'default';
                    }
                }
                
                document.documentElement.setAttribute('data-theme', themeName);
                
                var themeLink = document.getElementById('theme-style');
                if (themeLink) {
                    themeLink.href = '/static/admin/assets/css/' + themeName + '.css';
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
