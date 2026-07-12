package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
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
)

/*
全局变量定义
用于存储网络统计、当前公网IP及版本号
*/
var (
	primaryInterface  string
	lastTotalSent     uint64
	lastTotalRecv     uint64
	currentPublicIP   string
	Version           = "1.4.0"
	locationCache     string
	locationCacheTime time.Time
	locationCacheLock sync.RWMutex
)

/*
Agent配置结构体
定义Agent启动及上报所需的基础配置
*/
type AgentConfig struct {
	ServerURL      string
	NodeID         string
	NodeKey        string
	CustomName     string
	ReportInterval int
}

/*
网络探测任务结果结构体
网络探测任务的执行结果
*/
type DetectTaskResult struct {
	ID        uint64  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	LastDelay float64 `json:"last_delay"`
	LossRate  float64 `json:"loss_rate"`
}

/*
节点上报数据结构体
上报给服务端的完整数据包结构
*/
type NodeReport struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Location     string             `json:"location"`
	IP           string             `json:"ip"`
	OS           string             `json:"os"`
	CPU          float64            `json:"cpu"`
	Mem          float64            `json:"mem"`
	Disk         float64            `json:"disk"`
	ProcessCount int                `json:"process_count"`
	Load         []float64          `json:"load"`
	Uptime       uint64             `json:"uptime"`
	Up           float64            `json:"up"`
	Down         float64            `json:"down"`
	NetIn        uint64             `json:"net_in"`
	NetOut       uint64             `json:"net_out"`
	Docker       *DockerStatus      `json:"docker"`
	Tasks        []DetectTaskResult `json:"tasks"`
	NodeKey      string             `json:"node_key"`
	Version      string             `json:"version"`
}

/*
Docker状态结构体
汇总节点Docker容器的运行状态
*/
type DockerStatus struct {
	RunningCount  int               `json:"running_count"`
	TotalCount    int               `json:"total_count"`
	HealthyCount  int               `json:"healthy_count"`
	ImageSize     string            `json:"image_size"`
	NetIn         uint64            `json:"net_in"`
	NetOut        uint64            `json:"net_out"`
	ContainerList []ContainerDetail `json:"container_list"`
}

/*
容器详情结构体
单个容器的详细运行指标
*/
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

/*
网络任务结构体
用于存储从服务器获取的网络探测任务配置
*/
type NetworkTask struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Target   string `json:"target"`
	Port     int    `json:"port"`
	IsActive bool   `json:"is_active"`
}

/*
主函数入口
初始化配置、网卡及IP更新任务，并开启数据上报循环
*/
func main() {
	argID := flag.String("id", "", "节点ID")
	argName := flag.String("name", "", "节点名称")
	argKey := flag.String("key", "", "节点密钥")
	argURL := flag.String("url", "", "服务器地址")
	flag.Parse()

	if *argID == "" || *argURL == "" {
		log.Fatal("错误: 必须指定 --id 和 --url 参数")
	}

	config := &AgentConfig{
		ServerURL:      strings.TrimSuffix(*argURL, "/"),
		NodeID:         *argID,
		NodeKey:        *argKey,
		CustomName:     *argName,
		ReportInterval: 10,
	}

	if config.CustomName == "" {
		config.CustomName, _ = os.Hostname()
	}

	log.Println("====================================================")
	log.Printf("SLITE Agent v%s 启动成功", Version)
	log.Printf("节点名称: %s", config.CustomName)
	log.Printf("上报地址: %s", config.ServerURL)
	log.Println("====================================================")

	primaryInterface = getPrimaryInterface()
	initNetworkStats()

	go startIPUpdateScheduler()

	startReporting(config)
}

/*
执行网络探测任务
根据从服务器获取的任务配置执行ICMP、TCP及HTTP探测
*/
func doDetection(config *AgentConfig) []DetectTaskResult {
	tasks := getNetworkTasks(config)
	if len(tasks) == 0 {
		return doDefaultDetection()
	}

	var results []DetectTaskResult
	for _, task := range tasks {
		if !task.IsActive {
			continue
		}

		var delay, loss float64
		if task.Type == "icmp" {
			p, err := ping.NewPinger(task.Target)
			if err == nil {
				p.Count = 2
				p.Timeout = time.Second * 2
				p.SetPrivileged(runtime.GOOS == "linux")
				if err := p.Run(); err == nil {
					stats := p.Statistics()
					delay = float64(stats.AvgRtt.Milliseconds())
					loss = stats.PacketLoss
				} else {
					loss = 100
				}
			}
		} else if task.Type == "tcp" {
			start := time.Now()
			port := task.Port
			if port == 0 {
				port = 443
			}
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(task.Target, fmt.Sprintf("%d", port)), time.Second*2)
			if err == nil {
				conn.Close()
				delay = float64(time.Since(start).Milliseconds())
			} else {
				loss = 100
			}
		} else if task.Type == "http" {
			client := http.Client{Timeout: 3 * time.Second}
			start := time.Now()
			resp, err := client.Get(task.Target)
			if err == nil {
				resp.Body.Close()
				delay = float64(time.Since(start).Milliseconds())
			} else {
				loss = 100
			}
		}

		results = append(results, DetectTaskResult{
			ID:        task.ID,
			Name:      task.Name,
			Type:      task.Type,
			LastDelay: delay,
			LossRate:  loss,
		})
	}
	return results
}

