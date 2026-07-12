package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"slite/internal/database"
	"slite/internal/global"
	"slite/internal/models"
)

// GetTasks 获取所有任务
func GetTasks(c *gin.Context) {
	var tasks []models.Task
	global.DB.Order("sort_index ASC, created_at DESC").Find(&tasks)
	c.JSON(http.StatusOK, tasks)
}

// CreateTask 创建任务
func CreateTask(c *gin.Context) {
	var input struct {
		Name    string   `json:"name" binding:"required"`
		Type    string   `json:"type" binding:"required"`
		Target  string   `json:"target" binding:"required"`
		Port    int      `json:"port"`
		NodeIDs []string `json:"node_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "参数缺失"})
		return
	}

	var maxSort int
	global.DB.Model(&models.Task{}).Select("COALESCE(MAX(sort_index), 0)").Scan(&maxSort)

	newTask := models.Task{
		ID:        uint64(global.GlobalSnowflake.Generate().Int64()),
		Name:      input.Name,
		Type:      input.Type,
		Target:    input.Target,
		Port:      input.Port,
		NodeIDs:   input.NodeIDs,
		SortIndex: maxSort + 1,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	global.DB.Create(&newTask)
	global.AddSystemLog(models.LogTypeAudit, "info", fmt.Sprintf("管理员创建了网络探测任务: %s", input.Name))
	c.JSON(http.StatusOK, gin.H{"status": "success", "id": fmt.Sprintf("%d", newTask.ID)})
}

// ReorderTasks 重新排序任务
func ReorderTasks(c *gin.Context) {
	var input struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "参数绑定失败"})
		return
	}
	for index, id := range input.IDs {
		global.DB.Model(&models.Task{}).Where("id = ?", id).Update("sort_index", index)
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// UpdateTask 更新任务
func UpdateTask(c *gin.Context) {
	idStr := c.Param("id")
	var incoming models.Task
	if err := c.ShouldBindJSON(&incoming); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error"})
		return
	}
	global.DB.Model(&models.Task{}).Where("id = ?", idStr).Updates(map[string]interface{}{
		"name":      incoming.Name,
		"type":      incoming.Type,
		"target":    incoming.Target,
		"port":      incoming.Port,
		"node_ids":  incoming.NodeIDs,
		"is_active": incoming.IsActive,
	})
	global.AddSystemLog(models.LogTypeAudit, "info", fmt.Sprintf("管理员更新了探测任务 [%s] 的配置", incoming.Name))
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DeleteTask 删除任务
func DeleteTask(c *gin.Context) {
	id := c.Param("id")
	global.DB.Where("id = ?", id).Delete(&models.Task{})
	global.DB.Where("task_id = ?", id).Delete(&models.TaskHistory{})
	global.AddSystemLog(models.LogTypeAudit, "warn", fmt.Sprintf("管理员删除了探测任务: %s", id))
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GetTaskHistoryTimeRange 获取任务历史时间范围
func GetTaskHistoryTimeRange(c *gin.Context) {
	nodeID := c.Query("node_id")
	taskIDStr := c.Query("task_id")

	if nodeID == "" || taskIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "参数不完整"})
		return
	}

	taskID, _ := strconv.ParseUint(taskIDStr, 10, 64)
	maxHours := database.GetTaskHistoryTimeRange(nodeID, taskID)

	c.JSON(http.StatusOK, gin.H{
		"max_hours": maxHours,
		"node_id":   nodeID,
		"task_id":   taskID,
	})
}

// GetTaskHistoryData 获取任务历史数据
func GetTaskHistoryData(c *gin.Context) {
	nodeID := c.Query("node_id")
	taskIDStr := c.Query("task_id")
	hoursStr := c.DefaultQuery("hours", "1")

	if nodeID == "" || taskIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "参数不完整"})
		return
	}

	taskID, _ := strconv.ParseUint(taskIDStr, 10, 64)
	hours, _ := strconv.Atoi(hoursStr)
	if hours <= 0 {
		hours = 1
	}

	chartData := database.GetTaskHistoryData(nodeID, taskID, hours)
	c.JSON(http.StatusOK, chartData)
}
