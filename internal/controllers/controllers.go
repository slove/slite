package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"slite/internal/global"
	"slite/internal/models"
	"slite/internal/monitor"
)

// GetUptimeStats 获取指定节点的热力图数据、在线率统计及累计监控天数
func GetUptimeStats(nodeID string) ([]models.HeatMapItem, float64, string, int) {
	var totalDays int64
	global.DB.Model(&models.UptimeDaily{}).Where("node_id = ?", nodeID).Count(&totalDays)

	var logs []models.UptimeDaily
	global.DB.Where("node_id = ?", nodeID).Order("date desc").Limit(30).Find(&logs)

	var heatmap []models.HeatMapItem
	var totalRate float64

	for i := len(logs) - 1; i >= 0; i-- {
		heatmap = append(heatmap, models.HeatMapItem{
			Date:            logs[i].Date,
			Status:          logs[i].Status,
			UptimeRate:      logs[i].Rate,
			OfflineDuration: logs[i].OfflineDuration,
			DisplayInfo:     fmt.Sprintf("%s | 在线率: %.1f%%", logs[i].Date, logs[i].Rate),
		})
		totalRate += logs[i].Rate
	}

	if len(logs) == 0 {
		return []models.HeatMapItem{}, 0.0, "0s", int(totalDays)
	}

	avgRate := totalRate / float64(len(logs))
	offlineTotalDuration := "0s"
	if len(logs) > 0 {
		offlineTotalDuration = logs[0].OfflineDuration
	}

	return heatmap, avgRate, offlineTotalDuration, int(totalDays)
}

// SyncNodeStatsDays 统计数据库中节点的历史记录总数并更新到节点配置表
func SyncNodeStatsDays(nodeID string) int {
	var count int64
	global.DB.Model(&models.UptimeDaily{}).Where("node_id = ?", nodeID).Count(&count)
	finalCount := int(count)
	if finalCount == 0 {
		finalCount = 1
	}
	global.DB.Model(&models.NodeConfig{}).Where("node_id = ?", nodeID).Update("total_stats_days", finalCount)
	return finalCount
}

// SendTelegramMessage 构造 Telegram API 请求并以异步协程方式发送告警通知
func SendTelegramMessage(token string, chatID string, text string) {
	token = strings.TrimSpace(token)
	chatID = strings.TrimSpace(chatID)

	if token == "" || chatID == "" {
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	jsonBody, _ := json.Marshal(payload)

	go func() {
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			global.AddSystemLog(models.LogTypeAlert, "error", "Telegram 发送失败响应: "+string(body))
		}
	}()
}

// CanSendAlert 基于配置的频率限制检查特定节点是否可以发送新的告警
func CanSendAlert(nodeID string, intervalMinutes int) bool {
	global.AlertMu.Lock()
	defer global.AlertMu.Unlock()

	lastTime, exists := global.LastAlertMap[nodeID]
	if !exists || time.Since(lastTime) > time.Duration(intervalMinutes)*time.Minute {
		global.LastAlertMap[nodeID] = time.Now()
		return true
	}
	return false
}

// StartGlobalNetworkDetection 周期性执行探测任务并将结果分发至各个节点的内存缓存
func StartGlobalNetworkDetection() {
	exec := func() {
		var activeTasks []models.Task
		global.DB.Where("is_active = ?", true).Order("sort_index ASC, id ASC").Find(&activeTasks)
		if len(activeTasks) == 0 {
			return
		}

		global.NodesMu.RLock()
		nodeIDs := make([]string, 0, len(global.Nodes))
		for id := range global.Nodes {
			nodeIDs = append(nodeIDs, id)
		}
		global.NodesMu.RUnlock()

		for _, nid := range nodeIDs {
			go func(targetNodeID string) {
				var results []models.DetectTask
				for _, t := range activeTasks {
					shouldExecute := false
					if len(t.NodeIDs) == 0 {
						shouldExecute = true
					} else {
						for _, assignedID := range t.NodeIDs {
							if assignedID == targetNodeID {
								shouldExecute = true
								break
							}
						}
					}

					if !shouldExecute {
						continue
					}

					delay, loss := monitor.DoDetection(t)

					go func(taskID uint64, nID string, d float64) {
						global.DB.Create(&models.TaskHistory{
							TaskID:    taskID,
							NodeID:    nID,
							Delay:     d,
							CreatedAt: time.Now(),
						})
					}(t.ID, targetNodeID, delay)

					global.HistoryMu.Lock()
					hKey := fmt.Sprintf("%s_%d", targetNodeID, t.ID)
					taskHistoryNew, ok := global.TaskHistoryMap[hKey]
					if !ok {
						taskHistoryNew = make([]float64, 0)
					}
					taskHistoryNew = append(taskHistoryNew, delay)

					if len(taskHistoryNew) > 100 {
						taskHistoryNew = taskHistoryNew[1:]
					}
					global.TaskHistoryMap[hKey] = taskHistoryNew

					histCopy := make([]float64, len(taskHistoryNew))
					copy(histCopy, taskHistoryNew)
					global.HistoryMu.Unlock()

					results = append(results, models.DetectTask{
						ID:        t.ID,
						Name:      t.Name,
						Type:      t.Type,
						Target:    t.Target,
						Port:      t.Port,
						LastDelay: delay,
						LossRate:  loss,
						SortIndex: t.SortIndex,
						NodeIDs:   t.NodeIDs,
						History:   histCopy,
					})
				}

				global.NodesMu.Lock()
				if n, ok := global.Nodes[targetNodeID]; ok {
					n.Tasks = results
				}
				global.NodesMu.Unlock()
			}(nid)
		}
	}

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			exec()
			<-ticker.C
		}
	}()
}

