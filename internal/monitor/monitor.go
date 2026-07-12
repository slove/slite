package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/go-ping/ping"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"gorm.io/gorm"

	"slite/internal/alert"
	"slite/internal/global"
	"slite/internal/models"
	"slite/internal/utils"
)

/*
增强版地理位置获取函数
添加多个备用服务提高成功率
*/
func fetchGeoLocation() string {
	services := []string{
		"http://ip-api.com/json/?fields=status,country&lang=zh-CN",
		"https://ipinfo.io/json",
		"https://api.country.is/",
	}

	client := http.Client{Timeout: 3 * time.Second}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			log.Printf("请求 %s 失败: %v", service, err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("读取响应失败: %v", err)
			continue
		}

		if strings.Contains(service, "ip-api.com") {
			var result struct {
				Status  string `json:"status"`
				Country string `json:"country"`
			}
			if err := json.Unmarshal(body, &result); err == nil && result.Status == "success" {
				log.Printf("通过 ip-api.com 获取位置: %s", result.Country)
				return result.Country
			}
		} else if strings.Contains(service, "ipinfo.io") {
			var result struct {
				Country string `json:"country"`
			}
			if err := json.Unmarshal(body, &result); err == nil && result.Country != "" {
				log.Printf("通过 ipinfo.io 获取位置: %s", result.Country)
				return result.Country
			}
		} else if strings.Contains(service, "country.is") {
			var result struct {
				Country string `json:"country"`
			}
			if err := json.Unmarshal(body, &result); err == nil && result.Country != "" {
				log.Printf("通过 country.is 获取位置: %s", result.Country)
				return result.Country
			}
		}
	}

	log.Printf("所有服务均失败，返回默认位置")
	return "未知位置"
}

/*
通过系统路由表获取默认的出口网络接口名称
*/
func getDefaultInterface() string {
	cmd := exec.Command("sh", "-c", "ip route show default | head -1 | awk '{print $5}'")
	output, err := cmd.Output()
	if err == nil {
		iface := strings.TrimSpace(string(output))
		if iface != "" {
			return iface
		}
	}
	return ""
}

/*
过滤掉虚拟网卡、回环接口以及 Docker 产生的桥接网卡
*/
func shouldSkipInterface(name string) bool {
	if name == "lo" || strings.HasPrefix(name, "lo:") {
		return true
	}
	if strings.HasPrefix(name, "docker") ||
		strings.HasPrefix(name, "br-") ||
		strings.HasPrefix(name, "veth") {
		return true
	}
	if strings.HasPrefix(name, "virbr") ||
		strings.HasPrefix(name, "vnet") ||
		strings.HasPrefix(name, "tap") {
		return true
	}
	if strings.HasPrefix(name, "wg") ||
		strings.HasPrefix(name, "tun") ||
		strings.HasPrefix(name, "utun") {
		return true
	}
	return false
}

/*
验证指定的网卡名称在当前系统中是否存在
*/
func interfaceExists(name string) bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range interfaces {
		if iface.Name == name {
			return true
		}
	}
	return false
}

/*
综合判断并获取系统中最主要的核心网络接口
*/
func getPrimaryInterface() string {
	if iface := getDefaultInterface(); iface != "" && interfaceExists(iface) {
		return iface
	}

	commonInterfaces := []string{
		"eth0", "eth1", "eth2", "eth3",
		"enp0s3", "enp0s6", "enp1s0", "enp2s0",
		"ens3", "ens4", "ens5", "ens6",
		"eno1", "eno2",
		"wlan0", "wlan1",
	}

	for _, iface := range commonInterfaces {
		if interfaceExists(iface) {
			return iface
		}
	}

	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			name := iface.Name
			if !shouldSkipInterface(name) && iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
				return name
			}
		}
	}

	return "eth0"
}

/*
尝试从多个公共服务获取当前服务器的公网 IP
*/
func getPublicIP() string {
	client := http.Client{Timeout: 5 * time.Second}

	services := []string{
		"https://api.ipify.org",
		"https://checkip.amazonaws.com",
		"http://ipinfo.io/ip",
		"https://icanhazip.com",
	}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		if net.ParseIP(ip) != nil && ip != "" {
			return ip
		}
	}

	return ""
}