/*
执行默认网络探测任务
当没有从服务器获取到任务时使用默认任务
*/
func doDefaultDetection() []DetectTaskResult {
	targets := []struct {
		ID     uint64
		Name   string
		Type   string
		Target string
		Port   int
	}{
		{1, "Google DNS (ICMP)", "icmp", "8.8.8.8", 0},
		{2, "Google TCP Ping", "tcp", "www.google.com", 443},
		{3, "Google HTTPS", "http", "https://www.google.com", 0},
	}

	var results []DetectTaskResult
	for _, t := range targets {
		var delay, loss float64
		if t.Type == "icmp" {
			p, err := ping.NewPinger(t.Target)
			if err == nil {
				p.Count = 2
				p.Timeout = time.Second * 2
				p.SetPrivileged(runtime.GOOS == "linux")
				if err := p.Run(); err == nil {
					stats := p.Statistics()
					delay = float64(stats.AvgRtt.Milliseconds())
					loss = stats.PacketLoss
				} else {
					loss = 100
				}
			}
		} else if t.Type == "tcp" {
			start := time.Now()
			port := t.Port
			if port == 0 {
				port = 443
			}
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(t.Target, fmt.Sprintf("%d", port)), time.Second*2)
			if err == nil {
				conn.Close()
				delay = float64(time.Since(start).Milliseconds())
			} else {
				loss = 100
			}
		} else if t.Type == "http" {
			client := http.Client{Timeout: 3 * time.Second}
			start := time.Now()
			resp, err := client.Get(t.Target)
			if err == nil {
				resp.Body.Close()
				delay = float64(time.Since(start).Milliseconds())
			} else {
				loss = 100
			}
		}

		results = append(results, DetectTaskResult{
			ID:        t.ID,
			Name:      t.Name,
			Type:      t.Type,
			LastDelay: delay,
			LossRate:  loss,
		})
	}
	return results
}

/*
从服务器获取网络任务配置
向服务器请求该节点的网络探测任务配置
*/
func getNetworkTasks(config *AgentConfig) []NetworkTask {
	client := http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", config.ServerURL+"/api/agent/tasks", nil)
	if err != nil {
		return []NetworkTask{}
	}
	req.Header.Set("X-Node-ID", config.NodeID)
	req.Header.Set("X-Node-Key", config.NodeKey)

	resp, err := client.Do(req)
	if err != nil {
		return []NetworkTask{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []NetworkTask{}
	}

	var tasks []NetworkTask
	if err := json.Unmarshal(body, &tasks); err != nil {
		return []NetworkTask{}
	}

	return tasks
}

/*
获取地理位置信息
通过外部API获取当前节点的地理位置，每天只获取一次
*/
func fetchGeoLocation() string {
	locationCacheLock.RLock()
	if locationCache != "" && time.Since(locationCacheTime) < 24*time.Hour {
		locationCacheLock.RUnlock()
		return locationCache
	}
	locationCacheLock.RUnlock()

	locationCacheLock.Lock()
	defer locationCacheLock.Unlock()

	if locationCache != "" && time.Since(locationCacheTime) < 24*time.Hour {
		return locationCache
	}

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

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("%s 返回状态码: %d", service, resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
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
				locationCache = result.Country
				locationCacheTime = time.Now()
				return locationCache
			}
		} else if strings.Contains(service, "ipinfo.io") {
			var result struct {
				Country string `json:"country"`
			}
			if err := json.Unmarshal(body, &result); err == nil && result.Country != "" {
				log.Printf("通过 ipinfo.io 获取位置: %s", result.Country)
				locationCache = result.Country
				locationCacheTime = time.Now()
				return locationCache
			}
		} else if strings.Contains(service, "country.is") {
			var result struct {
				Country string `json:"country"`
			}
			if err := json.Unmarshal(body, &result); err == nil && result.Country != "" {
				log.Printf("通过 country.is 获取位置: %s", result.Country)
				locationCache = result.Country
				locationCacheTime = time.Now()
				return locationCache
			}
		}
	}

	log.Printf("所有服务均失败，返回默认位置")
	locationCache = "未知位置"
	locationCacheTime = time.Now()
	return locationCache
}

