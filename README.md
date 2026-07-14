# Slite — 轻量级服务器监控系统

Slite 是一个使用 Go 语言开发的现代化服务器监控系统，提供简洁美观的 Web 界面，支持多节点管理、实时监控、告警通知、数据备份与恢复等功能。它旨在帮助运维人员轻松掌握服务器集群的运行状态。

> 项目当前处于 v0.1.0 开发阶段，欢迎提交 Issue 和 Pull Request。

前端预览

![image](https://cdn.nodeimage.com/i/ZUUUzkjnPjN8M9wJxnh1xZD9ZtdrE57W.webp)

后端预览

![image](https://cdn.nodeimage.com/i/d8nHmlYg6y6nJ0HvQNEh18fOzXJDw6H4.webp)

## 主要特性

* **多节点监控** — 集中管理多台服务器，实时查看 CPU、内存、磁盘、网络流量、系统负载等核心指标
* **网络探测** — 支持 ICMP、TCP、HTTP 三种探测方式，监控节点到目标服务的网络延迟和丢包率
* **告警引擎** — 支持自定义阈值（CPU/内存/磁盘/负载/连接数等），通过 Telegram 推送告警，支持静默时段和频率限制
* **备份与恢复** — 提供数据库的备份、验证、下载、恢复功能，支持保留策略和加密存储
* **主题切换** — 支持亮色/暗色主题，支持跟随系统偏好，管理后台与监控面板独立主题
* **Docker 支持** — 自动检测并展示 Docker 容器运行状态、资源占用及网络流量
* **轻量高效** — 基于 Go 编译为静态二进制文件，资源占用低，部署简单
* **Agent 架构** — 服务端与 Agent 分离，Agent 支持 Linux/macOS/Windows 多平台

## 项目架构

Slite 采用经典的 C/S 架构，分为三个核心层：

### 第一层：前端展示层

* **监控面板** (路径: `/monitor`) — 面向普通用户，查看节点状态、网络探测结果、热力图等
* **管理后台** (路径: `/admin`) — 面向管理员，进行系统配置、告警规则、备份恢复、节点管理等
* **通信方式**: WebSocket 实时推送 + HTTP API

### 第二层：服务端核心层 (Slite Server)

* **路由层** — 处理 HTTP 请求路由，注册所有 API 端点
* **中间件层** — 认证鉴权（管理员/查看者）、日志记录
* **Handler 层** — 处理具体的业务逻辑请求
* **告警引擎** — 定时检查节点指标，触发告警并通过 Telegram 推送
* **备份管理** — 数据库备份、验证、恢复、清理
* **节点管理** — 节点注册、配置、分组、排序
* **网络探测调度** — 周期性执行 ICMP/TCP/HTTP 探测任务
* **WebSocket 广播** — 将系统状态实时推送给所有前端客户端
* **数据存储层** — SQLite 数据库，使用 GORM 作为 ORM 框架

### 第三层：Agent 采集层 (Slite Agent)

* **系统信息采集** — CPU、内存、磁盘、网络流量、系统负载、进程数、运行时长
* **Docker 信息采集** — 容器列表、运行状态、资源占用、网络流量
* **网络探测执行** — 执行服务端下发的 ICMP/TCP/HTTP 探测任务
* **数据上报** — 通过 HTTP JSON 接口定期将采集数据上报给服务端

### 通信流程

Agent 通过 HTTP POST 上报监控数据到服务端的 `/api/report` 接口，Agent 通过 HTTP GET 从服务端获取探测任务配置，服务端通过 WebSocket 将实时状态推送给浏览器端，管理员通过浏览器访问管理后台进行系统配置。

## 快速开始

### 前置要求

* Go 1.24+ （仅编译时需要）
* SQLite （内置，无需额外安装）

### 1. 克隆仓库

```bash
git clone https://github.com/slove/slite.git
cd slite

```

### 2. 编译服务端

```bash
go build -o dist/bin/slite-server ./cmd/server

```

或使用脚本全平台编译 Agent：

```bash
./scripts/build-agent.sh

```

### 3. 启动服务端

```bash
go run ./cmd/server

```

或使用管理脚本：

```bash
./scripts/slite.sh start

```

* 默认监听端口：8090
* 监控面板：http://localhost:8090/monitor
* 管理后台：http://localhost:8090/admin
* 默认管理员密钥：`admin888` （建议首次登录后修改）

### 4. 部署 Agent 到被监控节点

Agent 支持 Linux / macOS / Windows，可通过安装脚本快速部署：

```bash
curl -sSL http://your-server:8090/install.sh | bash -s -- \
--id your-node-id \
--url http://your-server:8090 \
--name "My Server"

```

或手动下载对应平台的二进制文件：

```bash
wget http://your-server:8090/dist/bin/slite-agent-linux-amd64 -O /opt/slite-agent
chmod +x /opt/slite-agent
/opt/slite-agent -id node-001 -url http://your-server:8090

```

## 配置说明

配置文件位于 `internal/configs/config.yaml`（首次运行会自动生成默认配置）：

```yaml
server:
  port: 8090
  mode: release

database:
  path: slite.db

security:
  admin_auth_key: admin888
  view_auth_key: ""
  view_auth_enabled: false

agent:
  report_interval: 10
  heartbeat_port: 8091

alert:
  check_interval: 30

```

## 开发指南

### 构建 Agent（支持多平台）

```bash
./scripts/build-agent.sh

```

输出位置：`dist/bin/slite-agent-{os}-{arch}`

### 运行测试

```bash
go test ./...

```

### 代码格式化

```bash
go fmt ./...

```

## 贡献指南

欢迎任何形式的贡献！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 提交 Pull Request

请确保代码符合 Go 语言规范，并添加必要的注释。

## 许可证

本项目采用 Apache License 2.0，可自由使用、修改和分发。详见 LICENSE 文件。

## 致谢

项目使用了以下优秀的开源库：

* Gin — Web 框架
* GORM — ORM 库
* gopsutil — 系统信息采集
* go-ping — ICMP 探测
* Docker SDK — Docker 容器信息采集
* snowflake — 分布式 ID 生成
