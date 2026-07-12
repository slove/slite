#!/bin/bash

# ==============================================================================
# Slite Monitor Agent 安装脚本
# 位置: scripts/install.sh
# 用法: ./scripts/install.sh --id <节点ID> --url <服务器地址>
# ==============================================================================

set -euo pipefail

# ==============================================================================
# 初始化参数
# ==============================================================================

NODE_ID=""
NODE_NAME=""
NODE_KEY=""
MASTER_URL=""
INSTALL_DIR="/opt/slite-agent"
SILENT_MODE=false

# 系统检测
OS_TYPE=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${ARCH}" in
  x86_64) ARCH_TYPE="amd64" ;;
  aarch64|arm64) ARCH_TYPE="arm64" ;;
  armv7l) ARCH_TYPE="armv7" ;;
  armv6l) ARCH_TYPE="armv6" ;;
  i386|i686) ARCH_TYPE="386" ;;
  *) ARCH_TYPE="amd64" ;;
esac

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ==============================================================================
# 工具函数
# ==============================================================================

log_info() {
    echo -e "${BLUE}[INFO] $1${NC}"
}

log_success() {
    echo -e "${GREEN}[✓] $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}[!] $1${NC}"
}

log_error() {
    echo -e "${RED}[✗] $1${NC}"
    exit 1
}

# 跨平台获取文件大小
get_file_size() {
    local file="$1"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        stat -f%z "$file" 2>/dev/null || echo "0"
    else
        stat -c%s "$file" 2>/dev/null || echo "0"
    fi
}

# ==============================================================================
# 参数解析
# ==============================================================================

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                NODE_ID="$2"
                shift 2
                ;;
            --name)
                NODE_NAME="$2"
                shift 2
                ;;
            --key)
                NODE_KEY="$2"
                shift 2
                ;;
            --url)
                MASTER_URL="$2"
                shift 2
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --silent)
                SILENT_MODE=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
}

# ==============================================================================
# 验证参数
# ==============================================================================

validate_arguments() {
    if [ -z "$NODE_ID" ]; then
        log_error "错误: 必须提供 --id 参数"
    fi
    
    if [ -z "$MASTER_URL" ]; then
        log_error "错误: 必须提供 --url 参数"
    fi
    
    if [ -z "$NODE_NAME" ]; then
        NODE_NAME=$(hostname)
        log_info "未提供节点名称，使用主机名: $NODE_NAME"
    fi
    
    if [ -z "$NODE_KEY" ]; then
        NODE_KEY="$NODE_ID"
        log_info "未提供节点密钥，使用节点ID作为密钥"
    fi
}

# ==============================================================================
# 服务器连通性测试
# ==============================================================================

test_server_connection() {
    log_info "测试监控服务器连通性..."
    
    local server_url=$(echo "$MASTER_URL" | sed 's/\/$//')
    
    if curl -s --max-time 10 "$server_url/api/report" >/dev/null 2>&1; then
        log_success "监控服务器连接正常"
        return 0
    fi
    
    local host_port=$(echo "$server_url" | sed -e 's|http://||' -e 's|https://||')
    local host=$(echo "$host_port" | cut -d: -f1)
    local port=$(echo "$host_port" | cut -d: -f2)
    [ -z "$port" ] && port=80
    
    if timeout 3 bash -c "cat < /dev/null > /dev/tcp/$host/$port" 2>/dev/null; then
        log_warning "服务器端口可达，但HTTP访问失败"
        log_warning "可能是服务器配置问题，安装将继续"
        return 0
    else
        log_error "无法连接到监控服务器 $host:$port"
        exit 1
    fi
}

# ==============================================================================
# 下载和安装
# ==============================================================================

stop_existing_agent() {
    log_info "停止可能存在的旧进程..."
    
    local pids=$(pgrep -f "slite-agent.*-id $NODE_ID" 2>/dev/null || true)
    
    if [ -n "$pids" ]; then
        kill $pids 2>/dev/null || true
        sleep 2
        
        if pgrep -f "slite-agent.*-id $NODE_ID" >/dev/null; then
            kill -9 $pids 2>/dev/null || true
            sleep 1
        fi
        log_success "已停止旧进程"
    fi
}

