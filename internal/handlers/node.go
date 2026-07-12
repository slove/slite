package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"slite/internal/alert"
	"slite/internal/global"
	"slite/internal/models"
)

// HandleAgentReport 处理Agent上报数据
func HandleAgentReport(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "heartbeat"})
		return
	}

	var report models.NodeReport
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error"})
		return
	}

	var conf models.NodeConfig
	if res := global.DB.Where("node_id = ?", report.ID).Limit(1).Find(&conf); res.RowsAffected == 0 {
		var maxSort int
		global.DB.Model(&models.NodeConfig{}).Select("COALESCE(MAX(sort_index), 0)").Scan(&maxSort)

		conf = models.NodeConfig{
			ID:             report.ID,
			GroupID:        1,
			CustomName:     report.Name,
			Location:       report.Location,
			IP:             report.IP,
			IsVisible:      true,
			UptimeRate:     100.0,
			SortIndex:      maxSort + 1,
			TotalStatsDays: 1,
		}
		global.DB.Create(&conf)
		global.AddSystemLog(models.LogTypeHost, "info", fmt.Sprintf("发现新节点自动注册: [%s] (%s)", report.Name, report.ID))
	} else {
		shouldUpdate := false
		updateFields := make(map[string]interface{})

		if (conf.Location == "" || conf.Location == "未知") && report.Location != "" && report.Location != "未知" {
			updateFields["location"] = report.Location
			shouldUpdate = true
		}

		if conf.Location != "" && report.Location != "" &&
			conf.Location != "未知" && report.Location != "未知" &&
			conf.Location != report.Location {
			updateFields["location"] = report.Location
			shouldUpdate = true
		}

		if shouldUpdate && len(updateFields) > 0 {
			global.DB.Model(&models.NodeConfig{}).Where("node_id = ?", report.ID).Updates(updateFields)
			conf.Location = report.Location
		}
	}

	updateTodayUptimeRecord(report.ID, true)

	global.DB.Where("node_id = ?", report.ID).Limit(1).Find(&conf)

	finalIP := conf.IP
	if finalIP == "" || finalIP == "0.0.0.0" {
		if report.IP != "" && report.IP != "0.0.0.0" {
			finalIP = report.IP
		} else {
			finalIP = c.ClientIP()
		}
	}

	global.NodesMu.Lock()
	prevNode, exists := global.Nodes[report.ID]

	var diffUp, diffDown uint64
	if exists {
		if report.NetOut > prevNode.NetOut && prevNode.NetOut > 0 {
			diffUp = report.NetOut - prevNode.NetOut
		}
		if report.NetIn > prevNode.NetIn && prevNode.NetIn > 0 {
			diffDown = report.NetIn - prevNode.NetIn
		}
	}

	currentTasks := loadNodeTasks(report.ID)

	global.Nodes[report.ID] = &models.Node{
		ID:             report.ID,
		GroupID:        conf.GroupID,
		Name:           report.Name,
		Location:       report.Location,
		IP:             finalIP,
		OS:             report.OS,
		CPU:            report.CPU,
		Mem:            report.Mem,
		Disk:           report.Disk,
		ProcessCount:   report.ProcessCount,
		Load:           report.Load,
		Uptime:         report.Uptime,
		Up:             report.Up,
		Down:           report.Down,
		NetIn:          report.NetIn,
		NetOut:         report.NetOut,
		Docker:         report.Docker,
		Online:         true,
		IsVisible:      conf.IsVisible,
		LastSeen:       time.Now(),
		Tasks:          currentTasks,
		SortIndex:      conf.SortIndex,
		TotalStatsDays: conf.TotalStatsDays,
	}
	global.NodesMu.Unlock()

	if diffUp > 0 || diffDown > 0 {
		global.DB.Model(&models.NodeConfig{}).Where("node_id = ?", report.ID).Updates(map[string]interface{}{
			"month_up":   gorm.Expr("month_up + ?", diffUp),
			"month_down": gorm.Expr("month_down + ?", diffDown),
		})
	}

	go alert.ExecuteAlertCheck(report.ID, report)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// HandleAgentUploadBasicInfo 处理Agent上传基础信息
