package utils

import (
	"fmt"
	"time"

	"slite/internal/global"
)

// FormatBytes 格式化字节大小（导出供其他包使用）
func FormatBytes(b int64) string {
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

// GenerateSnowflakeID 生成雪花ID（导出供其他包使用）
func GenerateSnowflakeID() string {
	if global.GlobalSnowflake != nil {
		return global.GlobalSnowflake.Generate().String()
	}
	return fmt.Sprintf("%d", time.Now().UnixNano()/1000000)
}
