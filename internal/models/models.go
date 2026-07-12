package models

import "time"

type SystemLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Timestamp  time.Time `gorm:"index" json:"timestamp"`
	Type       string    `gorm:"column:type;index;type:varchar(20)" json:"type"`
	Level      string    `gorm:"column:level;type:varchar(20)" json:"level"`
	Message    string    `gorm:"column:message;type:text" json:"message"`
	BackupID   uint64    `gorm:"column:backup_id;index" json:"backup_id,omitempty"`
	Operation  string    `gorm:"column:operation;type:varchar(50)" json:"operation,omitempty"`
	ExportType string    `gorm:"column:export_type;type:varchar(20)" json:"export_type,omitempty"`
	FileSize   int64     `gorm:"column:file_size" json:"file_size,omitempty"`
}

type NodeGroup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Color     string    `gorm:"column:color;type:varchar(20)" json:"color"`        // 新增
	IsDefault bool      `gorm:"column:is_default;default:false" json:"is_default"` // 新增
	SortIndex int       `gorm:"column:sort_index;type:int;default:0" json:"sort_index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NodeConfig struct {
	ID             string    `gorm:"column:node_id;primaryKey;type:varchar(50)" json:"id"`
	GroupID        uint      `gorm:"column:group_id;index;default:1" json:"group_id"`
	CustomName     string    `gorm:"column:custom_name;type:varchar(100)" json:"name"`
	Location       string    `gorm:"column:location;type:varchar(100)" json:"location"`
	IP             string    `gorm:"column:ip;type:varchar(50)" json:"ip"`
	SortIndex      int       `gorm:"column:sort_index;type:int;default:0" json:"sort_index"`
	IsVisible      bool      `gorm:"column:is_visible;type:boolean;default:true" json:"is_visible"`
	MonthUp        uint64    `gorm:"column:month_up;default:0" json:"month_up"`
	MonthDown      uint64    `gorm:"column:month_down;default:0" json:"month_down"`
	UptimeRate     float64   `gorm:"column:uptime_rate;default:100.0" json:"uptime_rate"`
	OfflineTotal   string    `gorm:"column:offline_total;default:'0s'" json:"offline_total"`
	TotalStatsDays int       `gorm:"column:total_stats_days;default:0" json:"total_stats_days"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Node struct {
	ID             string        `json:"id"`
	GroupID        uint          `json:"group_id"`
	Name           string        `json:"name"`
	Location       string        `json:"location"`
	IP             string        `json:"ip"`
	OS             string        `json:"os"`
	CPU            float64       `json:"cpu"`
	Mem            float64       `json:"mem"`
	Disk           float64       `json:"disk"`
	Load           []float64     `json:"load"`
	Uptime         uint64        `json:"uptime"`
	ProcessCount   int           `json:"process_count"`
	Up             float64       `json:"up"`
	Down           float64       `json:"down"`
	NetIn          uint64        `json:"net_in"`
	NetOut         uint64        `json:"net_out"`
	MonthUp        uint64        `json:"month_up"`
	MonthDown      uint64        `json:"month_down"`
	UptimeRate     float64       `json:"uptime_rate"`
	OfflineTotal   string        `json:"offline_total"`
	Online         bool          `json:"online"`
	IsVisible      bool          `json:"is_visible"`
	SortIndex      int           `json:"sort_index"`
	LastSeen       time.Time     `json:"last_seen"`
	Docker         *DockerStatus `json:"docker"`
	Tasks          []DetectTask  `json:"tasks"`
	UptimeHeatMap  []HeatMapItem `json:"uptime_heat_map"`
	TotalStatsDays int           `json:"total_stats_days"`
}

type NodeReport struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Location     string        `json:"location"`
	IP           string        `json:"ip"`
	OS           string        `json:"os"`
	CPU          float64       `json:"cpu"`
	Mem          float64       `json:"mem"`
	Disk         float64       `json:"disk"`
	ProcessCount int           `json:"process_count"`
	Load         []float64     `json:"load"`
	Uptime       uint64        `json:"uptime"`
	Up           float64       `json:"up"`
	Down         float64       `json:"down"`
	NetIn        uint64        `json:"net_in"`
	NetOut       uint64        `json:"net_out"`
	Docker       *DockerStatus `json:"docker"`
}

