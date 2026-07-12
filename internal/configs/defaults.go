package configs

import (
	"time"

	"slite/internal/models"
)

/*
系统默认值配置
集中管理所有默认配置，方便部署时修改
*/

/*
默认站点配置
*/
var DefaultSiteConfig = models.SiteConfig{
	ID:         1,
	SiteName:   "SLITE MONITORING",
	SiteSlogan: "你的每一次心跳，皆有回响。",
	SiteLogo:   "",
	SiteFooter: "© 2026 SLITE MONITORING",
	Version:    "0.1.0",
	Theme:      models.ThemeDefault,
	CustomCSS:  "",
}

/*
默认全局设置
*/
var DefaultGlobalSettings = models.GlobalSetting{
	ID:                  1,
	NodeRefreshInterval: 5,
	ShowNetDelay:        true,
	ShowDocker:          true,
	ShowHeatMap:         true,
	TaskShowCount:       8,
	MaxLogCount:         100,
	ViewAuthEnabled:     false,
	ViewAuthKey:         "123456",
	AdminAuthKey:        "admin888",
	NodeGroupEnabled:    true,
	DefaultNodeGroup:    "default",
}

/*
默认告警配置
*/
var DefaultAlertConfig = models.AlertConfig{
	ID: 1,
	Engine: models.AlertEngine{
		Enabled:     true,
		RateLimit:   10,
		QuietStart:  "01:00",
		QuietEnd:    "06:00",
		TargetNodes: []string{},
	},
	Metrics: models.AlertMetrics{
		CPU:           90.0,
		Memory:        85.0,
		Disk:          90.0,
		Load:          4.0,
		Inode:         85.0,
		Swap:          30.0,
		TimeDiff:      30,
		Connections:   5000,
		NetworkErrors: 100,
		MonthUp:       1000,
		MonthDown:     2000,
	},
	Switches: models.AlertSwitches{
		NodeOffline:     true,
		DiskReadOnly:    true,
		FileGrowth:      true,
		MemoryLeak:      true,
		ProcessWatchdog: true,
		PortService:     true,
		DNSCheck:        true,
		ServiceRestart:  true,
	},
	Channels: models.AlertChannels{
		Telegram: models.TGConfig{
			Enabled:  false,
			BotToken: "",
			ChatID:   "",
		},
	},
}

/*
默认主题配置
*/
var DefaultThemes = []models.ThemeConfig{
	{
		Name:        models.ThemeDefault,
		DisplayName: "默认主题",
		Primary:     "#3b82f6",
		Secondary:   "#10b981",
		Accent:      "#f59e0b",
		Background:  "#ffffff",
		Text:        "#111827",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	},
	{
		Name:        models.ThemeDark,
		DisplayName: "深色主题",
		Primary:     "#60a5fa",
		Secondary:   "#34d399",
		Accent:      "#fbbf24",
		Background:  "#111827",
		Text:        "#f3f4f6",
		IsActive:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	},
	{
		Name:        models.ThemeLight,
		DisplayName: "浅色主题",
		Primary:     "#2563eb",
		Secondary:   "#059669",
		Accent:      "#d97706",
		Background:  "#f9fafb",
		Text:        "#1f2937",
		IsActive:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	},
	{
		Name:        models.ThemeBlue,
		DisplayName: "蓝色主题",
		Primary:     "#3b82f6",
		Secondary:   "#0ea5e9",
		Accent:      "#6366f1",
		Background:  "#eff6ff",
		Text:        "#1e3a8a",
		IsActive:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	},
	{
		Name:        models.ThemeGreen,
		DisplayName: "绿色主题",
		Primary:     "#10b981",
		Secondary:   "#34d399",
		Accent:      "#059669",
		Background:  "#f0fdf4",
		Text:        "#064e3b",
		IsActive:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	},
	{
		Name:        models.ThemePurple,
		DisplayName: "紫色主题",
		Primary:     "#8b5cf6",
		Secondary:   "#a78bfa",
		Accent:      "#7c3aed",
		Background:  "#faf5ff",
		Text:        "#4c1d95",
		IsActive:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	},
}

/*
默认节点分组
*/
var DefaultNodeGroups = []models.NodeGroup{
	{
		Name:      "default",
		Color:     "#3b82f6",
		SortIndex: 0,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	},
	{
		Name:      "production",
		Color:     "#10b981",
		SortIndex: 1,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	},
	{
		Name:      "testing",
		Color:     "#f59e0b",
		SortIndex: 2,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	},
	{
		Name:      "development",
		Color:     "#8b5cf6",
		SortIndex: 3,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	},
	{
		Name:      "backup",
		Color:     "#64748b",
		SortIndex: 4,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	},
}

