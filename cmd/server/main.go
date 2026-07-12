package main

import (
	"log"
	"os"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"

	"slite/internal/alert"
	"slite/internal/configs"
	"slite/internal/controllers"
	"slite/internal/database"
	"slite/internal/global"
	"slite/internal/monitor"
	"slite/internal/routes"
)

func main() {
	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./configs/config.yaml"
	}
	cfg := configs.LoadConfig(configPath)

	// 初始化雪花ID生成器
	var err error
	global.GlobalSnowflake, err = snowflake.NewNode(1)
	if err != nil {
		log.Fatalf("[Critical] 初始化雪花算法失败: %v", err)
	}
	log.Println("[System] 雪花算法 ID 生成器初始化成功")

	// 初始化数据库
	database.InitDB()
	database.CheckDatabaseData()

	// 初始化告警模块
	alert.InitAlertModule()

	// 启动各种后台任务
	go monitor.StartSelfMonitoring()
	go controllers.CheckOffline()
	go controllers.StartHistoryWorker()
	go controllers.StartWSBroadcast()
	go controllers.StartGlobalNetworkDetection()

	// 配置Gin
	if cfg.Server.Mode != "" {
		gin.SetMode(cfg.Server.Mode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// 注册路由
	routes.RegisterHandlers(r)

	// 获取端口
	port := cfg.Server.Port
	if port == "" {
		port = "8090"
	}

	// 启动日志
	log.Println("====================================================")
	log.Println("[System] Slite Monitor Master Server v0.1.0")
	log.Println("[Alert] 告警模块已启用")
	log.Println("[Route] 看板入口: /monitor")
	log.Println("[Route] 管理后台: /admin")
	log.Println("[Route] WS接口: /ws")
	log.Printf("[System] 服务器监听端口: %s", port)
	log.Printf("[System] 运行模式: %s", cfg.Server.Mode)
	log.Println("====================================================")

	// 启动HTTP服务器
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[Critical] 服务器启动失败: %v", err)
	}
}