download_agent() {
    log_info "下载 Agent 二进制文件..."
    
    local temp_file="$INSTALL_DIR/slite-agent.tmp"
    local final_file="$INSTALL_DIR/slite-agent"
    
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
    
    local download_url="$MASTER_URL/dist/bin/slite-agent-$OS_TYPE-$ARCH_TYPE"
    
    log_info "下载地址: $download_url"
    
    if curl -L --max-time 30 --retry 3 -o "$temp_file" "$download_url"; then
        if [ -s "$temp_file" ]; then
            rm -f "$final_file" 2>/dev/null || true
            mv "$temp_file" "$final_file"
            chmod +x "$final_file"
            local file_size=$(get_file_size "$final_file")
            log_success "Agent 下载成功 ($file_size 字节)"
        else
            rm -f "$temp_file"
            log_error "下载的文件为空"
        fi
    else
        log_info "curl下载失败，尝试使用wget..."
        if command -v wget >/dev/null; then
            if wget -q --timeout=30 --tries=3 -O "$temp_file" "$download_url"; then
                if [ -s "$temp_file" ]; then
                    rm -f "$final_file" 2>/dev/null || true
                    mv "$temp_file" "$final_file"
                    chmod +x "$final_file"
                    log_success "Agent 下载成功 (使用wget)"
                else
                    rm -f "$temp_file"
                    log_error "wget下载的文件为空"
                fi
            else
                log_error "下载失败，请检查:"
                log_error "1. 网络连接"
                log_error "2. 服务器地址是否正确"
                log_error "3. 防火墙设置"
                exit 1
            fi
        else
            log_error "下载失败，且wget不可用"
            exit 1
        fi
    fi
}

create_config() {
    log_info "创建配置文件..."
    
    cat > "$INSTALL_DIR/slite-agent.conf" << EOF
# Slite Agent 配置文件
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')

NODE_ID="$NODE_ID"
NODE_NAME="$NODE_NAME"
NODE_KEY="$NODE_KEY"
MASTER_URL="$MASTER_URL"
REPORT_INTERVAL="10"

# 系统信息
OS_TYPE="$OS_TYPE"
ARCH="$ARCH"
INSTALL_DIR="$INSTALL_DIR"
EOF
    
    log_success "配置文件创建完成"
}

start_agent() {
    log_info "启动 Agent..."
    
    cd "$INSTALL_DIR"
    
    pkill -f "slite-agent.*-id $NODE_ID" 2>/dev/null || true
    sleep 1
    
    > slite-agent.log 2>/dev/null || true
    
    nohup ./slite-agent \
        -id "$NODE_ID" \
        -url "$MASTER_URL" \
        -name "$NODE_NAME" \
        -key "$NODE_KEY" \
        >> slite-agent.log 2>&1 &
    
    local max_attempts=10
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if pgrep -f "slite-agent.*-id $NODE_ID" >/dev/null; then
            log_success "Agent 启动成功 (PID: $(pgrep -f "slite-agent.*-id $NODE_ID"))"
            
            echo ""
            log_info "启动日志:"
            if [ -f "slite-agent.log" ]; then
                tail -5 slite-agent.log
            fi
            return 0
        fi
        sleep 1
        attempt=$((attempt + 1))
    done
    
    log_error "Agent 启动失败"
    if [ -f "slite-agent.log" ]; then
        log_info "错误日志:"
        tail -20 slite-agent.log
    fi
    exit 1
}

# ==============================================================================
# 主函数
# ==============================================================================

main() {
    echo "========================================"
    echo "    Slite Monitor Agent 安装程序"
    echo "========================================"
    
    parse_arguments "$@"
    validate_arguments
    
    log_info "安装参数:"
    log_info "  • 节点ID: $NODE_ID"
    log_info "  • 节点名称: $NODE_NAME"
    log_info "  • 服务器: $MASTER_URL"
    log_info "  • 安装目录: $INSTALL_DIR"
    echo ""
    
    test_server_connection
    stop_existing_agent
    download_agent
    create_config
    start_agent
    
    echo ""
    echo "========================================"
    echo "        安装完成!"
    echo "========================================"
    echo ""
    echo "节点已启动并开始上报数据到:"
    echo "  $MASTER_URL"
    echo ""
    echo "重要文件:"
    echo "  • 二进制文件: $INSTALL_DIR/slite-agent"
    echo "  • 配置文件: $INSTALL_DIR/slite-agent.conf"
    echo "  • 日志文件: $INSTALL_DIR/slite-agent.log"
    echo ""
    echo "查看实时日志:"
    echo "  tail -f $INSTALL_DIR/slite-agent.log"
    echo ""
    echo "停止Agent:"
    echo "  pkill -f 'slite-agent.*-id $NODE_ID'"
    echo ""
    echo "重新启动:"
    echo "  cd $INSTALL_DIR && nohup ./slite-agent \\"
    echo "    -id \"$NODE_ID\" \\"
    echo "    -url \"$MASTER_URL\" \\"
    echo "    -name \"$NODE_NAME\" \\"
    echo "    -key \"$NODE_KEY\" &"
    echo "========================================"
}

# 执行主函数
main "$@"