func HandleAgentUploadBasicInfo(c *gin.Context) {
	var input struct {
		ID       string `json:"id"`
		OS       string `json:"os"`
		CPUModel string `json:"cpu_model"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid Info Format"})
		return
	}

	global.NodesMu.Lock()
	if node, ok := global.Nodes[input.ID]; ok {
		if input.OS != "" {
			node.OS = input.OS
		}
	}
	global.NodesMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// HandleAgentTasks 处理Agent获取任务列表的请求
func HandleAgentTasks(c *gin.Context) {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		nodeID = c.GetHeader("X-Node-ID")
	}

	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Missing node ID",
		})
		return
	}

	tasks := loadNodeTasks(nodeID)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"tasks":  tasks,
	})
}

// GetNodeGroups 获取节点分组列表
func GetNodeGroups(c *gin.Context) {
	var groups []models.NodeGroup
	global.DB.Order("sort_index ASC, id ASC").Find(&groups)
	c.JSON(http.StatusOK, groups)
}

// CreateNodeGroup 创建节点分组
func CreateNodeGroup(c *gin.Context) {
	var group models.NodeGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "INVALID FORMAT"})
		return
	}

	if group.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "NAME REQUIRED"})
		return
	}

	global.DB.Create(&group)
	global.AddSystemLog(models.LogTypeAudit, "info", fmt.Sprintf("创建新节点分组: %s", group.Name))
	c.JSON(http.StatusOK, group)
}

// UpdateNodeGroup 更新节点分组
func UpdateNodeGroup(c *gin.Context) {
	id := c.Param("id")
	var group models.NodeGroup
	if err := global.DB.First(&group, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "GROUP NOT FOUND"})
		return
	}

	var input models.NodeGroup
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "INVALID FORMAT"})
		return
	}

	global.DB.Model(&group).Updates(map[string]interface{}{
		"name":       input.Name,
		"sort_index": input.SortIndex,
	})

	c.JSON(http.StatusOK, group)
}

// DeleteNodeGroup 删除节点分组
func DeleteNodeGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "1" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "CANNOT DELETE DEFAULT GROUP"})
		return
	}

	global.DB.Model(&models.NodeConfig{}).Where("group_id = ?", id).Update("group_id", 1)
	global.DB.Delete(&models.NodeGroup{}, id)

	global.AddSystemLog(models.LogTypeAudit, "info", fmt.Sprintf("删除了节点分组 ID: %s", id))
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GetNodesForAlert 获取可用于告警的节点列表
func GetNodesForAlert(c *gin.Context) {
	var nodes []models.NodeConfig
	global.DB.Order("sort_index ASC, node_id ASC").Find(&nodes)

	var nodeList []gin.H
	for _, node := range nodes {
		nodeList = append(nodeList, gin.H{
			"id":   node.ID,
			"name": node.CustomName,
		})
	}

	c.JSON(http.StatusOK, nodeList)
}

// loadNodeTasks 加载指定节点的任务列表
func loadNodeTasks(nodeID string) []models.DetectTask {
	var tasks []models.DetectTask

	var dbTasks []models.Task
	global.DB.Order("sort_index ASC, id ASC").Find(&dbTasks)

	for _, dbTask := range dbTasks {
		shouldAssign := false
		if len(dbTask.NodeIDs) == 0 {
			shouldAssign = true
		} else {
			for _, assignedNodeID := range dbTask.NodeIDs {
				if assignedNodeID == nodeID {
					shouldAssign = true
					break
				}
			}
		}

		if shouldAssign && dbTask.IsActive {
			var history []models.TaskHistory
			global.DB.Where("node_id = ? AND task_id = ?", nodeID, dbTask.ID).
				Order("created_at DESC").
				Limit(30).
				Find(&history)

			var delayHistory []float64
			for _, h := range history {
				delayHistory = append(delayHistory, h.Delay)
			}

			for i, j := 0, len(delayHistory)-1; i < j; i, j = i+1, j-1 {
				delayHistory[i], delayHistory[j] = delayHistory[j], delayHistory[i]
			}

			var lastDelay float64
			if len(delayHistory) > 0 {
				lastDelay = delayHistory[len(delayHistory)-1]
			}

			lossRate := 0.0
			if len(delayHistory) > 0 {
				lossCount := 0
				for _, d := range delayHistory {
					if d <= 0 {
						lossCount++
					}
				}
				lossRate = float64(lossCount) / float64(len(delayHistory)) * 100
			}

			tasks = append(tasks, models.DetectTask{
				ID:        dbTask.ID,
				Name:      dbTask.Name,
				Type:      dbTask.Type,
				Target:    dbTask.Target,
				Port:      dbTask.Port,
				LastDelay: lastDelay,
				LossRate:  lossRate,
				SortIndex: dbTask.SortIndex,
				NodeIDs:   dbTask.NodeIDs,
				History:   delayHistory,
			})
		}
	}

	if len(tasks) == 0 {
		tasks = getDefaultNetworkTasks(nodeID)
	}

	return tasks
}

// getDefaultNetworkTasks 获取默认网络监控任务
func getDefaultNetworkTasks(nodeID string) []models.DetectTask {
	defaultTasks := []struct {
		Name   string
		Type   string
		Target string
		Port   int
	}{
		{"Google DNS (ICMP)", "icmp", "8.8.8.8", 0},
		{"Google TCP Ping", "tcp", "www.google.com", 443},
		{"Google HTTPS", "http", "https://www.google.com", 0},
	}

	var tasks []models.DetectTask
	for i, d := range defaultTasks {
		taskID := uint64(time.Now().UnixNano()/1e6) + uint64(i)
		tasks = append(tasks, models.DetectTask{
			ID:        taskID,
			Name:      d.Name,
			Type:      d.Type,
			Target:    d.Target,
			Port:      d.Port,
			LastDelay: 0,
			LossRate:  0,
			SortIndex: i,
			NodeIDs:   []string{nodeID},
			History:   []float64{},
		})
	}

	return tasks
}

// updateTodayUptimeRecord 更新当日在线状态记录
func updateTodayUptimeRecord(nodeID string, isOnline bool) {
	today := time.Now().Format("2006-01-02")

	var record models.UptimeDaily
	result := global.DB.Where("node_id = ? AND date = ?", nodeID, today).First(&record)

	if result.Error != nil {
		status := "online"
		rate := 100.0
		if !isOnline {
			status = "offline"
			rate = 0.0
		}

		record = models.UptimeDaily{
			NodeID:          nodeID,
			Date:            today,
			Status:          status,
			Rate:            rate,
			OfflineDuration: "0s",
		}
		global.DB.Create(&record)
	} else {
		status := "online"
		rate := 100.0
		if !isOnline {
			status = "offline"
			rate = 0.0
		}

		global.DB.Model(&record).Where("node_id = ? AND date = ?", nodeID, today).Updates(map[string]interface{}{
			"status": status,
			"rate":   rate,
		})
	}
}
