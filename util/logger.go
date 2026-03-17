package util

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// LogLevel 日志级别
type LogLevel string

const (
	// LogLevelDebug 调试级别
	LogLevelDebug LogLevel = "DEBUG"
	// LogLevelInfo 信息级别
	LogLevelInfo LogLevel = "INFO"
	// LogLevelWarn 警告级别
	LogLevelWarn LogLevel = "WARN"
	// LogLevelError 错误级别
	LogLevelError LogLevel = "ERROR"
	// LogLevelFatal 致命级别
	LogLevelFatal LogLevel = "FATAL"
)

// ErrorLevel 错误级别
type ErrorLevel string

const (
	// ErrorLevelNonFatal 非致命错误
	ErrorLevelNonFatal ErrorLevel = "NON_FATAL"
	// ErrorLevelFatal 致命错误
	ErrorLevelFatal ErrorLevel = "FATAL"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp    string      `json:"timestamp"`
	Level        LogLevel    `json:"level"`
	ErrorLevel   ErrorLevel  `json:"errorLevel,omitempty"`
	Message      string      `json:"message"`
	Error        string      `json:"error,omitempty"`
	Stack        string      `json:"stack,omitempty"`
	Path         string      `json:"path,omitempty"`
	Method       string      `json:"method,omitempty"`
	Params       interface{} `json:"params,omitempty"`
	ClientIP     string      `json:"clientIP,omitempty"`
	StatusCode   int         `json:"statusCode,omitempty"`
	ResponseTime string      `json:"responseTime,omitempty"`
}

// Logger 日志记录器
type Logger struct{}

// NewLogger 创建新的日志记录器
func NewLogger() *Logger {
	return &Logger{}
}

// log 记录日志的内部方法
func (l *Logger) log(level LogLevel, errorLevel ErrorLevel, message string, err error, stack string, path string, method string, reqBody any, clientIP string, statusCode int, responseTime string) {
	// 获取模块名称和业务标识
	module, businessID := getModuleAndBusinessID()

	// 过滤堆栈信息
	filteredStack := filterStack(stack)

	// 控制台友好格式输出
	consoleFormat := formatConsoleLog(level, message, err, module, businessID, path, method, statusCode, responseTime, clientIP, reqBody, filteredStack)
	fmt.Println(consoleFormat)

	// // 同时输出JSON格式（可选，用于日志分析工具）
	// if level >= LogLevelError {
	// 	entry := LogEntry{
	// 		Timestamp:    time.Now().Format(time.RFC3339),
	// 		Level:        level,
	// 		ErrorLevel:   errorLevel,
	// 		Message:      message,
	// 		Error:        getErrorString(err),
	// 		Stack:        filteredStack,
	// 		Path:         path,
	// 		Method:       method,
	// 		Params:       sanitizeParams(params),
	// 		ClientIP:     clientIP,
	// 		StatusCode:   statusCode,
	// 		ResponseTime: responseTime,
	// 	}
	// 	jsonData, err := json.Marshal(entry)
	// 	if err == nil {
	// 		fmt.Println("[JSON]", string(jsonData))
	// 	}
	// }
}

// Debug 记录调试级别的日志
func (l *Logger) Debug(message string, params ...interface{}) {
	l.log(LogLevelDebug, "", message, nil, "", "", "", params, "", 0, "")
}

// Info 记录信息级别的日志
func (l *Logger) Info(message string, params ...interface{}) {
	l.log(LogLevelInfo, "", message, nil, "", "", "", params, "", 0, "")
}

// Warn 记录警告级别的日志
func (l *Logger) Warn(message string, err error, params ...interface{}) {
	// 警告级别只记录错误信息，不记录堆栈
	l.log(LogLevelWarn, ErrorLevelNonFatal, message, err, "", "", "", params, "", 0, "")
}

// Error 记录错误级别的日志
func (l *Logger) Error(message string, err error, params ...interface{}) {
	// 错误级别记录完整堆栈
	l.log(LogLevelError, ErrorLevelNonFatal, message, err, getStack(), "", "", params, "", 0, "")
}

// Fatal 记录致命级别的日志
func (l *Logger) Fatal(message string, err error, params ...interface{}) {
	// 致命级别记录完整堆栈
	l.log(LogLevelFatal, ErrorLevelFatal, message, err, getStack(), "", "", params, "", 0, "")
}

// LogAPIError 记录API错误
func (l *Logger) LogAPIError(message string, err error, path string, method string, reqBody any, clientIP string, statusCode int, responseTime string) {
	errorLevel := ErrorLevelNonFatal
	if statusCode >= 500 {
		errorLevel = ErrorLevelFatal
	}

	// 根据错误级别决定是否记录堆栈
	stack := ""
	if statusCode >= 500 {
		// 5xx错误记录完整堆栈
		stack = getStack()
	}

	l.log(LogLevelError, errorLevel, message, err, stack, path, method, reqBody, clientIP, statusCode, responseTime)
}

// getErrorString 获取错误字符串
func getErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// getStack 获取堆栈信息
func getStack() string {
	stack := make([]byte, 4096)
	n := runtime.Stack(stack, false)
	return string(stack[:n])
}

