package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"slite/internal/global"
	"slite/internal/models"
)

var lastAlertTimes = make(map[string]map[string]time.Time)
var alertCache = make(map[string]map[string]bool)

// ExecuteAlertCheck 执行告警检查（导出供其他包调用）
func ExecuteAlertCheck(nodeID string, report models.NodeReport) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Alert] 告警检查异常: %v", r)
		}
	}()

	var config models.AlertConfig
	if err := global.DB.First(&config, 1).Error; err != nil {
		return
	}

	if !config.Engine.Enabled {
		return
	}

	isTargetNode := false
	for _, targetID := range config.Engine.TargetNodes {
		if targetID == nodeID {
			isTargetNode = true
			break
		}
	}
	if !isTargetNode {
		return
	}

	// 检查是否在静默时段
	if isQuietTime(config.Engine.QuietStart, config.Engine.QuietEnd) {
		log.Printf("[Alert] 节点 %s 处于静默时段，跳过非紧急告警", nodeID)
		return
	}

	checkResourceAlerts(nodeID, report, &config)
	checkSystemAlerts(nodeID, report, &config)
	checkNetworkAlerts(nodeID, report, &config)
	checkProcessAndPortAlerts(nodeID, report, &config)
}

// isQuietTime 检查是否在静默时段
func isQuietTime(quietStart, quietEnd string) bool {
	if quietStart == "" || quietEnd == "" {
		return false
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	if quietEnd < quietStart {
		return currentTime >= quietStart || currentTime <= quietEnd
	}
	return currentTime >= quietStart && currentTime <= quietEnd
}

func checkResourceAlerts(nodeID string, report models.NodeReport, config *models.AlertConfig) {
	alerts := []string{}

	if report.CPU > config.Metrics.CPU {
		alerts = append(alerts, fmt.Sprintf("CPU使用率过高: %.1f%% > %.0f%%", report.CPU, config.Metrics.CPU))
	}

	if report.Mem > config.Metrics.Memory {
		alerts = append(alerts, fmt.Sprintf("内存使用率过高: %.1f%% > %.0f%%", report.Mem, config.Metrics.Memory))
	}

	if report.Disk > config.Metrics.Disk {
		alerts = append(alerts, fmt.Sprintf("磁盘使用率过高: %.1f%% > %.0f%%", report.Disk, config.Metrics.Disk))
	}

	if len(report.Load) > 0 {
		load1 := report.Load[0]
		if config.Metrics.Load > 0 && load1 > config.Metrics.Load {
			alerts = append(alerts, fmt.Sprintf("系统负载过高: %.2f > %.1f", load1, config.Metrics.Load))
		}
	}

	if len(alerts) > 0 {
		sendAlerts(nodeID, "资源告警", alerts, config)
	}
}

func checkSystemAlerts(nodeID string, report models.NodeReport, config *models.AlertConfig) {
	alerts := []string{}

	// 检查节点离线状态
	if config.Switches.NodeOffline {
		global.NodesMu.RLock()
		node, exists := global.Nodes[nodeID]
		global.NodesMu.RUnlock()

		if exists && !node.Online {
			alerts = append(alerts, "节点通信已断开")
		}
	}

	// 检查服务重启 - 通过进程数变化判断
	if config.Switches.ServiceRestart {
		if restartAlert := checkServiceRestart(nodeID, report); restartAlert != "" {
			alerts = append(alerts, restartAlert)
		}
	}

	// 检查SWAP使用率
	if config.Switches.Swap {
		if swapAlert := checkSwapUsage(nodeID, report); swapAlert != "" {
			alerts = append(alerts, swapAlert)
		}
	}

	// 检查时间偏移
	if config.Switches.TimeDiff {
		if timeDiffAlert := checkTimeDiff(nodeID, config); timeDiffAlert != "" {
			alerts = append(alerts, timeDiffAlert)
		}
	}

	if len(alerts) > 0 {
		sendAlerts(nodeID, "系统告警", alerts, config)
	}
}

// checkServiceRestart 检查服务是否重启过（通过进程启动时间变化判断）
func checkServiceRestart(nodeID string, report models.NodeReport) string {
	global.NodesMu.RLock()
	node, exists := global.Nodes[nodeID]
	global.NodesMu.RUnlock()

	if exists && node.ProcessCount > 0 {
		prevCount := node.ProcessCount
		currentCount := report.ProcessCount

		if prevCount > 0 && currentCount > 0 {
			changeRate := float64(currentCount-prevCount) / float64(prevCount) * 100
			if changeRate < -50 {
				return fmt.Sprintf("进程数异常减少: %d -> %d (变化率: %.1f%%)，可能发生了服务重启",
					prevCount, currentCount, changeRate)
			}
		}
	}
	return ""
}

// checkSwapUsage 检查SWAP使用率
func checkSwapUsage(nodeID string, report models.NodeReport) string {
	if report.Mem > 85 {
		global.NodesMu.RLock()
		node, exists := global.Nodes[nodeID]
		global.NodesMu.RUnlock()

		if exists && node.Uptime > 3600*24 {
			return fmt.Sprintf("内存使用率过高(%.1f%%)，可能引发SWAP使用，建议检查", report.Mem)
		}
	}
	return ""
}

// checkTimeDiff 检查时间偏移
func checkTimeDiff(nodeID string, config *models.AlertConfig) string {
	// 从全局节点缓存中获取节点的 LastSeen
	global.NodesMu.RLock()
	node, exists := global.Nodes[nodeID]
	global.NodesMu.RUnlock()

	if !exists {
		return ""
	}

	serverTime := time.Now()
	nodeTime := node.LastSeen

	if !nodeTime.IsZero() {
		diff := serverTime.Sub(nodeTime)
		if diff < 0 {
			diff = -diff
		}
		// 如果时间差超过30秒，发出告警（可配置）
		if diff > 30*time.Second {
			return fmt.Sprintf("节点时间偏移: 服务器与节点时间差 %.1f 秒", diff.Seconds())
		}
	}
	return ""
}

func checkNetworkAlerts(nodeID string, report models.NodeReport, config *models.AlertConfig) {
	alerts := []string{}

	// 检查月度流量配额
	var nodeConfig models.NodeConfig
	if err := global.DB.Where("node_id = ?", nodeID).First(&nodeConfig).Error; err == nil {
		monthUpGB := float64(nodeConfig.MonthUp) / (1024 * 1024 * 1024)
		monthDownGB := float64(nodeConfig.MonthDown) / (1024 * 1024 * 1024)

		if config.Metrics.MonthUp > 0 && monthUpGB > float64(config.Metrics.MonthUp) {
			alerts = append(alerts, fmt.Sprintf("月度上传流量超限: %.1fGB > %dGB",
				monthUpGB, config.Metrics.MonthUp))
		}
		if config.Metrics.MonthDown > 0 && monthDownGB > float64(config.Metrics.MonthDown) {
			alerts = append(alerts, fmt.Sprintf("月度下载流量超限: %.1fGB > %dGB",
				monthDownGB, config.Metrics.MonthDown))
		}
	}

	// 检查网络连接数
	if config.Metrics.Connections > 0 {
		if connCount := getNetworkConnections(nodeID); connCount > config.Metrics.Connections {
			alerts = append(alerts, fmt.Sprintf("网络连接数过高: %d > %d",
				connCount, config.Metrics.Connections))
		}
	}

	if len(alerts) > 0 {
		sendAlerts(nodeID, "网络告警", alerts, config)
	}
}

// getNetworkConnections 获取节点网络连接数
func getNetworkConnections(nodeID string) int {
	if nodeID == global.SelfNodeID {
		connCount, err := getLocalNetworkConnections()
		if err == nil {
			return connCount
		}
	}
	return 0
}

// getLocalNetworkConnections 获取本地网络连接数
func getLocalNetworkConnections() (int, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "linux" {
		cmd = exec.Command("sh", "-c", "ss -tun | wc -l")
	} else if runtime.GOOS == "darwin" {
		cmd = exec.Command("sh", "-c", "netstat -an | grep ESTABLISHED | wc -l")
	} else {
		return 0, fmt.Errorf("unsupported OS")
	}

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, err
	}
	return count, nil
}