type DockerStatus struct {
	RunningCount  int               `json:"running_count"`
	TotalCount    int               `json:"total_count"`
	HealthyCount  int               `json:"healthy_count"`
	ImageSize     string            `json:"image_size"`
	NetIn         uint64            `json:"net_in"`
	NetOut        uint64            `json:"net_out"`
	ContainerList []ContainerDetail `json:"container_list"`
}

type ContainerDetail struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	CPUUsage    string `json:"cpu_usage"`
	MemoryUsage string `json:"memory_usage"`
	DiskUsage   string `json:"disk_usage"`
	RootFsSize  string `json:"root_fs_size"`
	NetIn       uint64 `json:"net_in"`
	NetOut      uint64 `json:"net_out"`
}

type DetectTask struct {
	ID        uint64    `json:"id,string"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Target    string    `json:"target"`
	Port      int       `json:"port"`
	LastDelay float64   `json:"last_delay"`
	LossRate  float64   `json:"loss_rate"`
	SortIndex int       `json:"sort_index"`
	NodeIDs   []string  `json:"node_ids"`
	History   []float64 `json:"history"`
}

type TaskHistory struct {
	ID        uint      `gorm:"primaryKey"`
	NodeID    string    `gorm:"column:node_id;index;type:varchar(50)"`
	TaskID    uint64    `gorm:"column:task_id;index"`
	Delay     float64   `json:"delay"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

type UptimeDaily struct {
	ID              uint    `gorm:"primaryKey"`
	NodeID          string  `gorm:"column:node_id;index;type:varchar(50)"`
	Date            string  `gorm:"column:date;index;type:varchar(10)"`
	Rate            float64 `gorm:"column:rate"`
	Status          string  `gorm:"column:status"`
	OfflineDuration string  `gorm:"column:offline_duration"`
}

type HeatMapItem struct {
	Date            string  `json:"date"`
	Status          string  `json:"status"`
	UptimeRate      float64 `json:"uptime_rate"`
	OfflineDuration string  `json:"offline_duration"`
	DisplayInfo     string  `json:"displayInfo"`
}

type SiteConfig struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	SiteName   string `json:"site_name"`
	SiteSlogan string `json:"site_slogan"`
	SiteLogo   string `json:"site_logo"`
	SiteFooter string `json:"site_footer"`
	Version    string `json:"version"`
	Theme      string `json:"theme"`      // 新增
	CustomCSS  string `json:"custom_css"` // 新增
}

type GlobalSetting struct {
	ID                  uint   `gorm:"primaryKey" json:"id"`
	ShowDocker          bool   `gorm:"column:show_docker;default:true" json:"show_docker"`
	ShowNetDelay        bool   `gorm:"column:show_net_delay;default:true" json:"show_net_delay"`
	ShowHeatMap         bool   `gorm:"column:show_heatmap;default:true" json:"show_heatmap"`
	NodeRefreshInterval int    `gorm:"column:node_refresh_interval;default:5" json:"node_refresh_interval"`
	TaskShowCount       int    `gorm:"column:task_show_count;default:8" json:"task_show_count"`
	MaxLogCount         int    `gorm:"column:max_log_count;default:100" json:"max_log_count"`
	ViewAuthEnabled     bool   `gorm:"column:view_auth_enabled;default:false" json:"view_auth_enabled"`
	ViewAuthKey         string `gorm:"column:view_auth_key;type:varchar(100)" json:"view_auth_key"`
	AdminAuthKey        string `gorm:"column:admin_auth_key;type:varchar(100)" json:"admin_auth_key"`
	AdminTheme          string `gorm:"column:admin_theme;type:varchar(50);default:'default'" json:"admin_theme"`
	MonitorTheme        string `gorm:"column:monitor_theme;type:varchar(50);default:'default'" json:"monitor_theme"`
	ThemeMode           string `gorm:"column:theme_mode;type:varchar(20);default:'fixed'" json:"theme_mode"`
	NodeGroupEnabled    bool   `gorm:"column:node_group_enabled;default:true" json:"node_group_enabled"`                       // 新增
	DefaultNodeGroup    string `gorm:"column:default_node_group;type:varchar(50);default:'default'" json:"default_node_group"` // 新增
}

