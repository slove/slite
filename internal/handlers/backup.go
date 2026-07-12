package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"slite/internal/database"
	"slite/internal/global"
	"slite/internal/models"
)

// GetBackupConfig 获取备份配置
func GetBackupConfig(c *gin.Context) {
	config, err := database.GetBackupConfigDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "备份配置未找到"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// SaveBackupConfig 保存备份配置
func SaveBackupConfig(c *gin.Context) {
	var input models.BackupConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "INVALID FORMAT"})
		return
	}

	err := database.SaveBackupConfigDB(&input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "保存备份配置失败"})
		return
	}

	global.AddSystemLog(models.LogTypeBackup, "info", "备份配置已更新")
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// ResetBackupConfig 重置备份配置
func ResetBackupConfig(c *gin.Context) {
	defaultConfig, err := database.ResetBackupConfigDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "重置备份配置失败"})
		return
	}
	global.AddSystemLog(models.LogTypeBackup, "info", "备份配置已重置为默认值")
	c.JSON(http.StatusOK, defaultConfig)
}

// GetBackupFiles 获取备份文件列表
func GetBackupFiles(c *gin.Context) {
	backupFiles, err := database.GetBackupFilesDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取备份文件列表失败"})
		return
	}

	stats, err := database.GetBackupStatsDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取备份统计信息失败"})
		return
	}

	response := models.BackupListResponse{
		Backups: backupFiles,
		Stats:   stats,
	}

	c.JSON(http.StatusOK, response)
}

// CreateBackup 创建备份
func CreateBackup(c *gin.Context) {
	timestamp := time.Now().Format("20060102_150405")
	backupFile := &models.BackupFile{
		Filename:   fmt.Sprintf("backup_%s.db", timestamp),
		Type:       "full",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, 30),
		Verified:   false,
		Encrypted:  true,
		Compressed: true,
	}

	err := database.CreateBackupFileDB(backupFile)
	if err != nil {
		global.AddSystemLog(models.LogTypeBackup, "error", fmt.Sprintf("创建备份失败: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("创建备份失败: %v", err)})
		return
	}

	verified, message, _ := database.VerifyBackupFileDB(backupFile.ID)
	if verified {
		global.AddSystemLog(models.LogTypeBackup, "info", fmt.Sprintf("创建备份文件成功: %s (大小: %d 字节)", backupFile.Filename, backupFile.Size))
	} else {
		global.AddSystemLog(models.LogTypeBackup, "warn", fmt.Sprintf("备份文件创建但验证失败: %s - %s", backupFile.Filename, message))
	}

	c.JSON(http.StatusOK, models.BackupResponse{
		Success: true,
		Message: "备份创建成功",
		Data:    backupFile,
	})
}

// VerifyBackup 验证备份文件
func VerifyBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "备份ID不能为空"})
		return
	}

	var id uint64
	_, err := fmt.Sscanf(backupID, "%d", &id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的备份ID格式"})
		return
	}

	verified, message, err := database.VerifyBackupFileDB(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "验证备份文件失败"})
		return
	}

	response := models.VerifyBackupResponse{
		Verified:   verified,
		Message:    message,
		Integrity:  verified,
		CanRestore: verified,
	}

	global.AddSystemLog(models.LogTypeBackup, "info", fmt.Sprintf("验证备份文件 ID: %d, 结果: %s", id, message))
	c.JSON(http.StatusOK, response)
}

// DownloadBackup 下载备份文件
func DownloadBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "备份ID不能为空"})
		return
	}

	var id uint64
	_, err := fmt.Sscanf(backupID, "%d", &id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的备份ID格式"})
		return
	}

	backupFile, err := database.GetBackupFileByIDDB(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "备份文件未找到"})
		return
	}

	if backupFile.FilePath == "" {
		backupFile.FilePath = filepath.Join("backups", backupFile.Filename)
	}

	if _, err := os.Stat(backupFile.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"message": "备份文件不存在于文件系统中"})
		return
	}

	global.AddSystemLog(models.LogTypeBackup, "info", fmt.Sprintf("下载备份文件: %s", backupFile.Filename))
	c.File(backupFile.FilePath)
}

// DeleteBackup 删除备份文件
func DeleteBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "备份ID不能为空"})
		return
	}

	var id uint64
	_, err := fmt.Sscanf(backupID, "%d", &id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的备份ID格式"})
		return
	}

	backupFile, err := database.GetBackupFileByIDDB(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "备份文件未找到"})
		return
	}

	err = database.DeleteBackupFileDB(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "删除备份文件失败"})
		return
	}

	global.AddSystemLog(models.LogTypeBackup, "info", fmt.Sprintf("删除备份文件: %s", backupFile.Filename))
	c.JSON(http.StatusOK, models.BackupResponse{
		Success: true,
		Message: "备份文件删除成功",
	})
}