/*
获取特定网卡接口绑定的第一个 IPv4 地址
*/
func getInterfaceIP(ifaceName string) string {
	if ifaceName == "" {
		return ""
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return ""
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() {
			return ipNet.IP.String()
		}
	}

	return ""
}

/*
自动更新并保存当前节点的 IP 地址
*/
func updateIPAddress() {
	publicIP := getPublicIP()
	if publicIP != "" {
		global.CurrentPublicIP = publicIP
		log.Printf("公网IP地址: %s", global.CurrentPublicIP)
		return
	}

	ifaceIP := getInterfaceIP(global.PrimaryInterface)
	if ifaceIP != "" {
		global.CurrentPublicIP = ifaceIP
		log.Printf("使用网卡 %s 的IP地址: %s", global.PrimaryInterface, global.CurrentPublicIP)
		return
	}

	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() {
				global.CurrentPublicIP = ipNet.IP.String()
				log.Printf("使用本地IP地址: %s", global.CurrentPublicIP)
				return
			}
		}
	}

	global.CurrentPublicIP = ""
	log.Printf("无法获取IP地址")
}

/*
启动一个周期性的任务来维持 IP 地址的准确性
*/
func startIPUpdateScheduler() {
	updateIPAddress()

	global.IPUpdateTicker = time.NewTicker(1 * time.Hour)
	go func() {
		for range global.IPUpdateTicker.C {
			updateIPAddress()
		}
	}()
}

/*
启动每日在线统计的同步调度器
*/
func startDailyUptimeScheduler() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		log.Println("正在同步节点统计数据...")
	}
}

/*
管理网络探测任务的生命周期与结果存储
*/
func startNetworkDetectionScheduler() {
	time.Sleep(1 * time.Second)
	exec := func() {
		var activeTasks []models.Task
		global.DB.Where("is_active = ?", true).Find(&activeTasks)

		if len(activeTasks) == 0 {
			var count int64
			global.DB.Model(&models.Task{}).Count(&count)
			if count == 0 {
				defaults := []struct{ Name, Type, Target string }{
					{"Google DNS (ICMP)", "icmp", "8.8.8.8"},
					{"Google TCP Ping", "tcp", "www.google.com"},
					{"Google HTTPS", "http", "https://www.google.com"},
				}
				for _, d := range defaults {
					var tid uint64
					if global.GlobalSnowflake != nil {
						tid = uint64(global.GlobalSnowflake.Generate().Int64())
					} else {
						tid = uint64(time.Now().UnixNano() / 1e6)
					}

					global.DB.Create(&models.Task{
						ID:        tid,
						Name:      d.Name,
						Type:      d.Type,
						Target:    d.Target,
						Port:      80,
						IsActive:  true,
						CreatedAt: time.Now(),
					})
				}
				global.DB.Where("is_active = ?", true).Find(&activeTasks)
			}
		}

		var results []models.DetectTask
		for _, t := range activeTasks {
			delay, loss := DoDetection(t)

			go func(taskID uint64, d float64) {
				global.DB.Create(&models.TaskHistory{
					TaskID:    taskID,
					Delay:     d,
					CreatedAt: time.Now(),
				})
			}(t.ID, delay)

			global.HistoryMu.Lock()
			global.TaskHistory[t.ID] = append(global.TaskHistory[t.ID], delay)
			if len(global.TaskHistory[t.ID]) > 100 {
				global.TaskHistory[t.ID] = global.TaskHistory[t.ID][1:]
			}
			hist := make([]float64, len(global.TaskHistory[t.ID]))
			copy(hist, global.TaskHistory[t.ID])
			global.HistoryMu.Unlock()

			results = append(results, models.DetectTask{
				ID:        t.ID,
				Name:      t.Name,
				Type:      t.Type,
				LastDelay: delay,
				LossRate:  loss,
				History:   hist,
			})
		}

		global.NodesMu.Lock()
		if n, ok := global.Nodes[global.SelfNodeID]; ok {
			n.Tasks = results
		}
		global.NodesMu.Unlock()
	}

	exec()
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		exec()
	}
}