/*
获取主要网络接口
通过系统路由表获取默认的出口网卡名称
*/
func getPrimaryInterface() string {
	cmd := exec.Command("sh", "-c", "ip route show default | head -1 | awk '{print $5}'")
	output, err := cmd.Output()
	if err == nil {
		iface := strings.TrimSpace(string(output))
		if iface != "" {
			return iface
		}
	}
	return "eth0"
}

/*
初始化网络统计数据
初始化流量计数器基准值
*/
func initNetworkStats() {
	allNet, _ := psnet.IOCounters(true)
	for _, n := range allNet {
		if n.Name == primaryInterface {
			lastTotalSent = n.BytesSent
			lastTotalRecv = n.BytesRecv
			break
		}
	}
}

/*
启动IP更新调度器
定期访问公网接口更新节点的外部IP地址
*/
func startIPUpdateScheduler() {
	updateIP := func() {
		client := http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("https://api.ipify.org")
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			currentPublicIP = strings.TrimSpace(string(body))
		}
	}
	updateIP()
	for range time.NewTicker(1 * time.Hour).C {
		updateIP()
	}
}

/*
启动数据上报循环
根据预设的时间间隔驱动采集和数据上报任务
*/
func startReporting(config *AgentConfig) {
	interval := time.Duration(config.ReportInterval) * time.Second
	ticker := time.NewTicker(interval)
	lastTime := time.Now()

	for range ticker.C {
		curTime := time.Now()
		dur := curTime.Sub(lastTime).Seconds()

		report := collectData(config, dur)
		sendData(config, report)

		lastTime = curTime
	}
}

/*
收集系统数据
整合系统、网络、进程及Docker信息生成最终的上报报文
*/
func collectData(config *AgentConfig, dur float64) NodeReport {
	vmem, _ := mem.VirtualMemory()
	cpuP, _ := cpu.Percent(0, false)
	dstat, _ := disk.Usage("/")
	hInfo, _ := host.Info()
	avgL, _ := load.Avg()
	allNet, _ := psnet.IOCounters(true)
	pids, _ := process.Pids()

	var curIn, curOut uint64
	for _, n := range allNet {
		if n.Name == primaryInterface {
			curIn = n.BytesRecv
			curOut = n.BytesSent
			break
		}
	}

	var upS, downS float64
	if dur > 0 {
		upS = float64(curOut-lastTotalSent) / 1024 / dur
		downS = float64(curIn-lastTotalRecv) / 1024 / dur
	}
	lastTotalSent, lastTotalRecv = curOut, curIn

	cpuV := 0.0
	if len(cpuP) > 0 {
		cpuV = cpuP[0]
	}

	location := fetchGeoLocation()

	return NodeReport{
		ID:           config.NodeID,
		Name:         config.CustomName,
		Location:     location,
		IP:           currentPublicIP,
		OS:           fmt.Sprintf("%s %s", hInfo.Platform, hInfo.PlatformVersion),
		CPU:          cpuV,
		Mem:          vmem.UsedPercent,
		Disk:         dstat.UsedPercent,
		ProcessCount: len(pids),
		Load:         []float64{avgL.Load1, avgL.Load5, avgL.Load15},
		Uptime:       hInfo.Uptime,
		Up:           upS,
		Down:         downS,
		NetIn:        curIn,
		NetOut:       curOut,
		Docker:       getDockerInfo(),
		Tasks:        doDetection(config),
		NodeKey:      config.NodeKey,
		Version:      Version,
	}
}

/*
发送数据到服务器
通过HTTP POST将JSON报文发送至服务端API
*/
func sendData(config *AgentConfig, report NodeReport) {
	jsonData, _ := json.Marshal(report)
	resp, err := http.Post(config.ServerURL+"/api/report", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("上报失败: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("上报成功: CPU %.1f%% | 网速 ↑%.1f KB/s | 位置: %s", report.CPU, report.Up, report.Location)
}

/*
获取Docker信息
通过Docker SDK获取容器存活状态及镜像占用空间
*/
func getDockerInfo() *DockerStatus {
	ds := &DockerStatus{ContainerList: []ContainerDetail{}, ImageSize: "0 B"}
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
		ds.ImageSize = formatBytes(totalImgSize)
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true, Size: true})
	if err != nil {
		return ds
	}
	ds.TotalCount = len(containers)

	var totalNetIn, totalNetOut uint64
	for _, c := range containers {
		detail := ContainerDetail{
			Name:       c.Names[0][1:],
			Status:     c.Status,
			DiskUsage:  formatBytes(c.SizeRw),
			RootFsSize: formatBytes(c.SizeRootFs),
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

					detail.MemoryUsage = formatBytes(v.MemoryStats.Usage)

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
格式化字节数为人类可读格式
将字节数转换为人类可读的字符串格式
*/
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
