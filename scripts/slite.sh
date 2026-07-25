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

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

PIDFILE="$PROJECT_ROOT/slite.pid"
LOGFILE="$PROJECT_ROOT/slite.log"

# 运行模式：dev 或 prod
MODE="${MODE:-dev}"
# 服务监听端口，用于检测残留占用
PORT="${PORT:-8090}"

if [ "$MODE" = "prod" ]; then
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
    EXEC="go run $PROJECT_ROOT/cmd/server/main.go"
fi

# ==============================================================================
# 工具函数
# ==============================================================================

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[✓]${NC} $1"; }
log_error()   { echo -e "${RED}[✗]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[!]${NC} $1"; }

# 查真实占用端口的进程 PID（优先 ss，没有就用 lsof）
# 注意：这里每个管道末尾都加了 || true —— 查不到进程是正常情况（端口空闲），
# 在 set -o pipefail 下如果不加这个，空结果会被当成命令失败，
# 而 port_pid=$(get_port_pid) 这种赋值写法在 set -e 下会导致脚本直接静默退出。
get_port_pid() {
    if command -v ss >/dev/null 2>&1; then
        ss -ltnp 2>/dev/null | awk -v p=":$PORT" '$4 ~ p {print $0}' | grep -oP '(?<=pid=)\d+' | head -1 || true
    elif command -v lsof >/dev/null 2>&1; then
        lsof -ti tcp:"$PORT" 2>/dev/null | head -1 || true
    fi
}

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

    # PID 文件说没在跑，不代表端口真的空着——查一下真实占用情况
    local port_pid
    port_pid=$(get_port_pid)
    if [ -n "$port_pid" ]; then
        log_error "端口 $PORT 已被进程 $port_pid 占用（PID 文件未记录该进程，可能是残留进程）"
        log_info "运行 '$0 stop' 可自动清理占用端口的残留进程，或手动执行: kill -9 $port_pid"
        return 1
    fi

    log_info "正在启动服务..."
    rm -f "$PIDFILE"

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
    local acted=false

    if is_running; then
        local pid
        pid=$(cat "$PIDFILE")
        log_info "正在停止服务 (PID: $pid)..."

        # dev 模式下 go run 会 fork 出真正监听端口的子进程，必须一起杀
        pkill -P "$pid" 2>/dev/null || true
        kill "$pid" 2>/dev/null || true

        local count=0
        while kill -0 "$pid" 2>/dev/null && [ $count -lt 10 ]; do
            sleep 1
            count=$((count + 1))
        done

        if kill -0 "$pid" 2>/dev/null; then
            log_warning "服务未响应，强制终止..."
            pkill -9 -P "$pid" 2>/dev/null || true
            kill -9 "$pid" 2>/dev/null || true
            sleep 1
        fi

        rm -f "$PIDFILE"
        acted=true
    else
        rm -f "$PIDFILE" 2>/dev/null || true
    fi

    # 不管 PID 文件是否有效，都再核实一次端口是否真的释放了
    local port_pid
    port_pid=$(get_port_pid)
    if [ -n "$port_pid" ]; then
        log_warning "检测到端口 $PORT 仍被进程 $port_pid 占用，强制终止..."
        kill -9 "$port_pid" 2>/dev/null || true
        sleep 1
        acted=true
    fi

    if [ "$acted" = true ]; then
        log_success "服务已停止"
    else
        log_warning "未找到运行中的服务"
    fi
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
        local pid
        pid=$(cat "$PIDFILE")
        log_success "服务运行中，PID: $pid"
        log_info "运行模式: $MODE"
        ps -p "$pid" -o pid,ppid,etime,args 2>/dev/null | tail -n +2 || true
    else
        log_warning "服务未运行（PID 文件）"
    fi

    local port_pid
    port_pid=$(get_port_pid)
    if [ -n "$port_pid" ]; then
        log_info "端口 $PORT 当前被 PID $port_pid 占用"
    else
        log_info "端口 $PORT 当前空闲"
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
    echo "  PORT=8090  服务监听端口 [默认 8090]"
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
    echo "  运行模式: $MODE  |  端口: $PORT"
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

main "$@"