type Task struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Target    string    `json:"target"`
	Port      int       `json:"port"`
	SortIndex int       `gorm:"column:sort_index;type:int;default:0" json:"sort_index"`
	NodeIDs   []string  `gorm:"serializer:json" json:"node_ids"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type AlertConfig struct {
	ID              uint             `gorm:"primaryKey" json:"id"`
	Engine          AlertEngine      `gorm:"embedded;embeddedPrefix:engine_" json:"engine"`
	Metrics         AlertMetrics     `gorm:"embedded;embeddedPrefix:metrics_" json:"metrics"`
	Switches        AlertSwitches    `gorm:"embedded;embeddedPrefix:switches_" json:"switches"`
	Channels        AlertChannels    `gorm:"embedded;embeddedPrefix:channels_" json:"channels"`
	PortMonitors    []PortMonitor    `gorm:"serializer:json;column:port_monitors" json:"portMonitors"`
	ProcessMonitors []ProcessMonitor `gorm:"serializer:json;column:process_monitors" json:"processMonitors"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type AlertEngine struct {
	Enabled     bool     `gorm:"column:enabled" json:"enabled"`
	RateLimit   int      `gorm:"column:rate_limit" json:"rateLimit"`
	TargetNodes []string `gorm:"column:target_nodes;serializer:json" json:"targetNodes"`
	QuietStart  string   `gorm:"column:quiet_start" json:"quietStart"`
	QuietEnd    string   `gorm:"column:quiet_end" json:"quietEnd"`
}

type AlertMetrics struct {
	CPU           float64 `gorm:"column:cpu" json:"cpu"`
	Memory        float64 `gorm:"column:memory" json:"memory"`
	Disk          float64 `gorm:"column:disk" json:"disk"`
	Load          float64 `gorm:"column:load" json:"load"`
	Inode         float64 `gorm:"column:inode" json:"inode"`
	Swap          float64 `gorm:"column:swap" json:"swap"`          // 新增
	TimeDiff      int     `gorm:"column:time_diff" json:"timeDiff"` // 新增
	Connections   int     `gorm:"column:connections" json:"connections"`
	NetworkErrors int     `gorm:"column:network_errors" json:"networkErrors"` // 新增
	MonthUp       uint64  `gorm:"column:month_up" json:"monthUp"`
	MonthDown     uint64  `gorm:"column:month_down" json:"monthDown"`
	ProcessName   string  `gorm:"column:process_name" json:"processName"`
	PortService   string  `gorm:"column:port_service" json:"portService"`
}

type AlertSwitches struct {
	NodeOffline     bool `gorm:"column:node_offline" json:"nodeOffline"`
	DiskReadOnly    bool `gorm:"column:disk_readonly" json:"diskReadOnly"`       // 新增
	FileGrowth      bool `gorm:"column:file_growth" json:"fileGrowth"`           // 新增
	MemoryLeak      bool `gorm:"column:memory_leak" json:"memoryLeak"`           // 新增
	ProcessWatchdog bool `gorm:"column:process_watchdog" json:"processWatchdog"` // 新增
	PortService     bool `gorm:"column:port_service" json:"portService"`         // 新增
	DNSCheck        bool `gorm:"column:dns_check" json:"dnsCheck"`               // 新增
	ServiceRestart  bool `gorm:"column:service_restart" json:"serviceRestart"`
	Swap            bool `gorm:"column:swap" json:"swap"`
	TimeDiff        bool `gorm:"column:time_diff" json:"timeDiff"`
}

type AlertChannels struct {
	Telegram TGConfig `gorm:"embedded;embeddedPrefix:telegram_" json:"telegram"`
}

type TGConfig struct {
	Enabled  bool   `gorm:"column:enabled" json:"enabled"`
	BotToken string `gorm:"column:bot_token" json:"botToken"`
	ChatID   string `gorm:"column:chat_id" json:"chatId"`
}

