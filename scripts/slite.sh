#!/bin/bash
# ==============================================================================
# Slite 服务管理脚本（交互式菜单）
# 位置: scripts/slite.sh
# 用法: ./scripts/slite.sh          # 进入交互式菜单
#       ./scripts/slite.sh start    # 直接启动
#       ./scripts/slite.sh stop     # 直接停止
#       ./scripts/slite.sh restart  # 直接重启
#       ./scripts/slite.sh logs     # 查看日志
#       ./scripts/slite.sh status   # 查看状态
# ==============================================================================

set -euo pipefail

# ==============================================================================
# 颜色定义
# ==============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ==============================================================================
# 配置
# ==============================================================================

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# PID 文件和日志文件（放在项目根目录）
PIDFILE="$PROJECT_ROOT/slite.pid"
LOGFILE="$PROJECT_ROOT/slite.log"

# 运行模式：dev 或 prod
# dev: go run ./cmd/server
# prod: 直接运行编译好的二进制文件
MODE="${MODE:-dev}"

# 根据模式设置执行命令
if [ "$MODE" = "prod" ]; then
    # 生产模式：运行编译好的二进制文件
    if [ -f "$PROJECT_ROOT/dist/bin/slite-server" ]; then
        EXEC="$PROJECT_ROOT/dist/bin/slite-server"
    elif [ -f "$PROJECT_ROOT/slite-server" ]; then
        EXEC="$PROJECT_ROOT/slite-server"
    else
        echo -e "${RED}错误: 找不到 slite-server 二进制文件${NC}"
        echo "请先编译: go build -o slite-server ./cmd/server"
        exit 1
    fi
else
    # 开发模式：使用 go run
    EXEC="go run $PROJECT_ROOT/cmd/server/main.go"
fi

# ==============================================================================
# 工具函数
# ==============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# 检查服务是否运行
is_running() {
    if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# ==============================================================================
# 服务管理函数
# ==============================================================================

start() {
    if is_running; then
        log_warning "服务已在运行 (PID: $(cat "$PIDFILE"))"
        return 0
    fi

    log_info "正在启动服务..."
    
    # 确保 PID 文件被清理
    rm -f "$PIDFILE"
    
    # 启动服务
    cd "$PROJECT_ROOT"
    nohup $EXEC >> "$LOGFILE" 2>&1 &
    local pid=$!
    echo $pid > "$PIDFILE"
    
    sleep 2
    
    if is_running; then
        log_success "服务已启动，PID: $pid，日志: $LOGFILE"
        log_info "运行模式: $MODE"
    else
        log_error "服务启动失败，请检查日志: $LOGFILE"
        tail -20 "$LOGFILE" 2>/dev/null || true
        return 1
    fi
}

stop() {
    if ! is_running; then
        if [ -f "$PIDFILE" ]; then
            rm -f "$PIDFILE"
        fi
        # 尝试通过进程名终止
        log_info "尝试通过进程名终止..."
        pkill -f "slite-server" 2>/dev/null && log_success "已终止" || log_warning "未找到运行中的服务"
        return 0
    fi
    
    local pid=$(cat "$PIDFILE")
    log_info "正在停止服务 (PID: $pid)..."
    
    # 先尝试优雅停止
    kill "$pid" 2>/dev/null || true
    
    # 等待最多 10 秒
    local count=0
    while kill -0 "$pid" 2>/dev/null && [ $count -lt 10 ]; do
        sleep 1
        count=$((count + 1))
    done
    
    # 如果还在运行，强制终止
    if kill -0 "$pid" 2>/dev/null; then
        log_warning "服务未响应，强制终止..."
        kill -9 "$pid" 2>/dev/null || true
        sleep 1
    fi
    
    rm -f "$PIDFILE"
    log_success "服务已停止"
}

restart() {
    log_info "正在重启服务..."
    stop
    sleep 1
    start
}

logs() {
    if [ -f "$LOGFILE" ]; then
        log_info "正在显示实时日志（按 Ctrl+C 退出）..."
        tail -f "$LOGFILE"
    else
        log_error "日志文件 $LOGFILE 不存在"
    fi
}

status() {
    if is_running; then
        local pid=$(cat "$PIDFILE")
        log_success "服务运行中，PID: $pid"
        log_info "运行模式: $MODE"
        # 显示进程信息
        ps -p "$pid" -o pid,ppid,etime,args 2>/dev/null | tail -n +2 || true
    else
        log_warning "服务未运行"
    fi
}

usage() {
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  start     启动服务"
    echo "  stop      停止服务"
    echo "  restart   重启服务"
    echo "  logs      查看实时日志"
    echo "  status    查看运行状态"
    echo "  menu      显示交互式菜单 (默认)"
    echo ""
    echo "环境变量:"
    echo "  MODE=dev   开发模式 (go run) [默认]"
    echo "  MODE=prod  生产模式 (运行二进制文件)"
    echo ""
    echo "示例:"
    echo "  $0                  # 进入交互式菜单"
    echo "  $0 start            # 开发模式启动"
    echo "  MODE=prod $0 start  # 生产模式启动"
}

# ==============================================================================
# 显示菜单
# ==============================================================================

show_menu() {
    clear
    echo "===================================================="
    echo "              Slite 服务管理菜单"
    echo "===================================================="
    echo "  运行模式: $MODE"
    echo ""
    echo "  1. 启动服务"
    echo "  2. 停止服务"
    echo "  3. 重启服务"
    echo "  4. 查看实时日志"
    echo "  5. 查看运行状态"
    echo "  0. 退出"
    echo "===================================================="
    echo -n "请选择操作 [0-5]: "
}

# ==============================================================================
# 主函数
# ==============================================================================

main() {
    # 如果没有参数，默认进入交互式菜单
    if [ $# -eq 0 ]; then
        while true; do
            show_menu
            read choice
            case $choice in
                1) start ;;
                2) stop ;;
                3) restart ;;
                4) logs ;;
                5) status ;;
                0) echo "退出"; exit 0 ;;
                *) echo -e "${RED}无效输入，请重新选择${NC}"; sleep 1 ;;
            esac
            if [[ "$choice" != "4" && "$choice" != "0" ]]; then
                echo -n "按回车键返回菜单..."
                read
            fi
        done
        exit 0
    fi

    # 有参数时处理命令
    case "${1:-}" in
        start)
            start
            ;;
        stop)
            stop
            ;;
        restart)
            restart
            ;;
        logs)
            logs
            ;;
        status)
            status
            ;;
        menu)
            while true; do
                show_menu
                read choice
                case $choice in
                    1) start ;;
                    2) stop ;;
                    3) restart ;;
                    4) logs ;;
                    5) status ;;
                    0) echo "退出"; exit 0 ;;
                    *) echo -e "${RED}无效输入，请重新选择${NC}"; sleep 1 ;;
                esac
                if [[ "$choice" != "4" && "$choice" != "0" ]]; then
                    echo -n "按回车键返回菜单..."
                    read
                fi
            done
            ;;
        -h|--help|help)
            usage
            ;;
        *)
            echo -e "${RED}未知命令: $1${NC}"
            usage
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"