// DoDetection 根据任务类型执行具体的网络连通性或延迟探测（导出供其他包使用）
func DoDetection(t models.Task) (float64, float64) {
	if t.Type == "icmp" {
		p, err := ping.NewPinger(t.Target)
		if err != nil {
			return 0, 100
		}
		p.Count = 1
		p.Timeout = time.Second * 2
		p.SetPrivileged(true)
		if err := p.Run(); err != nil {
			return 0, 100
		}
		return float64(p.Statistics().AvgRtt.Milliseconds()), p.Statistics().PacketLoss
	} else if t.Type == "tcp" {
		start := time.Now()
		port := t.Port
		if port == 0 {
			port = 443
		}
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(t.Target, fmt.Sprintf("%d", port)), time.Second*2)
		if err != nil {
			return 0, 100
		}
		conn.Close()
		return float64(time.Since(start).Milliseconds()), 0
	} else if t.Type == "http" {
		client := http.Client{Timeout: 3 * time.Second}
		start := time.Now()
		resp, err := client.Get(t.Target)
		if err != nil {
			return 0, 100
		}
		resp.Body.Close()
		return float64(time.Since(start).Milliseconds()), 0
	}
	return 0, 0
}

/*
访问 Docker 守护进程获取容器状态及资源消耗详情
*/
func getDockerInfo() *models.DockerStatus {
	ds := &models.DockerStatus{ContainerList: []models.ContainerDetail{}, ImageSize: "0 B"}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return ds
	}
	defer cli.Close()

	ctx := context.Background()

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err == nil {
		var totalImgSize int64
		for _, img := range images {
			totalImgSize += img.Size
		}
		ds.ImageSize = utils.FormatBytes(totalImgSize)
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true, Size: true})
	if err != nil {
		return ds
	}
	ds.TotalCount = len(containers)

	var totalNetIn, totalNetOut uint64
	for _, c := range containers {
		detail := models.ContainerDetail{
			Name:       c.Names[0][1:],
			Status:     c.Status,
			DiskUsage:  utils.FormatBytes(c.SizeRw),
			RootFsSize: utils.FormatBytes(c.SizeRootFs),
			CPUUsage:   "0.0",
		}

		if c.State == "running" {
			ds.RunningCount++
			ds.HealthyCount++

			stats, err := cli.ContainerStatsOneShot(ctx, c.ID)
			if err == nil {
				var v struct {
					CPUStats struct {
						CPUUsage struct {
							TotalUsage uint64 `json:"total_usage"`
						} `json:"cpu_usage"`
						SystemCPUUsage uint64 `json:"system_cpu_usage"`
						OnlineCPUs     uint32 `json:"online_cpus"`
					} `json:"cpu_stats"`
					PreCPUStats struct {
						CPUUsage struct {
							TotalUsage uint64 `json:"total_usage"`
						} `json:"cpu_usage"`
						SystemCPUUsage uint64 `json:"system_cpu_usage"`
					} `json:"precpu_stats"`
					MemoryStats struct {
						Usage int64 `json:"usage"`
					} `json:"memory_stats"`
					Networks map[string]struct {
						RxBytes uint64 `json:"rx_bytes"`
						TxBytes uint64 `json:"tx_bytes"`
					} `json:"networks"`
				}
				if err := json.NewDecoder(stats.Body).Decode(&v); err == nil {
					cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
					systemDelta := float64(v.CPUStats.SystemCPUUsage) - float64(v.PreCPUStats.SystemCPUUsage)
					onlineCPUs := float64(v.CPUStats.OnlineCPUs)
					if onlineCPUs == 0 {
						onlineCPUs = 1
					}
					if systemDelta > 0 && cpuDelta > 0 {
						cpuPercent := (cpuDelta / systemDelta) * onlineCPUs * 100.0
						detail.CPUUsage = fmt.Sprintf("%.1f", cpuPercent)
					}

					detail.MemoryUsage = utils.FormatBytes(v.MemoryStats.Usage)
					var netIn, netOut uint64
					for _, network := range v.Networks {
						netIn += network.RxBytes
						netOut += network.TxBytes
					}
					detail.NetIn = netIn
					detail.NetOut = netOut
					totalNetIn += netIn
					totalNetOut += netOut
				}
				stats.Body.Close()
			}
		} else {
			detail.MemoryUsage = "0 B"
			detail.CPUUsage = "0.0"
		}
		ds.ContainerList = append(ds.ContainerList, detail)
	}
	ds.NetIn = totalNetIn
	ds.NetOut = totalNetOut
	return ds
}

