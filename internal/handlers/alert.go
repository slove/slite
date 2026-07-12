package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"slite/internal/alert"
	"slite/internal/controllers"
	"slite/internal/global"
	"slite/internal/models"
)

// GetAlertConfig 获取告警配置
func GetAlertConfig(c *gin.Context) {
	var config models.AlertConfig
	if err := global.DB.First(&config, 1).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "CONFIG NOT FOUND"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// SaveAlertConfig 保存告警配置
func SaveAlertConfig(c *gin.Context) {
	var input models.AlertConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "INVALID FORMAT"})
		return
	}

	var existing models.AlertConfig
	if err := global.DB.First(&existing, 1).Error; err != nil {
		input.ID = 1
		global.DB.Create(&input)
	} else {
		input.ID = 1
		global.DB.Save(&input)
	}

	global.AddSystemLog(models.LogTypeAudit, "info", "告警配置已更新")
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// TestAlertPush 测试告警推送
func TestAlertPush(c *gin.Context) {
	var input models.AlertConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "INVALID FORMAT"})
		return
	}

	if !input.Channels.Telegram.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"message": "TELEGRAM CHANNEL DISABLED"})
		return
	}

	if input.Channels.Telegram.BotToken == "" || input.Channels.Telegram.ChatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "TELEGRAM CONFIG INCOMPLETE"})
		return
	}

	var siteConfig models.SiteConfig
	global.DB.First(&siteConfig, 1)

	testMessage := fmt.Sprintf(`[监控告警] %s
---------------------------
节点状态: 监控中
生效节点: %d 台主机
告警间隔: %d 分钟
静默时段: %s - %s

关键指标:
- CPU使用率: %.0f%%
- 内存使用率: %.0f%%
- 磁盘使用率: %.0f%%
- 系统负载: %.1f
- Inode使用率: %.0f%%
- 连接数阈值: %d
- 月度上传配额: %dGB
- 月度下载配额: %dGB
- 监控进程: %s
- 监控端口: %s

监控状态:
- 节点离线检测: %s
- 服务重启检测: %s
- SWAP使用率监控: %s
- 时间偏移检测: %s

触发时间: %s
建议操作: 请登录服务器查看详细状态
---------------------------`,
		siteConfig.SiteName,
		len(input.Engine.TargetNodes),
		input.Engine.RateLimit,
		input.Engine.QuietStart,
		input.Engine.QuietEnd,
		input.Metrics.CPU,
		input.Metrics.Memory,
		input.Metrics.Disk,
		input.Metrics.Load,
		input.Metrics.Inode,
		input.Metrics.Connections,
		input.Metrics.MonthUp,
		input.Metrics.MonthDown,
		input.Metrics.ProcessName,
		input.Metrics.PortService,
		controllers.FormatBool(input.Switches.NodeOffline),
		controllers.FormatBool(input.Switches.ServiceRestart),
		controllers.FormatBool(input.Switches.Swap),
		controllers.FormatBool(input.Switches.TimeDiff),
		time.Now().Format("2006-01-02 15:04:05"))

	err := alert.SendTelegramAlert(input.Channels.Telegram.BotToken, input.Channels.Telegram.ChatID, testMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "SEND FAILED", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