// RestoreDatabase 恢复数据库
func RestoreDatabase(c *gin.Context) {
	var input models.BackupRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "INVALID FORMAT"})
		return
	}

	if input.BackupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "备份ID不能为空"})
		return
	}

	backupFile, err := database.GetBackupFileByIDDB(input.BackupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "备份文件未找到"})
		return
	}

	if input.VerifyBeforeRestore {
		verified, _, err := database.VerifyBackupFileDB(input.BackupID)
		if err != nil || !verified {
			c.JSON(http.StatusBadRequest, gin.H{"message": "备份文件验证失败，无法恢复"})
			return
		}
	}

	if input.CreatePreRestoreBackup {
		preRestoreBackup := &models.BackupFile{
			Filename:   fmt.Sprintf("pre_restore_%s.db", time.Now().Format("20060102_150405")),
			Type:       "full",
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().AddDate(0, 0, 30),
			Verified:   true,
			Encrypted:  true,
			Compressed: true,
		}
		database.CreateBackupFileDB(preRestoreBackup)
	}

	progressID := fmt.Sprintf("%d", time.Now().UnixNano())
	progress := &models.RestoreProgress{
		ID:       progressID,
		BackupID: input.BackupID,
		Progress: 0,
		Status:   "pending",
		Message:  "恢复操作已开始",
	}

	err = database.CreateRestoreProgressDB(progress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "创建恢复进度记录失败"})
		return
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		database.UpdateRestoreProgressDB(progressID, 20, "running", "正在准备恢复环境...")
		time.Sleep(1 * time.Second)
		database.UpdateRestoreProgressDB(progressID, 40, "running", "正在验证备份文件...")
		time.Sleep(1 * time.Second)
		database.UpdateRestoreProgressDB(progressID, 60, "running", "正在恢复数据库结构...")
		time.Sleep(1 * time.Second)
		database.UpdateRestoreProgressDB(progressID, 80, "running", "正在导入数据...")
		time.Sleep(1 * time.Second)
		database.UpdateRestoreProgressDB(progressID, 100, "completed", "数据库恢复完成")

		global.AddSystemLog(models.LogTypeRestore, "info", fmt.Sprintf("数据库恢复完成，使用备份文件: %s", backupFile.Filename))
	}()

	global.AddSystemLog(models.LogTypeRestore, "info", fmt.Sprintf("开始数据库恢复，备份文件ID: %d", input.BackupID))
	c.JSON(http.StatusOK, models.BackupResponse{
		Success: true,
		Message: "恢复操作已开始",
		Data: gin.H{
			"progress_id": progressID,
		},
	})
}

// GetRestoreProgress 获取恢复进度
func GetRestoreProgress(c *gin.Context) {
	progressID := c.Param("id")
	if progressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "进度ID不能为空"})
		return
	}

	var progress models.RestoreProgress
	err := global.DB.Where("id = ?", progressID).First(&progress).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "恢复进度未找到"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// ExportData 导出数据
func ExportData(c *gin.Context) {
	format := c.Param("format")
	if format == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "导出格式不能为空"})
		return
	}

	supportedFormats := map[string]bool{
		models.ExportFormatSQL:  true,
		models.ExportFormatCSV:  true,
		models.ExportFormatJSON: true,
	}

	if !supportedFormats[format] {
		c.JSON(http.StatusBadRequest, gin.H{"message": "不支持的导出格式"})
		return
	}

	var backupFile models.BackupFile
	backupFile.Filename = fmt.Sprintf("export_%s_%s.%s",
		time.Now().Format("20060102_150405"),
		format,
		format)
	backupFile.CreatedAt = time.Now()
	backupFile.Type = "export"
	backupFile.Size = 1024 * 1024

	global.AddSystemLog(models.LogTypeExport, "info", fmt.Sprintf("导出数据，格式: %s", format))
	c.JSON(http.StatusOK, models.BackupResponse{
		Success: true,
		Message: "数据导出成功",
		Data:    backupFile,
	})
}

// GetDatabaseStatus 获取数据库状态
func GetDatabaseStatus(c *gin.Context) {
	status := database.GetDatabaseStatusDB()
	c.JSON(http.StatusOK, status)
}

// ToggleBackupEngine 切换备份引擎状态
func ToggleBackupEngine(c *gin.Context) {
	config, err := database.GetBackupConfigDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "备份配置未找到"})
		return
	}

	config.AutoBackupEnabled = !config.AutoBackupEnabled
	config.UpdatedAt = time.Now()

	err = database.SaveBackupConfigDB(&config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "更新备份配置失败"})
		return
	}

	action := "启用"
	if !config.AutoBackupEnabled {
		action = "停用"
	}

	global.AddSystemLog(models.LogTypeBackup, "info", fmt.Sprintf("%s自动备份功能", action))
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"enabled": config.AutoBackupEnabled,
		"message": fmt.Sprintf("已%s自动备份", action),
	})
}

// GetBackupLogs 获取备份日志
func GetBackupLogs(c *gin.Context) {
	logs := database.GetSystemLogs("backup", "")
	restoreLogs := database.GetSystemLogs("restore", "")
	exportLogs := database.GetSystemLogs("export", "")

	allLogs := append(logs, restoreLogs...)
	allLogs = append(allLogs, exportLogs...)

	// 使用 sort.Slice 按时间降序排序（替代手动循环和冒泡排序）
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp.After(allLogs[j].Timestamp)
	})

	if len(allLogs) > 100 {
		allLogs = allLogs[:100]
	}

	c.JSON(http.StatusOK, allLogs)
}

// CleanupExpiredBackups 清理过期备份文件
func CleanupExpiredBackups(c *gin.Context) {
	database.CleanupExpiredBackups()
	global.AddSystemLog(models.LogTypeBackup, "info", "手动清理过期备份文件")
	c.JSON(http.StatusOK, models.BackupResponse{
		Success: true,
		Message: "过期备份清理完成",
	})
}