/*
默认系统常量
*/
var DefaultSystemConstants = struct {
	ServerPort           string
	AgentReportInterval  int
	TaskCheckInterval    int
	AlertCheckInterval   int
	MaxHistoryHours      int
	CleanupIntervalHours int
	BackupRetentionDays  int
	BackupSchedule       string
}{
	ServerPort:           "8080",
	AgentReportInterval:  30,
	TaskCheckInterval:    60,
	AlertCheckInterval:   60,
	MaxHistoryHours:      168,
	CleanupIntervalHours: 24,
	BackupRetentionDays:  30,
	BackupSchedule:       "0 2 * * *",
}

/*
默认数据库备份配置
*/
var DefaultBackupConfig = struct {
	BackupDirectory      string
	MaxBackupFiles       int
	AutoBackupEnabled    bool
	AutoBackupSchedule   string
	BackupCompression    bool
	VerifyAfterBackup    bool
	NotificationOnBackup bool
}{
	BackupDirectory:      "database_backups",
	MaxBackupFiles:       30,
	AutoBackupEnabled:    true,
	AutoBackupSchedule:   "0 2 * * *",
	BackupCompression:    false,
	VerifyAfterBackup:    true,
	NotificationOnBackup: true,
}

/*
部署建议
*/
var DeploymentRecommendations = struct {
	DockerComposeVersion    string
	ReverseProxyConfig      string
	FirewallRules           []string
	MonitoringTips          []string
	BackupStrategy          []string
	SecurityRecommendations []string
}{
	DockerComposeVersion: "3.8",
	ReverseProxyConfig: `# Nginx反向代理配置示例
location / {
    proxy_pass http://localhost:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}`,
	FirewallRules: []string{
		"开放8080端口供Web访问",
		"限制Agent上报端口访问",
		"配置SSH访问限制",
	},
	MonitoringTips: []string{
		"建议为不同环境创建不同的节点分组",
		"生产环境建议开启所有告警开关",
		"根据服务器配置调整资源阈值",
		"定期检查告警历史记录",
		"配置Telegram推送接收紧急告警",
	},
	BackupStrategy: []string{
		"启用自动数据库备份功能",
		"定期测试备份文件恢复",
		"将备份文件存储在异地",
		"保留至少30天的备份历史",
		"定期验证备份文件完整性",
	},
	SecurityRecommendations: []string{
		"修改默认的管理员密码",
		"启用访问认证功能",
		"定期更新系统组件",
		"配置防火墙规则",
		"监控异常登录行为",
	},
}

/*
默认数据库备份策略
*/
var DefaultBackupStrategies = []struct {
	Name        string
	Description string
	Schedule    string
	Retention   int
	Enabled     bool
}{
	{
		Name:        "每日备份",
		Description: "每天凌晨2点执行完整备份",
		Schedule:    "0 2 * * *",
		Retention:   7,
		Enabled:     true,
	},
	{
		Name:        "每周备份",
		Description: "每周日凌晨3点执行完整备份",
		Schedule:    "0 3 * * 0",
		Retention:   30,
		Enabled:     true,
	},
	{
		Name:        "月度备份",
		Description: "每月1号凌晨4点执行完整备份",
		Schedule:    "0 4 1 * *",
		Retention:   365,
		Enabled:     true,
	},
}

/*
默认数据库恢复策略
*/
var DefaultRestoreStrategies = []struct {
	Name        string
	Description string
	RestoreType string
	VerifyAfter bool
}{
	{
		Name:        "完整恢复",
		Description: "完整恢复数据库到指定备份点",
		RestoreType: models.RestoreTypeFull,
		VerifyAfter: true,
	},
	{
		Name:        "测试恢复",
		Description: "恢复数据库到测试环境",
		RestoreType: models.RestoreTypePartial,
		VerifyAfter: true,
	},
}

/*
默认备份目录结构
*/
var DefaultBackupDirectoryStructure = struct {
	Root          string
	Daily         string
	Weekly        string
	Monthly       string
	Temp          string
	Logs          string
	ConfigBackups string
}{
	Root:          "database_backups",
	Daily:         "daily",
	Weekly:        "weekly",
	Monthly:       "monthly",
	Temp:          "temp",
	Logs:          "logs",
	ConfigBackups: "configs",
}

/*
默认备份文件名模板
*/
var DefaultBackupFilenameTemplates = struct {
	DailyTemplate   string
	WeeklyTemplate  string
	MonthlyTemplate string
	ManualTemplate  string
}{
	DailyTemplate:   "slite_backup_daily_%s.db",
	WeeklyTemplate:  "slite_backup_weekly_%s.db",
	MonthlyTemplate: "slite_backup_monthly_%s.db",
	ManualTemplate:  "slite_backup_manual_%s_%s.db",
}
