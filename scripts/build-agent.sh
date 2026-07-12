#!/bin/bash

# ==============================================================================
# Slite Agent 全平台交叉编译脚本
# 从 cmd/agent/main.go 编译
# ==============================================================================

set -e

# 1. 确定目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
OUTPUT_DIR="$PROJECT_ROOT/dist/bin"

mkdir -p "$OUTPUT_DIR"

# 2. 定义编译平台 (统一格式：OS/ARCH/SUFFIX)
PLATFORMS=(
    "linux/amd64/none"
    "linux/arm64/none"
    "windows/amd64/.exe"
    "windows/arm64/.exe"
    "darwin/amd64/none"
    "darwin/arm64/none"
)

echo "--- 开始全平台交叉编译 (Source: cmd/agent/main.go) ---"
cd "$PROJECT_ROOT"

# 检查源文件是否存在
SOURCE_FILE="cmd/agent/main.go"
if [ ! -f "$SOURCE_FILE" ]; then
    echo "错误: 找不到 $SOURCE_FILE 文件！"
    exit 1
fi

for PLATFORM in "${PLATFORMS[@]}"; do
    IFS="/" read -r OS ARCH SUFFIX <<< "$PLATFORM"
    
    EXT=""
    if [ "$SUFFIX" != "none" ]; then
        EXT="$SUFFIX"
    fi
    
    OUTPUT_NAME="$OUTPUT_DIR/slite-agent-${OS}-${ARCH}${EXT}"
    
    echo -n "正在编译: ${OS}/${ARCH} ... "
    
    CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH \
    go build -buildmode=exe -ldflags="-s -w" -o "$OUTPUT_NAME" "$SOURCE_FILE"
    
    if [ $? -eq 0 ]; then
        echo "成功"
    else
        echo "失败"
    fi
done

echo "------------------------------------------------"
echo "编译完成！文件位置: $OUTPUT_DIR"
echo "------------------------------------------------"

echo "正在验证文件格式..."
for FILE in "$OUTPUT_DIR"/slite-agent-linux-*; do
    if [ -f "$FILE" ]; then
        FILE_TYPE=$(file "$FILE")
        if [[ "$FILE_TYPE" =~ "archive" ]]; then
            echo -e "\033[31m警告: $FILE 仍然是归档格式！请检查环境。\033[0m"
        else
            echo -e "\033[32m通过: $FILE_TYPE\033[0m"
        fi
    fi
done

ls -lh "$OUTPUT_DIR"