package global

import (
	"net/http"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"slite/internal/models"
)

// 全局变量定义
var (
	// 节点管理
	Nodes   = make(map[string]*models.Node)
	NodesMu sync.RWMutex

	// 任务历史
	TaskHistoryMap = make(map[string][]float64)
	HistoryMu      sync.Mutex

	// WebSocket
	Upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	Clients   = make(map[*websocket.Conn]bool)
	ClientsMu sync.Mutex

	// 雪花ID生成器
	GlobalSnowflake *snowflake.Node

	// 告警频率控制
	LastAlertMap = make(map[string]time.Time)
	AlertMu      sync.Mutex

	// 监控服务器自身信息
	SelfNodeID       string
	SelfLocation     string
	PrimaryInterface string
	LastTotalSent    uint64
	LastTotalRecv    uint64
	CurrentPublicIP  string
	IPUpdateTicker   *time.Ticker

	// 任务历史
	TaskHistory = make(map[uint64][]float64)

	// 数据库连接
	DB *gorm.DB
)

// AddSystemLog 添加系统日志（全局函数，避免循环导入）
func AddSystemLog(logType string, level string, message string) {
	newLog := models.SystemLog{
		Timestamp: time.Now(),
		Type:      logType,
		Level:     level,
		Message:   message,
	}
	DB.Create(&newLog)

	var logCount int64
	DB.Model(&models.SystemLog{}).Count(&logCount)
	var settings models.GlobalSetting
	DB.First(&settings, 1)
	if logCount > int64(settings.MaxLogCount) {
		var oldestLog models.SystemLog
		DB.Order("timestamp asc").First(&oldestLog)
		DB.Delete(&oldestLog)
	}
}
