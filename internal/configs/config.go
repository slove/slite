package configs

import (
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 系统配置结构
type Config struct {
	Server struct {
		Port         string `yaml:"port"`
		Mode         string `yaml:"mode"`
		ReadTimeout  int    `yaml:"read_timeout"`
		WriteTimeout int    `yaml:"write_timeout"`
	} `yaml:"server"`

	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`

	Log struct {
		Level  string `yaml:"level"`
		Output string `yaml:"output"`
		File   string `yaml:"file"`
	} `yaml:"log"`

	Agent struct {
		ReportInterval int `yaml:"report_interval"`
		HeartbeatPort  int `yaml:"heartbeat_port"`
	} `yaml:"agent"`

	Alert struct {
		CheckInterval int `yaml:"check_interval"`
	} `yaml:"alert"`

	Security struct {
		AdminAuthKey    string `yaml:"admin_auth_key"`
		ViewAuthKey     string `yaml:"view_auth_key"`
		ViewAuthEnabled bool   `yaml:"view_auth_enabled"`
	} `yaml:"security"`
}

var (
	GlobalConfig *Config
)

// LoadConfig 加载配置文件
func LoadConfig(configPath string) *Config {
	cfg := &Config{}
	cfg.setDefaults()

	if configPath != "" && fileExists(configPath) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			log.Printf("[Config] 读取配置文件失败: %v，使用默认配置", err)
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				log.Printf("[Config] 解析配置文件失败: %v，使用默认配置", err)
			} else {
				log.Printf("[Config] 已加载配置文件: %s", configPath)
			}
		}
	}

	cfg.applyEnvOverrides()
	GlobalConfig = cfg
	return cfg
}

// setDefaults 设置默认值
func (c *Config) setDefaults() {
	c.Server.Port = getEnv("SERVER_PORT", "8090")
	c.Server.Mode = getEnv("GIN_MODE", "release")
	c.Server.ReadTimeout = 30
	c.Server.WriteTimeout = 30

	c.Database.Path = getEnv("DB_PATH", "slite.db")

	c.Log.Level = getEnv("LOG_LEVEL", "info")
	c.Log.Output = getEnv("LOG_OUTPUT", "stdout")
	c.Log.File = getEnv("LOG_FILE", "slite.log")

	c.Agent.ReportInterval = 10
	c.Agent.HeartbeatPort = 8091

	c.Alert.CheckInterval = 30

	// 安全配置：允许为空
	c.Security.AdminAuthKey = getEnv("ADMIN_AUTH_KEY", "admin888")
	c.Security.ViewAuthKey = getEnv("VIEW_AUTH_KEY", "") // 默认为空
	c.Security.ViewAuthEnabled = false                   // 默认关闭查看认证
}

// applyEnvOverrides 应用环境变量覆盖
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("SERVER_PORT"); v != "" {
		c.Server.Port = v
	}
	if v := os.Getenv("GIN_MODE"); v != "" {
		c.Server.Mode = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		c.Database.Path = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("LOG_OUTPUT"); v != "" {
		c.Log.Output = v
	}
	if v := os.Getenv("LOG_FILE"); v != "" {
		c.Log.File = v
	}
	if v := os.Getenv("ADMIN_AUTH_KEY"); v != "" {
		c.Security.AdminAuthKey = v
	}
	if v := os.Getenv("VIEW_AUTH_KEY"); v != "" {
		c.Security.ViewAuthKey = v
	}
	if v := os.Getenv("VIEW_AUTH_ENABLED"); v != "" {
		c.Security.ViewAuthEnabled = strings.ToLower(v) == "true" || v == "1"
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetString 根据环境变量或配置获取字符串（辅助函数）
func GetString(key string, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// GetInt 根据环境变量或配置获取整数
func GetInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}