/*
获取默认的本地网络探测任务列表（不带nodeID参数）
当数据库中没有配置任务时使用
*/
func getLocalDefaultNetworkTasks() []models.Task {
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

	var tasks []models.Task
	for i, d := range defaultTasks {
		var tid uint64
		if global.GlobalSnowflake != nil {
			tid = uint64(global.GlobalSnowflake.Generate().Int64())
		} else {
			tid = uint64(time.Now().UnixNano()/1e6) + uint64(i)
		}

		tasks = append(tasks, models.Task{
			ID:        tid,
			Name:      d.Name,
			Type:      d.Type,
			Target:    d.Target,
			Port:      d.Port,
			SortIndex: i,
			IsActive:  true,
			CreatedAt: time.Now(),
		})
	}

	return tasks
}

/*
获取当前节点的网络探测任务列表
优先从数据库获取活动任务，如果数据库中没有任务则返回默认任务列表
*/
func getNetworkDetectionTasks() []models.Task {
	var activeTasks []models.Task
	global.DB.Where("is_active = ?", true).Order("sort_index ASC").Find(&activeTasks)

	if len(activeTasks) == 0 {
		log.Println("数据库中无活动任务，使用默认网络检测任务")
		return getLocalDefaultNetworkTasks()
	}

	return activeTasks
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

// isQuietTime 检查是否在静默时段（内部使用）
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

/*
启动告警检查调度器
*/
func startAlertCheckScheduler() {
	log.Println("启动告警检查调度器...")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var config models.AlertConfig
		if err := global.DB.First(&config, 1).Error; err != nil {
			continue
		}

		if !config.Engine.Enabled {
			continue
		}

		if isQuietTime(config.Engine.QuietStart, config.Engine.QuietEnd) {
			continue
		}

		global.NodesMu.RLock()
		nodesCopy := make(map[string]*models.Node)
		for k, v := range global.Nodes {
			nodesCopy[k] = v
		}
		global.NodesMu.RUnlock()

		for _, nodeID := range config.Engine.TargetNodes {
			if node, exists := nodesCopy[nodeID]; exists {
				report := models.NodeReport{
					ID:           node.ID,
					Name:         node.Name,
					Location:     node.Location,
					IP:           node.IP,
					OS:           node.OS,
					CPU:          node.CPU,
					Mem:          node.Mem,
					Disk:         node.Disk,
					Load:         node.Load,
					Uptime:       node.Uptime,
					Up:           node.Up,
					Down:         node.Down,
					NetIn:        node.NetIn,
					NetOut:       node.NetOut,
					ProcessCount: node.ProcessCount,
					Docker:       node.Docker,
				}

				alert.ExecuteAlertCheck(nodeID, report)
			}
		}
	}
}