// CheckOffline 周期性扫描节点状态，判断是否离线并触发上下线通知
func CheckOffline() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		global.NodesMu.Lock()

		// 直接查询数据库获取告警配置
		var config models.AlertConfig
		if err := global.DB.First(&config, 1).Error; err != nil {
			global.NodesMu.Unlock()
			continue
		}

		for nodeID, node := range global.Nodes {
			isThresholdExceeded := now.Sub(node.LastSeen) > 15*time.Second

			if isThresholdExceeded && node.Online {
				node.Online = false
				global.AddSystemLog(models.LogTypeHost, "error", fmt.Sprintf("检测到节点离线: [%s] (%s)", node.Name, nodeID))

				shouldAlert := true
				if len(config.Engine.TargetNodes) > 0 {
					found := false
					for _, targetID := range config.Engine.TargetNodes {
						if targetID == nodeID {
							found = true
							break
						}
					}
					shouldAlert = found
				}

				if shouldAlert && config.Engine.Enabled && config.Channels.Telegram.Enabled && config.Switches.NodeOffline {
					msg := fmt.Sprintf("<b>节点离线通知</b>\n━━━━━━━━━━━━━━━\n<b>节点:</b> %s\n<b>状态:</b> 失去联系\n<b>时间:</b> %s", node.Name, now.Format("15:04:05"))
					SendTelegramMessage(config.Channels.Telegram.BotToken, config.Channels.Telegram.ChatID, msg)
				}
			} else if !isThresholdExceeded && !node.Online {
				node.Online = true
				global.AddSystemLog(models.LogTypeHost, "info", fmt.Sprintf("节点已恢复上线: [%s] (%s)", node.Name, nodeID))

				shouldAlert := true
				if len(config.Engine.TargetNodes) > 0 {
					found := false
					for _, targetID := range config.Engine.TargetNodes {
						if targetID == nodeID {
							found = true
							break
						}
					}
					shouldAlert = found
				}

				if shouldAlert && config.Engine.Enabled && config.Channels.Telegram.Enabled && config.Switches.NodeOffline {
					msg := fmt.Sprintf("<b>节点上线恢复</b>\n━━━━━━━━━━━━━━━\n<b>节点:</b> %s\n<b>状态:</b> 已恢复在线\n<b>时间:</b> %s", node.Name, now.Format("15:04:05"))
					SendTelegramMessage(config.Channels.Telegram.BotToken, config.Channels.Telegram.ChatID, msg)
				}
			}
		}
		global.NodesMu.Unlock()
	}
}

// StartHistoryWorker 每小时记录一次所有节点的当前在线状态到每日统计表
func StartHistoryWorker() {
	runHistoryRecord := func() {
		now := time.Now()
		todayStr := now.Format("2006-01-02")
		global.NodesMu.RLock()
		for id, node := range global.Nodes {
			var daily models.UptimeDaily
			status := "online"
			if !node.Online {
				status = "offline"
			}
			result := global.DB.Where("node_id = ? AND date = ?", id, todayStr).First(&daily)
			if result.Error != nil {
				global.DB.Create(&models.UptimeDaily{NodeID: id, Date: todayStr, Status: status, Rate: 100.0})
			} else {
				global.DB.Model(&daily).Update("status", status)
			}
		}
		global.NodesMu.RUnlock()
	}
	runHistoryRecord()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		runHistoryRecord()
	}
}

// StartWSBroadcast WebSocket 广播中心，负责聚合系统全局状态并实时推送到所有已连接的客户端
func StartWSBroadcast() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var configs []models.NodeConfig
		global.DB.Order("sort_index ASC, node_id ASC").Find(&configs)

		global.NodesMu.RLock()
		serverList := make([]*models.Node, 0)
		for _, cfg := range configs {
			if !cfg.IsVisible {
				continue
			}

			var displayNode models.Node
			if n, exists := global.Nodes[cfg.ID]; exists {
				displayNode = *n
			} else {
				displayNode = models.Node{
					ID:     cfg.ID,
					Online: false,
					Tasks:  make([]models.DetectTask, 0),
				}
			}

			displayNode.Name = cfg.CustomName
			displayNode.Location = cfg.Location
			if cfg.IP != "" && cfg.IP != "0.0.0.0" {
				displayNode.IP = cfg.IP
			}
			displayNode.SortIndex = cfg.SortIndex
			displayNode.MonthUp = cfg.MonthUp
			displayNode.MonthDown = cfg.MonthDown
			displayNode.UptimeRate = cfg.UptimeRate
			displayNode.OfflineTotal = cfg.OfflineTotal

			serverList = append(serverList, &displayNode)
		}
		global.NodesMu.RUnlock()

		var settings models.GlobalSetting
		global.DB.First(&settings, 1)
		var config models.SiteConfig
		global.DB.First(&config, 1)

		payload := gin.H{
			"servers":         serverList,
			"global_settings": settings,
			"site_config":     config,
			"ws_status":       fmt.Sprintf("最后同步: %s", time.Now().Format("15:04:05")),
		}

		global.ClientsMu.Lock()
		for client := range global.Clients {
			err := client.WriteJSON(payload)
			if err != nil {
				client.Close()
				delete(global.Clients, client)
			}
		}
		global.ClientsMu.Unlock()
	}
}

// FormatBool 格式化布尔值为中文
func FormatBool(b bool) string {
	if b {
		return "开启"
	}
	return "关闭"
}