func checkProcessAndPortAlerts(nodeID string, report models.NodeReport, config *models.AlertConfig) {
	alerts := []string{}

	// 检查进程名称是否存在
	if config.Metrics.ProcessName != "" {
		if exists := checkProcessExists(nodeID, config.Metrics.ProcessName); !exists {
			alerts = append(alerts, fmt.Sprintf("关键进程不存在: %s", config.Metrics.ProcessName))
		}
	}

	// 检查端口服务是否开放
	if config.Metrics.PortService != "" {
		if port, err := strconv.Atoi(config.Metrics.PortService); err == nil {
			if !checkPortOpen(nodeID, port) {
				alerts = append(alerts, fmt.Sprintf("端口服务不可达: %d", port))
			}
		}
	}

	if len(alerts) > 0 {
		sendAlerts(nodeID, "服务告警", alerts, config)
	}
}

// checkProcessExists 检查指定进程是否存在
func checkProcessExists(nodeID string, processName string) bool {
	if nodeID == global.SelfNodeID {
		return checkLocalProcessExists(processName)
	}

	global.NodesMu.RLock()
	node, exists := global.Nodes[nodeID]
	global.NodesMu.RUnlock()

	if exists && node.Docker != nil {
		for _, container := range node.Docker.ContainerList {
			if strings.Contains(container.Name, processName) {
				return true
			}
		}
	}
	return false
}

