package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"slite/internal/global"
	"slite/internal/models"
)

/*
初始化数据库连接和表结构
1. 建立与 SQLite 的连接
2. 处理告警配置结构的兼容性迁移
3. 自动迁移所有定义的模型结构
4. 创建物理索引以优化查询
5. 初始化本地备份目录
*/
func InitDB() {
	var err error
	global.DB, err = gorm.Open(sqlite.Open("slite.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}

	if global.DB.Migrator().HasColumn(&models.AlertConfig{}, "channels") {
		log.Println("检测到旧版告警配置结构，正在执行结构重写...")
		global.DB.Migrator().DropTable(&models.AlertConfig{})
	}

	err = global.DB.AutoMigrate(
		&models.Task{},
		&models.TaskHistory{},
		&models.UptimeDaily{},
		&models.SiteConfig{},
		&models.GlobalSetting{},
		&models.NodeConfig{},
		&models.AlertConfig{},
		&models.SystemLog{},
		&models.NodeGroup{},
		&models.BackupConfig{},
		&models.BackupFile{},
		&models.RestoreProgress{},
		&models.BackupSchedule{},
		&models.ThemeSetting{},
	)

	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_task_history_lookup ON task_histories (node_id, task_id, created_at)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_backup_files_created ON backup_files (created_at, type)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_backup_files_expires ON backup_files (expires_at)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_restore_progress_status ON restore_progresses (status, updated_at)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_task_history_created_at ON task_histories (created_at DESC)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_backup_files_id ON backup_files (id)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_backup_files_verified ON backup_files (verified)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_backup_files_created_at_id ON backup_files (created_at DESC, id DESC)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_system_logs_timestamp ON system_logs (timestamp DESC)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_node_config_node_id ON node_configs (node_id)")
	global.DB.Exec("CREATE INDEX IF NOT EXISTS idx_uptime_dailies_node_date ON uptime_dailies (node_id, date DESC)")

	if err := os.MkdirAll("backups", 0755); err != nil {
		log.Printf("[Backup] 创建备份目录失败: %v", err)
	}

	initDefaultConfig()
}

/*
初始化系统默认配置
包含节点分组、站点信息、全局设置、告警引擎、备份配置及主题风格
仅在各表无数据时执行初始化插入
*/
func initDefaultConfig() {
	var groupCount int64
	global.DB.Model(&models.NodeGroup{}).Count(&groupCount)
	if groupCount == 0 {
		defaultGroup := models.NodeGroup{
			ID:        1,
			Name:      "默认分组",
			SortIndex: 0,
		}
		global.DB.Create(&defaultGroup)
		log.Println("已初始化唯一默认节点分组")
	}

	var siteCount int64
	global.DB.Model(&models.SiteConfig{}).Count(&siteCount)
	if siteCount == 0 {
		defaultConfig := models.SiteConfig{
			ID:         1,
			SiteName:   "SLITE MONITORING",
			SiteSlogan: "你的每一次心跳，皆有回响。",
			SiteLogo:   "",
			SiteFooter: "© 2026 SLITE MONITORING",
			Version:    "0.1.0",
		}
		global.DB.Create(&defaultConfig)
		log.Println("已初始化默认站点外观配置")
	}

	var globalCount int64
	global.DB.Model(&models.GlobalSetting{}).Count(&globalCount)
	if globalCount == 0 {
		defaultSettings := models.GlobalSetting{
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
			AdminTheme:          models.ThemeDefault,
			MonitorTheme:        models.ThemeDefault,
			ThemeMode:           models.ThemeModeFixed,
		}
		global.DB.Create(&defaultSettings)
		log.Println("已初始化默认全局显示及安全设置")
	}

	var alertCount int64
	global.DB.Model(&models.AlertConfig{}).Count(&alertCount)
	if alertCount == 0 {
		defaultAlert := models.AlertConfig{
			ID: 1,
			Engine: models.AlertEngine{
				Enabled:     true,
				RateLimit:   10,
				TargetNodes: []string{},
				QuietStart:  "01:00",
				QuietEnd:    "06:00",
			},
			Metrics: models.AlertMetrics{
				CPU:         90.0,
				Memory:      85.0,
				Disk:        90.0,
				Load:        4.0,
				Inode:       85.0,
				Connections: 5000,
				MonthUp:     1000,
				MonthDown:   2000,
				ProcessName: "",
				PortService: "",
			},
			Switches: models.AlertSwitches{
				NodeOffline:    true,
				ServiceRestart: true,
				Swap:           true,
				TimeDiff:       true,
			},
			Channels: models.AlertChannels{
				Telegram: models.TGConfig{
					Enabled:  false,
					BotToken: "",
					ChatID:   "",
				},
			},
		}
		global.DB.Create(&defaultAlert)
		log.Println("已初始化默认模块化告警策略配置")
	}

	var backupConfigCount int64
	global.DB.Model(&models.BackupConfig{}).Count(&backupConfigCount)
	if backupConfigCount == 0 {
		defaultBackupConfig := models.BackupConfig{
			ID:                        1,
			AutoBackupEnabled:         false,
			BackupStrategy:            models.BackupStrategyDailyFull,
			BackupRetentionDays:       30,
			BackupMaxCount:            100,
			BackupScheduleTime:        "02:00",
			BackupEncryptionEnabled:   true,
			BackupEncryptionAlgorithm: models.EncryptionAES256GCM,
			BackupVerifyIntegrity:     true,
			BackupCompressionEnabled:  true,
			EncryptionKeyHash:         "",
			CreatedAt:                 time.Now(),
			UpdatedAt:                 time.Now(),
		}
		global.DB.Create(&defaultBackupConfig)
		log.Println("已初始化默认备份管理系统配置")
	}

	var themeCount int64
	global.DB.Model(&models.ThemeSetting{}).Count(&themeCount)
	if themeCount == 0 {
		defaultTheme := models.ThemeSetting{
			ID:           1,
			AdminTheme:   models.ThemeDefault,
			MonitorTheme: models.ThemeDefault,
			ThemeMode:    models.ThemeModeFixed,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		global.DB.Create(&defaultTheme)
		log.Println("已初始化默认主题配置")
	}

	go CleanupExpiredBackups() // 修正：调用导出的函数
}

/*
获取站点前端显示的相关外观配置
默认获取 ID 为 1 的主配置
*/
func GetSiteConfig() models.SiteConfig {
	var config models.SiteConfig
	global.DB.First(&config, 1)
	return config
}

/*
获取系统全局设置
包括监控频率、模块开关及认证信息
*/
func GetGlobalSettings() models.GlobalSetting {
	var settings models.GlobalSetting
	global.DB.First(&settings, 1)
	return settings
}

/*
从数据库获取备份系统的核心配置
*/
func GetBackupConfigDB() (models.BackupConfig, error) {
	var config models.BackupConfig
	err := global.DB.First(&config, 1).Error
	return config, err
}

/*
更新备份系统配置记录
自动同步更新时间戳
*/
func SaveBackupConfigDB(config *models.BackupConfig) error {
	config.UpdatedAt = time.Now()
	return global.DB.Save(config).Error
}

/*
将备份配置重置为出厂默认值
*/
func ResetBackupConfigDB() (models.BackupConfig, error) {
	defaultConfig := models.BackupConfig{
		ID:                        1,
		AutoBackupEnabled:         false,
		BackupStrategy:            models.BackupStrategyDailyFull,
		BackupRetentionDays:       30,
		BackupMaxCount:            100,
		BackupScheduleTime:        "02:00",
		BackupEncryptionEnabled:   true,
		BackupEncryptionAlgorithm: models.EncryptionAES256GCM,
		BackupVerifyIntegrity:     true,
		BackupCompressionEnabled:  true,
		EncryptionKeyHash:         "",
		UpdatedAt:                 time.Now(),
	}
	err := global.DB.Save(&defaultConfig).Error
	return defaultConfig, err
}

/*
查询所有处于激活状态的监控任务
并按指定的排序列和 ID 顺序返回
*/
func GetActiveTasks() []models.Task {
	var tasks []models.Task
	global.DB.Where("is_active = ?", true).Order("sort_index ASC, id ASC").Find(&tasks)
	return tasks
}

/*
保存并维护节点每日在线率数据
自动计算并同步更新节点总配置表中的统计天数
*/
func SaveNodeUptimeDaily(logData *models.UptimeDaily) error {
	nodeIDStr := fmt.Sprintf("%v", logData.NodeID)
	var existing models.UptimeDaily
	result := global.DB.Where("node_id = ? AND date = ?", nodeIDStr, logData.Date).First(&existing)
	var err error
	if result.Error != nil {
		err = global.DB.Create(logData).Error
	} else {
		err = global.DB.Model(&existing).Updates(map[string]interface{}{
			"rate":             logData.Rate,
			"status":           logData.Status,
			"offline_duration": logData.OfflineDuration,
		}).Error
	}

	if err == nil {
		updateSQL := `
			UPDATE node_configs 
			SET total_stats_days = (
				SELECT COUNT(*) FROM uptime_dailies WHERE node_id = ?
			) 
			WHERE node_id = ?`
		res := global.DB.Exec(updateSQL, nodeIDStr, nodeIDStr)
		if res.Error != nil {
			log.Printf("[Database] 更新节点 %s 统计天数失败: %v", nodeIDStr, res.Error)
		}
	}
	return err
}

/*
计算特定任务在数据库中的历史数据覆盖范围
返回小时数，用于前端历史图表的时间轴缩放控制
*/
func GetTaskHistoryTimeRange(nodeID string, taskID uint64) int {
	var oldestHistory models.TaskHistory
	result := global.DB.Where("node_id = ? AND task_id = ?", nodeID, taskID).
		Order("created_at ASC").
		First(&oldestHistory)
	if result.Error != nil {
		return 1
	}
	hoursSpan := int(time.Since(oldestHistory.CreatedAt).Hours())
	if hoursSpan < 1 {
		return 1
	}
	return hoursSpan
}

/*
获取指定任务的历史延迟数据序列
自动根据查询的时长范围调整时间标签的显示颗粒度
*/
func GetTaskHistoryData(nodeID string, taskID uint64, hours int) models.HistoryChartResponse {
	var histories []models.TaskHistory
	startTime := time.Now().Add(time.Duration(-hours) * time.Hour)
	global.DB.Where("node_id = ? AND task_id = ? AND created_at >= ?", nodeID, taskID, startTime).
		Order("created_at ASC").
		Find(&histories)

	var labels []string
	var delays []float64
	for _, h := range histories {
		timeStr := ""
		if hours <= 1 {
			timeStr = h.CreatedAt.Format("15:04:05")
		} else if hours <= 24 {
			timeStr = h.CreatedAt.Format("15:04")
		} else if hours <= 72 {
			timeStr = h.CreatedAt.Format("01-02 15:04")
		} else {
			timeStr = h.CreatedAt.Format("01-02")
		}
		labels = append(labels, timeStr)
		delays = append(delays, h.Delay)
	}
	return models.HistoryChartResponse{
		Labels: labels,
		Values: delays,
	}
}

/*
查询系统操作或审计日志
支持按类型过滤及关键词搜索，限制返回最近的 200 条记录
*/
func GetSystemLogs(logType string, query string) []models.SystemLog {
	var logs []models.SystemLog
	db := global.DB.Order("timestamp desc")
	if logType != "" && logType != "all" {
		db = db.Where("type = ?", logType)
	}
	if query != "" {
		db = db.Where("message LIKE ?", "%"+query+"%")
	}
	db.Limit(200).Find(&logs)
	return logs
}

/*
在数据库中创建备份文件记录并执行物理文件备份
包含目录检查、物理复制、哈希计算及大小统计
*/
func CreateBackupFileDB(backupFile *models.BackupFile) error {
	err := global.DB.Create(backupFile).Error
	if err != nil {
		return err
	}
	if err := os.MkdirAll("backups", 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %v", err)
	}
	filePath := filepath.Join("backups", backupFile.Filename)
	backupFile.FilePath = filePath
	size, checksum, err := performDatabaseBackup(filePath)
	if err != nil {
		global.DB.Delete(backupFile)
		return fmt.Errorf("执行数据库备份失败: %v", err)
	}
	backupFile.Size = size
	backupFile.Checksum = checksum
	return global.DB.Save(backupFile).Error
}

/*
执行物理数据库文件复制并计算校验值
使用 io.Copy 确保流式处理大文件，避免内存溢出
*/
func performDatabaseBackup(destPath string) (int64, string, error) {
	sourcePath := "slite.db"
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return 0, "", fmt.Errorf("打开源数据库文件失败: %v", err)
	}
	defer srcFile.Close()
	destFile, err := os.Create(destPath)
	if err != nil {
		return 0, "", fmt.Errorf("创建备份文件失败: %v", err)
	}
	defer destFile.Close()
	bytesCopied, err := io.Copy(destFile, srcFile)
	if err != nil {
		os.Remove(destPath)
		return 0, "", fmt.Errorf("复制数据库文件失败: %v", err)
	}
	hash := sha256.New()
	srcFile.Seek(0, 0)
	if _, err := io.Copy(hash, srcFile); err != nil {
		return bytesCopied, "", fmt.Errorf("计算文件哈希失败: %v", err)
	}
	checksum := hex.EncodeToString(hash.Sum(nil))
	return bytesCopied, checksum, nil
}

/*
获取所有备份文件的列表
自动检查物理文件是否存在，并同步记录文件的实际大小
*/
func GetBackupFilesDB() ([]models.BackupFile, error) {
	var backupFiles []models.BackupFile
	err := global.DB.Order("created_at DESC").Find(&backupFiles).Error
	for i := range backupFiles {
		if backupFiles[i].FilePath != "" {
			if info, err := os.Stat(backupFiles[i].FilePath); err == nil {
				if backupFiles[i].Size != info.Size() {
					backupFiles[i].Size = info.Size()
					global.DB.Save(&backupFiles[i])
				}
			} else {
				backupFiles[i].Verified = false
				global.DB.Save(&backupFiles[i])
			}
		}
	}
	return backupFiles, err
}

/*
获取备份子系统的综合统计数据
包含文件总数、存储占用、最后备份时间及健康度评估
*/
func GetBackupStatsDB() (models.BackupStats, error) {
	var stats models.BackupStats
	var totalCount int64
	global.DB.Model(&models.BackupFile{}).Count(&totalCount)
	stats.TotalCount = int(totalCount)
	var totalSize int64
	global.DB.Model(&models.BackupFile{}).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)
	if totalSize < 1024 {
		stats.TotalSize = fmt.Sprintf("%d Bytes", totalSize)
	} else if totalSize < 1024*1024 {
		stats.TotalSize = fmt.Sprintf("%.2f KB", float64(totalSize)/1024)
	} else if totalSize < 1024*1024*1024 {
		stats.TotalSize = fmt.Sprintf("%.2f MB", float64(totalSize)/(1024*1024))
	} else {
		stats.TotalSize = fmt.Sprintf("%.2f GB", float64(totalSize)/(1024*1024*1024))
	}
	var lastBackup models.BackupFile
	global.DB.Order("created_at DESC").Limit(1).Find(&lastBackup)
	if lastBackup.ID != 0 {
		stats.LastBackupTime = lastBackup.CreatedAt.Format("2006-01-02 15:04:05")
	} else {
		stats.LastBackupTime = "无记录"
	}
	var verifiedCount int64
	global.DB.Model(&models.BackupFile{}).Where("verified = ?", true).Count(&verifiedCount)
	if totalCount > 0 {
		stats.Health = int(float64(verifiedCount) / float64(totalCount) * 100)
	} else {
		stats.Health = 0
	}
	var fullCount, incrementalCount int64
	global.DB.Model(&models.BackupFile{}).Where("type = ?", models.BackupTypeFull).Count(&fullCount)
	global.DB.Model(&models.BackupFile{}).Where("type = ?", models.BackupTypeIncremental).Count(&incrementalCount)
	stats.FullCount = int(fullCount)
	stats.IncrementalCount = int(incrementalCount)
	return stats, nil
}

/*
根据 ID 获取单个备份文件的详情
自动校验文件在物理存储中的最新状态
*/
func GetBackupFileByIDDB(id uint64) (models.BackupFile, error) {
	var backupFile models.BackupFile
	err := global.DB.First(&backupFile, id).Error
	if err == nil && backupFile.FilePath != "" {
		if info, err := os.Stat(backupFile.FilePath); err == nil {
			if backupFile.Size != info.Size() {
				backupFile.Size = info.Size()
				global.DB.Save(&backupFile)
			}
		}
	}
	return backupFile, err
}

/*
验证备份文件的物理完整性
重新计算文件校验和并与记录进行比对
*/
func VerifyBackupFileDB(id uint64) (bool, string, error) {
	var backupFile models.BackupFile
	err := global.DB.First(&backupFile, id).Error
	if err != nil {
		return false, "", err
	}
	if backupFile.FilePath == "" {
		return false, "备份文件路径不存在", nil
	}
	fileInfo, err := os.Stat(backupFile.FilePath)
	if err != nil {
		return false, "备份文件不存在于文件系统中", nil
	}
	if backupFile.Size != fileInfo.Size() {
		return false, "备份文件大小不匹配", nil
	}
	file, err := os.Open(backupFile.FilePath)
	if err != nil {
		return false, "无法打开备份文件进行验证", nil
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, "计算文件哈希失败", nil
	}
	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if backupFile.Checksum == "" {
		backupFile.Checksum = actualChecksum
		backupFile.Verified = true
		global.DB.Save(&backupFile)
		return true, "备份文件完整性验证通过", nil
	}
	if actualChecksum != backupFile.Checksum {
		return false, "备份文件校验和不匹配", nil
	}
	backupFile.Verified = true
	global.DB.Save(&backupFile)
	return true, "备份文件完整性验证通过", nil
}

/*
从系统中永久删除备份记录
同时会从磁盘中移除对应的物理文件
*/
func DeleteBackupFileDB(id uint64) error {
	var backupFile models.BackupFile
	err := global.DB.First(&backupFile, id).Error
	if err != nil {
		return err
	}
	if backupFile.FilePath != "" {
		os.Remove(backupFile.FilePath)
	}
	return global.DB.Delete(&backupFile).Error
}

/*
创建数据库恢复操作的任务进度记录
使用当前纳秒级时间戳作为唯一标识 ID
*/
func CreateRestoreProgressDB(progress *models.RestoreProgress) error {
	progress.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	progress.CreatedAt = time.Now()
	progress.UpdatedAt = time.Now()
	return global.DB.Create(progress).Error
}

/*
实时更新恢复任务的百分比进度、状态和提示消息
在任务完成或失败时记录最终完成时间
*/
func UpdateRestoreProgressDB(id string, progress int, status string, message string) error {
	updateData := map[string]interface{}{
		"progress":   progress,
		"status":     status,
		"message":    message,
		"updated_at": time.Now(),
	}
	if status == "completed" || status == "failed" {
		now := time.Now()
		updateData["completed_at"] = &now
	}
	return global.DB.Model(&models.RestoreProgress{}).Where("id = ?", id).Updates(updateData).Error
}

// CleanupExpiredBackups 后台清理任务：根据保留策略移除陈旧备份（导出）
// 支持按日期过期清理和按最大文件数量限制清理
func CleanupExpiredBackups() {
	var backupConfig models.BackupConfig
	global.DB.First(&backupConfig, 1)
	if backupConfig.BackupRetentionDays > 0 {
		expirationDate := time.Now().AddDate(0, 0, -backupConfig.BackupRetentionDays)
		var expiredBackups []models.BackupFile
		global.DB.Where("created_at < ?", expirationDate).Find(&expiredBackups)
		for _, backup := range expiredBackups {
			if backup.FilePath != "" {
				os.Remove(backup.FilePath)
			}
			global.DB.Delete(backup)
		}
		if len(expiredBackups) > 0 {
			log.Printf("[Backup] 已清理 %d 个过期备份文件", len(expiredBackups))
		}
	}
	if backupConfig.BackupMaxCount > 0 {
		var totalCount int64
		global.DB.Model(&models.BackupFile{}).Count(&totalCount)
		if totalCount > int64(backupConfig.BackupMaxCount) {
			var oldestBackups []models.BackupFile
			global.DB.Order("created_at ASC").Limit(int(totalCount - int64(backupConfig.BackupMaxCount))).Find(&oldestBackups)
			for _, backup := range oldestBackups {
				if backup.FilePath != "" {
					os.Remove(backup.FilePath)
				}
				global.DB.Delete(backup)
			}
			log.Printf("[Backup] 已清理超出数量限制的备份文件，保留最新的 %d 个", backupConfig.BackupMaxCount)
		}
	}
}

/*
获取 SQLite 数据库的底层运行状态
包括文件大小、表统计、行数统计及最近备份记录
*/
func GetDatabaseStatusDB() models.DatabaseStatus {
	var status models.DatabaseStatus
	if global.DB != nil {
		status.Online = true
	} else {
		status.Online = false
		return status
	}
	var pageCount, pageSize int64
	global.DB.Raw("SELECT page_count FROM pragma_page_count()").Scan(&pageCount)
	global.DB.Raw("SELECT page_size FROM pragma_page_size()").Scan(&pageSize)
	size := pageCount * pageSize
	if size < 1024 {
		status.Size = fmt.Sprintf("%d Bytes", size)
	} else if size < 1024*1024 {
		status.Size = fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		status.Size = fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	} else {
		status.Size = fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	}
	var totalRows int64 = 0
	var tables []string
	global.DB.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tables)
	for _, table := range tables {
		var count int64
		global.DB.Table(table).Count(&count)
		totalRows += count
	}
	status.Rows = fmt.Sprintf("%d", totalRows)
	var tableCount int64
	global.DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tableCount)
	status.TableCount = int(tableCount)
	var lastBackup models.BackupFile
	global.DB.Order("created_at DESC").Limit(1).Find(&lastBackup)
	if lastBackup.ID != 0 {
		status.LastBackupID = lastBackup.ID
		status.LastBackupAt = lastBackup.CreatedAt.Format("2006-01-02 15:04:05")
	} else {
		status.LastBackupAt = "无备份"
	}
	var verifiedCount int64
	global.DB.Model(&models.BackupFile{}).Where("verified = ?", true).Count(&verifiedCount)
	var totalBackupCount int64
	global.DB.Model(&models.BackupFile{}).Count(&totalBackupCount)
	if totalBackupCount > 0 {
		status.BackupHealth = int(float64(verifiedCount) / float64(totalBackupCount) * 100)
	} else {
		status.BackupHealth = 0
	}
	return status
}

