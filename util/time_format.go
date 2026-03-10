package util

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// TimePrecisionLevel 时间精确度级别
type TimePrecisionLevel string

const (
	// TimePrecisionSecond 秒级精确度
	TimePrecisionSecond TimePrecisionLevel = "second"
	// TimePrecisionMinute 分钟级精确度
	TimePrecisionMinute TimePrecisionLevel = "minute"
	// TimePrecisionMillisecond 毫秒级精确度
	TimePrecisionMillisecond TimePrecisionLevel = "millisecond"
	// TimePrecisionMicrosecond 微秒级精确度
	TimePrecisionMicrosecond TimePrecisionLevel = "microsecond"
	// TimePrecisionNanosecond 纳秒级精确度
	TimePrecisionNanosecond TimePrecisionLevel = "nanosecond"
)

// TimeFormatter 时间格式化器
type TimeFormatter struct {
	level  TimePrecisionLevel
	format string
	mutex  sync.RWMutex
}

// globalFormatter 全局时间格式化器实例
var globalFormatter = &TimeFormatter{
	level:  TimePrecisionSecond,
	format: "",
}

// SetTimePrecision 设置全局时间精确度配置
func SetTimePrecision(level string, format string) error {
	globalFormatter.mutex.Lock()
	defer globalFormatter.mutex.Unlock()

	// 验证并设置级别
	validLevels := map[string]TimePrecisionLevel{
		"second":      TimePrecisionSecond,
		"minute":      TimePrecisionMinute,
		"millisecond": TimePrecisionMillisecond,
		"microsecond": TimePrecisionMicrosecond,
		"nanosecond":  TimePrecisionNanosecond,
	}

	precisionLevel, exists := validLevels[strings.ToLower(level)]
	if !exists {
		return fmt.Errorf("invalid time precision level: %s", level)
	}

	globalFormatter.level = precisionLevel

	// 如果提供了自定义格式，使用自定义格式
	if format != "" {
		globalFormatter.format = format
	} else {
		// 根据级别自动生成格式
		globalFormatter.format = getDefaultFormat(precisionLevel)
	}

	return nil
}

// GetTimePrecision 获取当前时间精确度配置
func GetTimePrecision() (TimePrecisionLevel, string) {
	globalFormatter.mutex.RLock()
	defer globalFormatter.mutex.RUnlock()
	return globalFormatter.level, globalFormatter.format
}

// getDefaultFormat 根据精确度级别获取默认格式
func getDefaultFormat(level TimePrecisionLevel) string {
	switch level {
	case TimePrecisionMinute:
		return "2006-01-02 15:04"
	case TimePrecisionSecond:
		return "2006-01-02 15:04:05"
	case TimePrecisionMillisecond:
		return "2006-01-02 15:04:05.000"
	case TimePrecisionMicrosecond:
		return "2006-01-02 15:04:05.000000"
	case TimePrecisionNanosecond:
		return "2006-01-02 15:04:05.000000000"
	default:
		return "2006-01-02 15:04:05"
	}
}

// FormatTime 根据配置格式化时间
func FormatTime(t time.Time) string {
	globalFormatter.mutex.RLock()
	defer globalFormatter.mutex.RUnlock()

	if t.IsZero() {
		return ""
	}

	return t.Format(globalFormatter.format)
}

// FormatTimeToISO8601 根据 SCIM 2.0 规范格式化时间为ISO 8601格式（用于API响应）
func FormatTimeToISO8601(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	// SCIM 2.0 规范要求：ISO 8601 日期时间格式，精确到秒级别，包含时区信息
	// 格式示例：2024-04-28T11:28:38+00:00
	return t.Format("2006-01-02T15:04:05Z07:00")
}

// ParseTime 根据配置解析时间字符串
func ParseTime(timeStr string) (time.Time, error) {
	globalFormatter.mutex.RLock()
	defer globalFormatter.mutex.RUnlock()

	if timeStr == "" {
		return time.Time{}, nil
	}

	// 尝试使用当前配置的格式解析
	t, err := time.Parse(globalFormatter.format, timeStr)
	if err == nil {
		return t, nil
	}

	// 尝试解析ISO 8601格式
	formats := []string{
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05.999999Z07:00",
		"2006-01-02T15:04:05.999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}

	for _, format := range formats {
		t, err = time.Parse(format, timeStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

// TruncateTime 根据精确度级别截断时间
func TruncateTime(t time.Time) time.Time {
	globalFormatter.mutex.RLock()
	defer globalFormatter.mutex.RUnlock()

	switch globalFormatter.level {
	case TimePrecisionMinute:
		return t.Truncate(time.Minute)
	case TimePrecisionSecond:
		return t.Truncate(time.Second)
	case TimePrecisionMillisecond:
		return t.Truncate(time.Millisecond)
	case TimePrecisionMicrosecond:
		return t.Truncate(time.Microsecond)
	case TimePrecisionNanosecond:
		return t // 纳秒级不需要截断
	default:
		return t.Truncate(time.Second)
	}
}

// GetTimePrecisionLevel 获取当前时间精确度级别
func GetTimePrecisionLevel() TimePrecisionLevel {
	globalFormatter.mutex.RLock()
	defer globalFormatter.mutex.RUnlock()
	return globalFormatter.level
}

// IsValidTimePrecisionLevel 验证时间精确度级别是否有效
func IsValidTimePrecisionLevel(level string) bool {
	validLevels := []string{"second", "minute", "millisecond", "microsecond", "nanosecond"}
	level = strings.ToLower(level)
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// FormatTimeForLog 格式化时间用于日志记录
func FormatTimeForLog(t time.Time) string {
	globalFormatter.mutex.RLock()
	defer globalFormatter.mutex.RUnlock()

	if t.IsZero() {
		return ""
	}

	// 日志使用标准格式，但可以根据精确度调整
	switch globalFormatter.level {
	case TimePrecisionMinute:
		return t.Format("2006-01-02 15:04")
	case TimePrecisionSecond:
		return t.Format("2006-01-02 15:04:05")
	case TimePrecisionMillisecond:
		return t.Format("2006-01-02 15:04:05.000")
	case TimePrecisionMicrosecond:
		return t.Format("2006-01-02 15:04:05.000000")
	case TimePrecisionNanosecond:
		return t.Format("2006-01-02 15:04:05.000000000")
	default:
		return t.Format("2006-01-02 15:04:05")
	}
}

// GetLogTimeFormat 获取日志时间格式字符串
func GetLogTimeFormat() string {
	globalFormatter.mutex.RLock()
	defer globalFormatter.mutex.RUnlock()

	switch globalFormatter.level {
	case TimePrecisionMinute:
		return "2006-01-02 15:04"
	case TimePrecisionSecond:
		return "2006-01-02 15:04:05"
	case TimePrecisionMillisecond:
		return "2006-01-02 15:04:05.000"
	case TimePrecisionMicrosecond:
		return "2006-01-02 15:04:05.000000"
	case TimePrecisionNanosecond:
		return "2006-01-02 15:04:05.000000000"
	default:
		return "2006-01-02 15:04:05"
	}
}