// checkLocalProcessExists 检查本地进程是否存在
func checkLocalProcessExists(processName string) bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("pgrep -f '%s' | head -1", processName))
	} else {
		cmd = exec.Command("cmd", "/c", fmt.Sprintf("tasklist | findstr /i '%s'", processName))
	}

	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// checkPortOpen 检查端口是否开放
func checkPortOpen(nodeID string, port int) bool {
	global.NodesMu.RLock()
	node, exists := global.Nodes[nodeID]
	global.NodesMu.RUnlock()

	if !exists || node.IP == "" || node.IP == "0.0.0.0" {
		return false
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(node.IP, strconv.Itoa(port)), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func sendAlerts(nodeID string, alertType string, alerts []string, config *models.AlertConfig) {
	global.AlertMu.Lock()

	if lastAlertTimes[alertType] == nil {
		lastAlertTimes[alertType] = make(map[string]time.Time)
	}
	if alertCache[alertType] == nil {
		alertCache[alertType] = make(map[string]bool)
	}

	lastTime, exists := lastAlertTimes[alertType][nodeID]
	alertKey := fmt.Sprintf("%s_%s_%s", nodeID, alertType, strings.Join(alerts, "|"))
	isCached := alertCache[alertType][alertKey]

	shouldSend := true

	if exists {
		timeSinceLast := time.Since(lastTime)
		if timeSinceLast < time.Duration(config.Engine.RateLimit)*time.Minute {
			shouldSend = false
		}
	}

	if isCached {
		shouldSend = false
	}

	if shouldSend {
		lastAlertTimes[alertType][nodeID] = time.Now()
		alertCache[alertType][alertKey] = true

		time.AfterFunc(24*time.Hour, func() {
			global.AlertMu.Lock()
			delete(alertCache[alertType], alertKey)
			global.AlertMu.Unlock()
		})

		global.AlertMu.Unlock()

		message := buildAlertMessage(nodeID, alertType, alerts, config)

		if config.Channels.Telegram.Enabled &&
			config.Channels.Telegram.BotToken != "" &&
			config.Channels.Telegram.ChatID != "" {
			go sendTelegramAlert(config.Channels.Telegram.BotToken, config.Channels.Telegram.ChatID, message)
		}

		logMessage := fmt.Sprintf("节点 %s 触发 %s: %s", nodeID, alertType, strings.Join(alerts, "; "))
		global.AddSystemLog(models.LogTypeAlert, "warn", logMessage)

		log.Printf("[Alert] %s", logMessage)
	} else {
		global.AlertMu.Unlock()
	}
}

func buildAlertMessage(nodeID string, alertType string, alerts []string, config *models.AlertConfig) string {
	var sb strings.Builder

	nodeName := nodeID
	global.NodesMu.RLock()
	if node, exists := global.Nodes[nodeID]; exists {
		nodeName = node.Name
	}
	global.NodesMu.RUnlock()

	var siteConfig models.SiteConfig
	global.DB.First(&siteConfig, 1)

	siteName := siteConfig.SiteName
	if siteName == "" {
		siteName = "SLITE 监控系统"
	}

	sb.WriteString(fmt.Sprintf("[%s] %s - %s\n", siteName, alertType, nodeName))
	sb.WriteString("---------------------------\n")

	for _, alert := range alerts {
		sb.WriteString(fmt.Sprintf("• %s\n", alert))
	}

	sb.WriteString("---------------------------\n")
	sb.WriteString(fmt.Sprintf("节点ID: %s\n", nodeID))
	sb.WriteString(fmt.Sprintf("触发时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("建议操作: 请登录服务器查看详细状态")

	return sb.String()
}

func sendTelegramAlert(botToken, chatID, message string) error {
	if botToken == "" || chatID == "" {
		return fmt.Errorf("Telegram配置不完整")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSON编码失败: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var tgErr struct {
			Description string `json:"description"`
		}
		json.Unmarshal(body, &tgErr)

		if tgErr.Description != "" {
			return fmt.Errorf("Telegram API错误: %s", tgErr.Description)
		} else {
			return fmt.Errorf("Telegram API返回错误: %s", string(body))
		}
	}

	return nil
}

// SendTelegramAlert 发送Telegram告警（对外接口）
func SendTelegramAlert(botToken, chatID, message string) error {
	return sendTelegramAlert(botToken, chatID, message)
}

func cleanupAlertCache() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		global.AlertMu.Lock()

		for alertType, nodeTimes := range lastAlertTimes {
			for nodeID, lastTime := range nodeTimes {
				if time.Since(lastTime) > 24*time.Hour {
					delete(lastAlertTimes[alertType], nodeID)
				}
			}
		}

		global.AlertMu.Unlock()
	}
}

// InitAlertModule 初始化告警模块
func InitAlertModule() {
	go cleanupAlertCache()
	log.Println("[Alert] 告警模块初始化完成")
}