/*
强制检查并自我修复数据库核心配置项
确保系统关键运行参数即使在数据库记录被误删后也能自动恢复
*/
func CheckDatabaseData() {
	var settings models.GlobalSetting
	err := global.DB.Limit(1).Find(&settings, 1).Error
	if err != nil || settings.ID == 0 {
		log.Println("[Database] 未发现核心配置，正在执行数据修复与初始化...")
		newSettings := models.GlobalSetting{
			ID:                  1,
			ViewAuthEnabled:     false,
			ViewAuthKey:         "888888",
			AdminAuthKey:        "123456",
			NodeRefreshInterval: 2,
			MaxLogCount:         100,
			AdminTheme:          models.ThemeDefault,
			MonitorTheme:        models.ThemeDefault,
			ThemeMode:           models.ThemeModeFixed,
		}
		global.DB.Create(&newSettings)
		global.DB.Create(&models.SiteConfig{
			ID:         1,
			SiteName:   "SLITE MONITOR",
			SiteSlogan: "REAL-TIME SYSTEM SURVEILLANCE",
			Version:    "v0.1.0",
		})
		global.DB.Create(&models.AlertConfig{
			ID: 1,
			Engine: models.AlertEngine{
				Enabled:     true,
				RateLimit:   10,
				TargetNodes: []string{},
				QuietStart:  "01:00",
				QuietEnd:    "06:00",
			},
			Metrics: models.AlertMetrics{
				CPU:         90.0,
				Memory:      85.0,
				Disk:        90.0,
				Load:        4.0,
				Inode:       85.0,
				Connections: 5000,
				MonthUp:     1000,
				MonthDown:   2000,
				ProcessName: "",
				PortService: "",
			},
			Switches: models.AlertSwitches{
				NodeOffline:    true,
				ServiceRestart: true,
				Swap:           true,
				TimeDiff:       true,
			},
			Channels: models.AlertChannels{
				Telegram: models.TGConfig{
					Enabled: false,
				},
			},
		})
		global.DB.FirstOrCreate(&models.NodeGroup{ID: 1}, models.NodeGroup{Name: "默认分组"})
		global.DB.Create(&models.BackupConfig{
			ID:                        1,
			AutoBackupEnabled:         false,
			BackupStrategy:            models.BackupStrategyDailyFull,
			BackupRetentionDays:       30,
			BackupMaxCount:            100,
			BackupScheduleTime:        "02:00",
			BackupEncryptionEnabled:   true,
			BackupEncryptionAlgorithm: models.EncryptionAES256GCM,
			BackupVerifyIntegrity:     true,
			BackupCompressionEnabled:  true,
			CreatedAt:                 time.Now(),
			UpdatedAt:                 time.Now(),
		})
		global.DB.Create(&models.ThemeSetting{
			ID:           1,
			AdminTheme:   models.ThemeDefault,
			MonitorTheme: models.ThemeDefault,
			ThemeMode:    models.ThemeModeFixed,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		})
		if err := os.MkdirAll("backups", 0755); err != nil {
			log.Printf("[Database] 创建备份目录失败: %v", err)
		}
		log.Println("[Database] 初始化完成：管理密钥 123456 / 访问密钥 888888")
	}
}

/*
获取当前系统的外观主题配置 (数据库底层函数)
注意：此处返回的是用户选择的主题设置，而不是主题样式定义
*/
func GetThemeConfigDB() models.ThemeSetting {
	var config models.ThemeSetting
	global.DB.First(&config, 1)
	return config
}

/*
保存全局设置的更新内容
*/
func SaveGlobalSettings(s *models.GlobalSetting) error {
	return global.DB.Model(&models.GlobalSetting{}).Where("id = ?", 1).Updates(s).Error
}

/*
更新系统外观主题配置 (数据库底层函数)
并同步刷新更新时间戳
注意：此处操作的是用户选择的主题设置（ThemeSetting）
*/
func SaveThemeConfigDB(t *models.ThemeSetting) error {
	t.UpdatedAt = time.Now()
	return global.DB.Model(&models.ThemeSetting{}).Where("id = ?", 1).Updates(t).Error
}