/*
初始化控制台自身的监控引擎
*/
func StartSelfMonitoring() {
	log.Println("正在启动监控服务器引擎...")

	const MasterNodeFixedID = "10000"
	currentEnvLocation := fetchGeoLocation()

	var conf models.NodeConfig
	result := global.DB.Where("node_id = ?", MasterNodeFixedID).Limit(1).Find(&conf)

	if result.RowsAffected == 0 {
		global.SelfNodeID = MasterNodeFixedID
		global.SelfLocation = currentEnvLocation
		conf = models.NodeConfig{
			ID:             global.SelfNodeID,
			CustomName:     "监控服务器",
			Location:       global.SelfLocation,
			IsVisible:      true,
			UptimeRate:     100.0,
			OfflineTotal:   "0s",
			TotalStatsDays: 1,
		}
		global.DB.Create(&conf)
		log.Printf("首次运行，已创建固定控制台节点 ID: %s", global.SelfNodeID)
	} else {
		global.SelfNodeID = conf.ID
		global.SelfLocation = conf.Location
		log.Printf("已加载控制台配置")
	}

	global.PrimaryInterface = getPrimaryInterface()
	log.Printf("确定主网卡: %s", global.PrimaryInterface)

	startIPUpdateScheduler()

	global.NodesMu.Lock()
	global.Nodes[global.SelfNodeID] = &models.Node{
		ID:             global.SelfNodeID,
		Name:           conf.CustomName,
		Location:       global.SelfLocation,
		IsVisible:      conf.IsVisible,
		Tasks:          make([]models.DetectTask, 0),
		UptimeHeatMap:  make([]models.HeatMapItem, 0),
		Load:           []float64{0, 0, 0},
		Online:         true,
		TotalStatsDays: conf.TotalStatsDays,
	}
	global.NodesMu.Unlock()

	go startNetworkDetectionScheduler()
	go startDailyUptimeScheduler()

	allNet, err := psnet.IOCounters(true)
	if err == nil {
		for _, net := range allNet {
			if net.Name == global.PrimaryInterface {
				global.LastTotalSent = net.BytesSent
				global.LastTotalRecv = net.BytesRecv
				break
			}
		}
	}

	lastTime := time.Now()
	fastTicker := time.NewTicker(5 * time.Second)

	go startAlertCheckScheduler()

	for range fastTicker.C {
		vmem, _ := mem.VirtualMemory()
		cpuP, _ := cpu.Percent(0, false)
		dstat, _ := disk.Usage("/")
		hInfo, _ := host.Info()
		avgL, _ := load.Avg()
		allNet, _ := psnet.IOCounters(true)
		pids, _ := process.Pids()
		curTime := time.Now()

		var upS, downS float64
		var currentNetIn, currentNetOut uint64

		for _, net := range allNet {
			if net.Name == global.PrimaryInterface {
				currentNetIn = net.BytesRecv
				currentNetOut = net.BytesSent
				break
			}
		}

		dur := curTime.Sub(lastTime).Seconds()
		diffUp := currentNetOut - global.LastTotalSent
		diffDown := currentNetIn - global.LastTotalRecv

		if dur > 0 {
			upS = float64(diffUp) / 1024 / dur
			downS = float64(diffDown) / 1024 / dur
		}

		if diffUp > 0 || diffDown > 0 {
			global.DB.Model(&models.NodeConfig{}).Where("node_id = ?", global.SelfNodeID).Updates(map[string]interface{}{
				"month_up":   gorm.Expr("month_up + ?", diffUp),
				"month_down": gorm.Expr("month_down + ?", diffDown),
			})
		}

		global.LastTotalSent = currentNetOut
		global.LastTotalRecv = currentNetIn

		dockerInfo := getDockerInfo()

		global.NodesMu.Lock()
		if n, ok := global.Nodes[global.SelfNodeID]; ok {
			var dbConf models.NodeConfig
			if err := global.DB.Where("node_id = ?", global.SelfNodeID).Limit(1).Find(&dbConf).Error; err == nil {
				n.Name = dbConf.CustomName
				n.Location = dbConf.Location
				n.IsVisible = dbConf.IsVisible
				n.MonthUp = dbConf.MonthUp
				n.MonthDown = dbConf.MonthDown
				n.TotalStatsDays = dbConf.TotalStatsDays
			}

			if global.CurrentPublicIP != "" {
				n.IP = global.CurrentPublicIP
			} else {
				n.IP = getInterfaceIP(global.PrimaryInterface)
				if n.IP == "" {
					n.IP = "0.0.0.0"
				}
			}

			n.OS = fmt.Sprintf("%s %s", hInfo.Platform, hInfo.PlatformVersion)
			if len(cpuP) > 0 {
				n.CPU = cpuP[0]
			}
			n.Mem = vmem.UsedPercent
			n.Disk = dstat.UsedPercent
			n.ProcessCount = len(pids)
			n.Load = []float64{avgL.Load1, avgL.Load5, avgL.Load15}
			n.Uptime = hInfo.Uptime
			n.Up = upS
			n.Down = downS
			n.NetIn = global.LastTotalRecv
			n.NetOut = global.LastTotalSent
			n.Online = true
			n.LastSeen = curTime
			n.Docker = dockerInfo

			updateTodayUptimeRecord(global.SelfNodeID, true)
		}
		global.NodesMu.Unlock()
		lastTime = curTime
	}
}

/*
获取网络任务信息
用于获取当前节点的网络探测任务状态
*/
func GetNetworkTasks() []models.DetectTask {
	global.NodesMu.RLock()
	defer global.NodesMu.RUnlock()

	if node, ok := global.Nodes[global.SelfNodeID]; ok {
		return node.Tasks
	}

	return []models.DetectTask{}
}