type PortMonitor struct {
	Port int    `json:"port"`
	Name string `json:"name"`
	Host string `json:"host"`
}

type ProcessMonitor struct {
	Name    string `json:"name"`
	Cmdline string `json:"cmdline"`
}

type HistoryChartResponse struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
}

type BackupConfig struct {
	ID                        uint      `gorm:"primaryKey" json:"id"`
	AutoBackupEnabled         bool      `gorm:"column:auto_backup_enabled;default:false" json:"auto_backup_enabled"`
	BackupStrategy            string    `gorm:"column:backup_strategy;type:varchar(50);default:'daily_full'" json:"backup_strategy"`
	BackupRetentionDays       int       `gorm:"column:backup_retention_days;default:30" json:"backup_retention_days"`
	BackupMaxCount            int       `gorm:"column:backup_max_count;default:100" json:"backup_max_count"`
	BackupScheduleTime        string    `gorm:"column:backup_schedule_time;type:varchar(8);default:'02:00'" json:"backup_schedule_time"`
	BackupEncryptionEnabled   bool      `gorm:"column:backup_encryption_enabled;default:true" json:"backup_encryption_enabled"`
	BackupEncryptionAlgorithm string    `gorm:"column:backup_encryption_algorithm;type:varchar(20);default:'AES-256-GCM'" json:"backup_encryption_algorithm"`
	BackupVerifyIntegrity     bool      `gorm:"column:backup_verify_integrity;default:true" json:"backup_verify_integrity"`
	BackupCompressionEnabled  bool      `gorm:"column:backup_compression_enabled;default:true" json:"backup_compression_enabled"`
	EncryptionKeyHash         string    `gorm:"column:encryption_key_hash;type:varchar(128)" json:"encryption_key_hash"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type BackupFile struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement:true" json:"id,string"`
	Filename   string    `gorm:"column:filename;type:varchar(255);not null;index" json:"filename"`
	FilePath   string    `gorm:"column:file_path;type:varchar(500)" json:"file_path"`
	Type       string    `gorm:"column:type;type:varchar(20);index;default:'full'" json:"type"`
	Size       int64     `gorm:"column:size;default:0" json:"size"`
	Checksum   string    `gorm:"column:checksum;type:varchar(128);index" json:"checksum"`
	Verified   bool      `gorm:"column:verified;default:false" json:"verified"`
	Encrypted  bool      `gorm:"column:encrypted;default:true" json:"encrypted"`
	Compressed bool      `gorm:"column:compressed;default:true" json:"compressed"`
	Metadata   string    `gorm:"column:metadata;type:text" json:"metadata"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
	ExpiresAt  time.Time `gorm:"index" json:"expires_at"`
}

type DatabaseStatus struct {
	Online       bool   `json:"online"`
	Size         string `json:"size"`
	Rows         string `json:"rows"`
	TableCount   int    `json:"table_count"`
	LastBackupID uint64 `json:"last_backup_id"`
	LastBackupAt string `json:"last_backup_at"`
	BackupHealth int    `json:"backup_health"`
}

type BackupRequest struct {
	BackupID               uint64 `json:"backupId"`
	CreatePreRestoreBackup bool   `json:"createPreRestoreBackup"`
	VerifyBeforeRestore    bool   `json:"verifyBeforeRestore"`
}

type BackupResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type BackupStats struct {
	TotalCount       int    `json:"totalCount"`
	TotalSize        string `json:"totalSize"`
	LastBackupTime   string `json:"lastBackupTime"`
	Health           int    `json:"health"`
	FullCount        int    `json:"fullCount"`
	IncrementalCount int    `json:"incrementalCount"`
}

type BackupListResponse struct {
	Backups []BackupFile `json:"backups"`
	Stats   BackupStats  `json:"stats"`
}

type VerifyBackupRequest struct {
	BackupID uint64 `json:"backupId"`
	Checksum string `json:"checksum,omitempty"`
}

type VerifyBackupResponse struct {
	Verified   bool   `json:"verified"`
	Checksum   string `json:"checksum,omitempty"`
	Size       int64  `json:"size,omitempty"`
	Integrity  bool   `json:"integrity"`
	CanRestore bool   `json:"can_restore"`
	Message    string `json:"message,omitempty"`
}

type RestoreProgress struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	BackupID    uint64     `gorm:"index" json:"backup_id"`
	Progress    int        `gorm:"default:0" json:"progress"`
	Status      string     `gorm:"type:varchar(20)" json:"status"`
	Message     string     `gorm:"type:text" json:"message"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type BackupSchedule struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NextRun   time.Time `gorm:"index" json:"next_run"`
	LastRun   time.Time `json:"last_run"`
	Status    string    `gorm:"type:varchar(20)" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// 原 ThemeConfig 重命名，用于存储用户选择的主题设置
type ThemeSetting struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AdminTheme   string    `gorm:"column:admin_theme;type:varchar(50);default:'default'" json:"admin_theme"`
	MonitorTheme string    `gorm:"column:monitor_theme;type:varchar(50);default:'default'" json:"monitor_theme"`
	ThemeMode    string    `gorm:"column:theme_mode;type:varchar(20);default:'fixed'" json:"theme_mode"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// 新增主题样式定义，用于配置主题颜色等
type ThemeConfig struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Primary     string    `json:"primary"`
	Secondary   string    `json:"secondary"`
	Accent      string    `json:"accent"`
	Background  string    `json:"background"`
	Text        string    `json:"text"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// 新增文件类型定义（用于扫描主题文件）
type ThemeFile struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Area     string `json:"area"`
	Type     string `json:"type"`
	FileSize int64  `json:"file_size"`
	Modified string `json:"modified"`
}

type ThemeListResponse struct {
	AdminThemes    []ThemeFile `json:"admin_themes"`
	MonitorThemes  []ThemeFile `json:"monitor_themes"`
	CurrentAdmin   string      `json:"current_admin"`
	CurrentMonitor string      `json:"current_monitor"`
	ThemeMode      string      `json:"theme_mode"`
}

type ThemeSwitchRequest struct {
	Area  string `json:"area"`
	Theme string `json:"theme"`
	Mode  string `json:"mode"`
}

type ThemeSwitchResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Theme   string `json:"theme,omitempty"`
	Mode    string `json:"mode,omitempty"`
}

type ThemeValidateResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	Theme   string `json:"theme,omitempty"`
}

// 常量定义
const (
	LogTypeSystem   = "system"
	LogTypeHost     = "host"
	LogTypeNetwork  = "network"
	LogTypeAlert    = "alert"
	LogTypeSecurity = "security"
	LogTypeAudit    = "audit"
	LogTypeBackup   = "backup"
	LogTypeRestore  = "restore"
	LogTypeExport   = "export"

	TaskTypeICMP = "icmp"
	TaskTypeTCP  = "tcp"
	TaskTypeHTTP = "http"

	ExportFormatSQL  = "sql"
	ExportFormatCSV  = "csv"
	ExportFormatJSON = "json"

	BackupStrategyDailyFull      = "daily_full"
	BackupStrategyWeeklyFullInc  = "weekly_full_inc"
	BackupStrategyMonthlyArchive = "monthly_archive"

	EncryptionAES256GCM = "AES-256-GCM"
	EncryptionAES256CBC = "AES-256-CBC"
	EncryptionChaCha20  = "ChaCha20"

	BackupTypeFull        = "full"
	BackupTypeIncremental = "incremental"

	ThemeModeFixed   = "fixed"
	ThemeModeSystem  = "system"
	ThemeAreaAdmin   = "admin"
	ThemeAreaMonitor = "monitor"
	ThemeDefault     = "default"
	ThemeDark        = "dark"
	ThemeLight       = "light"  // 新增
	ThemeBlue        = "blue"   // 新增
	ThemeGreen       = "green"  // 新增
	ThemePurple      = "purple" // 新增

	RestoreTypeFull    = "full"    // 新增
	RestoreTypePartial = "partial" // 新增
)
