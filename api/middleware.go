package api

import (
	"fmt"
	"net/http"
	"scim-go/model"
	"scim-go/util"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Auth SCIM Bearer Token认证中间件
// 支持缓存验证结果以提升性能
func Auth(validToken string) gin.HandlerFunc {
	// 预计算Bearer前缀，避免每次请求都进行字符串拼接
	expectedPrefix := "Bearer " + validToken

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// 快速路径：直接比较，避免字符串分割操作
		if authHeader != expectedPrefix {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Schemas:  model.ErrorSchema.String(),
				Detail:   "Invalid or missing Bearer Token",
				Status:   http.StatusUnauthorized,
				ScimType: "invalidToken",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// BindQuery 绑定SCIM查询参数中间件
// 优化参数解析和验证逻辑
func BindQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		var q model.ResourceQuery
		if err := c.ShouldBindQuery(&q); err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Schemas:  model.ErrorSchema.String(),
				Detail:   "Invalid query parameters: " + err.Error(),
				Status:   http.StatusBadRequest,
				ScimType: "invalidValue",
			})
			c.Abort()
			return
		}

		// 规范化属性参数：去除空格并统一为小写
		if q.Attributes != "" {
			q.Attributes = normalizeAttributeParam(q.Attributes)
		}
		if q.ExcludedAttributes != "" {
			q.ExcludedAttributes = normalizeAttributeParam(q.ExcludedAttributes)
		}

		// 验证查询参数
		if err := q.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Schemas:  model.ErrorSchema.String(),
				Detail:   "Invalid query parameters: " + err.Error(),
				Status:   http.StatusBadRequest,
				ScimType: "invalidValue",
			})
			c.Abort()
			return
		}

		// 验证属性格式（在数据查询之前完成验证）
		if q.Attributes != "" {
			if err := util.ValidateAttributeFormat(q.Attributes); err != nil {
				c.JSON(http.StatusBadRequest, model.ErrorResponse{
					Schemas:  model.ErrorSchema.String(),
					Detail:   "Invalid attributes format: " + err.Error(),
					Status:   http.StatusBadRequest,
					ScimType: "invalidSyntax",
				})
				c.Abort()
				return
			}
		}
		if q.ExcludedAttributes != "" {
			if err := util.ValidateAttributeFormat(q.ExcludedAttributes); err != nil {
				c.JSON(http.StatusBadRequest, model.ErrorResponse{
					Schemas:  model.ErrorSchema.String(),
					Detail:   "Invalid excludedAttributes format: " + err.Error(),
					Status:   http.StatusBadRequest,
					ScimType: "invalidSyntax",
				})
				c.Abort()
				return
			}
		}

		c.Set("scim_query", &q)
		c.Next()
	}
}

// normalizeAttributeParam 规范化属性参数字符串
// 去除多余空格并统一为小写，提升后续匹配性能
func normalizeAttributeParam(param string) string {
	// 快速路径：空字符串直接返回
	if param == "" {
		return ""
	}

	// 分割、清理、重组
	parts := strings.Split(param, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, strings.ToLower(trimmed))
		}
	}
	return strings.Join(result, ",")
}

// Pagination 分页参数校验中间件（限制最大条数）
// 添加性能监控和参数边界检查
func Pagination(defaultCount, maxCount int) gin.HandlerFunc {
	// 预计算边界值，避免每次请求都进行计算
	if defaultCount <= 0 {
		defaultCount = 20
	}
	if maxCount <= 0 || maxCount > 1000 {
		maxCount = 100
	}

	return func(c *gin.Context) {
		q, ok := c.Get("scim_query")
		if !ok {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{
				Schemas:  model.ErrorSchema.String(),
				Detail:   "Query parameters not found",
				Status:   http.StatusInternalServerError,
				ScimType: "internalError",
			})
			c.Abort()
			return
		}
		query := q.(*model.ResourceQuery)

		// 校验StartIndex（SCIM标准从1开始）
		if query.StartIndex < 1 {
			query.StartIndex = 1
		}

		// 校验Count，应用默认值和最大值限制
		if query.Count <= 0 {
			query.Count = defaultCount
		} else if query.Count > maxCount {
			query.Count = maxCount
		}

		c.Set("scim_query", query)
		c.Next()
	}
}

// PerformanceMonitor 性能监控中间件
// 记录请求处理时间和响应状态，用于性能分析
func PerformanceMonitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		// 计算处理时间
		duration := time.Since(start)

		// 构建日志信息
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		// 如果有查询参数，追加到路径
		if raw != "" {
			path = path + "?" + raw
		}

		// 根据状态码选择日志级别
		if statusCode >= 500 {
			// 服务器错误，记录警告
			gin.DefaultErrorWriter.Write([]byte(
				formatLog(method, path, statusCode, duration, clientIP, true),
			))
		} else if duration > 100*time.Millisecond {
			// 慢请求，记录警告
			gin.DefaultErrorWriter.Write([]byte(
				formatLog(method, path, statusCode, duration, clientIP, true),
			))
		}

		// 始终记录到标准输出（生产环境可配置）
		gin.DefaultWriter.Write([]byte(
			formatLog(method, path, statusCode, duration, clientIP, false),
		))
	}
}

// formatLog 格式化日志输出
func formatLog(method, path string, status int, duration time.Duration, clientIP string, isWarning bool) string {
	var statusColor string
	switch {
	case status >= 200 && status < 300:
		statusColor = "\033[32m" // 绿色
	case status >= 300 && status < 400:
		statusColor = "\033[33m" // 黄色
	case status >= 400 && status < 500:
		statusColor = "\033[31m" // 红色
	default:
		statusColor = "\033[35m" // 紫色
	}
	resetColor := "\033[0m"

	warningPrefix := ""
	if isWarning {
		warningPrefix = "[SLOW] "
	}

	return warningPrefix + statusColor + method + resetColor +
		" | " + statusColor + padRight(status, 3) + resetColor +
		" | " + formatDuration(duration) +
		" | " + clientIP +
		" | " + path + "\n"
}

// padRight 右对齐数字
func padRight(num, width int) string {
	s := ""
	for i := 0; i < width-len(toString(num)); i++ {
		s += " "
	}
	return s + toString(num)
}

// toString 将整数转换为字符串
func toString(num int) string {
	if num == 0 {
		return "0"
	}
	var result []byte
	for num > 0 {
		result = append([]byte{byte('0' + num%10)}, result...)
		num /= 10
	}
	return string(result)
}

// formatDuration 格式化持续时间
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return "0μs"
	} else if d < time.Millisecond {
		return d.Round(time.Microsecond).String()
	} else if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(time.Millisecond).String()
}

// ErrorHandler 通用错误处理函数
// 统一错误响应格式，简化代码
func ErrorHandler(c *gin.Context, err error, status int, scimType string) {
	c.JSON(status, model.ErrorResponse{
		Schemas:  model.ErrorSchema.String(),
		Detail:   err.Error(),
		Status:   status,
		ScimType: scimType,
	})
}

// Recovery 自定义恢复中间件
// 捕获panic并返回友好的错误信息
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录panic信息
				gin.DefaultErrorWriter.Write([]byte(
					"[PANIC] " + c.Request.Method + " " + c.Request.URL.Path +
						" - " + fmt.Sprintf("%v", err) + "\n",
				))

				c.JSON(http.StatusInternalServerError, model.ErrorResponse{
					Schemas:  model.ErrorSchema.String(),
					Detail:   "Internal server error",
					Status:   http.StatusInternalServerError,
					ScimType: "internalError",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
