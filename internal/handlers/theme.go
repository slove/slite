package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"slite/internal/database"
	"slite/internal/global"
	"slite/internal/models"
)

// GetThemeConfig 获取主题配置（用户选择的主题设置）
func GetThemeConfig(c *gin.Context) {
	config := database.GetThemeConfigDB()
	c.JSON(http.StatusOK, config)
}

// SaveThemeConfig 保存主题配置
func SaveThemeConfig(c *gin.Context) {
	var input models.ThemeSetting
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "INVALID FORMAT"})
		return
	}

	if input.AdminTheme == "" {
		input.AdminTheme = "default"
	}
	if input.MonitorTheme == "" {
		input.MonitorTheme = "default"
	}
	if input.ThemeMode == "" {
		input.ThemeMode = "fixed"
	}

	err := database.SaveThemeConfigDB(&input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "保存主题配置失败"})
		return
	}

	global.AddSystemLog(models.LogTypeAudit, "info", "主题配置已更新")
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GetThemeList 获取主题列表
func GetThemeList(c *gin.Context) {
	adminThemes := scanThemeFiles("admin")
	monitorThemes := scanThemeFiles("monitor")
	config := database.GetThemeConfigDB()

	response := models.ThemeListResponse{
		AdminThemes:    adminThemes,
		MonitorThemes:  monitorThemes,
		CurrentAdmin:   config.AdminTheme,
		CurrentMonitor: config.MonitorTheme,
		ThemeMode:      config.ThemeMode,
	}

	c.JSON(http.StatusOK, response)
}

// scanThemeFiles 扫描主题文件
func scanThemeFiles(area string) []models.ThemeFile {
	var themes []models.ThemeFile

	basePath := "./static"
	if area == "admin" {
		basePath = filepath.Join(basePath, "admin", "assets", "css")
	} else if area == "monitor" {
		basePath = filepath.Join(basePath, "monitor", "assets", "css")
	} else {
		return themes
	}

	files, err := os.ReadDir(basePath)
	if err != nil {
		return themes
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".css") {
			continue
		}

		var themeType string
		if strings.HasPrefix(filename, area+"-") {
			themeType = strings.TrimSuffix(strings.TrimPrefix(filename, area+"-"), ".css")
		} else if filename == "default.css" || filename == "dark.css" {
			themeType = strings.TrimSuffix(filename, ".css")
		} else {
			continue
		}

		filePath := filepath.Join(basePath, filename)
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		themes = append(themes, models.ThemeFile{
			Name:     themeType,
			Path:     filePath,
			Area:     area,
			Type:     "css",
			FileSize: fileInfo.Size(),
			Modified: fileInfo.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return themes
}

// SwitchTheme 切换主题
func SwitchTheme(c *gin.Context) {
	var input models.ThemeSwitchRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "INVALID FORMAT"})
		return
	}

	if input.Area != "admin" && input.Area != "monitor" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的主题区域"})
		return
	}

	if input.Theme == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "主题名称不能为空"})
		return
	}

	if input.Mode != "fixed" && input.Mode != "system" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的主题模式"})
		return
	}

	config := database.GetThemeConfigDB()

	if input.Area == "admin" {
		config.AdminTheme = input.Theme
	} else {
		config.MonitorTheme = input.Theme
	}
	config.ThemeMode = input.Mode

	err := database.SaveThemeConfigDB(&config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "保存主题配置失败"})
		return
	}

	logMessage := fmt.Sprintf("切换%s主题: %s, 模式: %s", input.Area, input.Theme, input.Mode)
	global.AddSystemLog(models.LogTypeAudit, "info", logMessage)

	response := models.ThemeSwitchResponse{
		Success: true,
		Message: "主题切换成功",
		Theme:   input.Theme,
		Mode:    input.Mode,
	}

	c.JSON(http.StatusOK, response)
}

// ValidateTheme 验证主题文件
func ValidateTheme(c *gin.Context) {
	area := c.Query("area")
	theme := c.Query("theme")

	if area != "admin" && area != "monitor" {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "message": "无效的主题区域"})
		return
	}

	if theme == "" {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "message": "主题名称不能为空"})
		return
	}

	themes := scanThemeFiles(area)
	themeExists := false

	for _, t := range themes {
		if t.Name == theme {
			themeExists = true
			break
		}
	}

	if themeExists {
		c.JSON(http.StatusOK, models.ThemeValidateResponse{
			Valid:   true,
			Message: "主题文件存在",
			Theme:   theme,
		})
	} else {
		c.JSON(http.StatusOK, models.ThemeValidateResponse{
			Valid:   false,
			Message: "主题文件不存在",
			Theme:   theme,
		})
	}
}