// sanitizeParams 清理参数，移除敏感信息
func sanitizeParams(params interface{}) interface{} {
	if params == nil {
		return nil
	}

	// 转换为JSON字符串
	data, err := json.Marshal(params)
	if err != nil {
		return params
	}

	// 替换敏感信息
	jsonStr := string(data)
	jsonStr = strings.ReplaceAll(jsonStr, "password", "[REDACTED]")
	jsonStr = strings.ReplaceAll(jsonStr, "token", "[REDACTED]")
	jsonStr = strings.ReplaceAll(jsonStr, "secret", "[REDACTED]")
	jsonStr = strings.ReplaceAll(jsonStr, "key", "[REDACTED]")

	// 解析回对象
	var result interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return params
	}

	return result
}

// getModuleAndBusinessID 获取模块名称和业务标识
func getModuleAndBusinessID() (string, string) {
	// 获取调用者信息
	pc, _, _, ok := runtime.Caller(4) // 跳过4层调用栈
	if !ok {
		return "unknown", ""
	}

	funcName := runtime.FuncForPC(pc).Name()
	// 提取模块名称（包路径的最后部分）
	parts := strings.Split(funcName, "/")
	module := "unknown"
	if len(parts) > 0 {
		module = parts[len(parts)-1]
		// 去除函数名，只保留模块名
		module = strings.Split(module, ".")[0]
	}

	// 这里可以根据实际业务逻辑提取业务标识
	// 例如从参数中获取用户ID、订单ID等
	businessID := ""

	return module, businessID
}

// filterStack 过滤堆栈信息，只保留与业务代码相关的部分
func filterStack(stack string) string {
	if stack == "" {
		return ""
	}

	lines := strings.Split(stack, "\n")
	var filteredLines []string

	// 过滤掉框架和标准库的调用
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 保留包含项目包路径的行
		if strings.Contains(line, "scim-go/") {
			// 简化路径，只保留相对路径
			simplified := line
			if idx := strings.Index(simplified, "scim-go/"); idx != -1 {
				simplified = simplified[idx:]
			}
			filteredLines = append(filteredLines, simplified)
		}
	}

	// 限制堆栈深度，最多保留10行
	if len(filteredLines) > 10 {
		filteredLines = filteredLines[:10]
	}

	return strings.Join(filteredLines, "\n")
}

// formatConsoleLog 格式化控制台日志输出
func formatConsoleLog(level LogLevel, message string, err error, module string, businessID string, path string, method string, statusCode int, responseTime string, clientIP string, reqBody any, stack string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// 颜色代码
	var levelColor, resetColor string
	switch level {
	case LogLevelDebug:
		levelColor = "\033[36m" // 青色
	case LogLevelInfo:
		levelColor = "\033[32m" // 绿色
	case LogLevelWarn:
		levelColor = "\033[33m" // 黄色
	case LogLevelError:
		levelColor = "\033[31m" // 红色
	case LogLevelFatal:
		levelColor = "\033[35m" // 紫色
	default:
		levelColor = "\033[0m" // 默认
	}
	resetColor = "\033[0m"

	// 构建日志格式
	logFormat := fmt.Sprintf("%s | %s%-5s%s | %-15s | %s",
		timestamp,
		levelColor, level, resetColor,
		module,
		message,
	)

	// 添加业务标识
	if businessID != "" {
		logFormat += fmt.Sprintf(" [ID: %s]", businessID)
	}

	// 添加错误信息
	if err != nil {
		logFormat += fmt.Sprintf(" | Error: %s", err.Error())
	}

	// 添加API相关信息
	if path != "" {
		logFormat += fmt.Sprintf(" | %s %s", method, path)
	}

	// 添加状态码和响应时间
	if statusCode > 0 {
		logFormat += fmt.Sprintf(" | Status: %d", statusCode)
	}
	if responseTime != "" {
		logFormat += fmt.Sprintf(" | Time: %s", responseTime)
	}

	// 添加客户端IP
	if clientIP != "" {
		logFormat += fmt.Sprintf(" | IP: %s", clientIP)
	}
	if reqBody != nil {
		if bodyBytes, err := json.Marshal(reqBody); err == nil {
			logFormat += fmt.Sprintf(" | body: %v", string(bodyBytes))
		}
	}

	// 添加堆栈信息（只在有堆栈且级别为错误或以上时）
	if stack != "" && (level == LogLevelError || level == LogLevelFatal) {
		logFormat += "\n[Stack] " + stack
	}

	return logFormat
}

// 全局日志记录器实例
var globalLogger = NewLogger()

// Debug 全局调试日志
func Debug(message string, params ...interface{}) {
	globalLogger.Debug(message, params...)
}

// Info 全局信息日志
func Info(message string, params ...interface{}) {
	globalLogger.Info(message, params...)
}

// Warn 全局警告日志
func Warn(message string, err error, params ...interface{}) {
	globalLogger.Warn(message, err, params...)
}

// Error 全局错误日志
func Error(message string, err error, params ...interface{}) {
	globalLogger.Error(message, err, params...)
}

// Fatal 全局致命错误日志
func Fatal(message string, err error, params ...interface{}) {
	globalLogger.Fatal(message, err, params...)
}

// LogAPIError 全局API错误日志
func LogAPIError(message string, err error, path string, method string, reqBody any, clientIP string, statusCode int, responseTime string) {
	globalLogger.LogAPIError(message, err, path, method, reqBody, clientIP, statusCode, responseTime)
}
