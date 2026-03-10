package api

import (
	"scim-go/util"

	"github.com/gin-gonic/gin"
)

// GetRequestProtocol 获取请求协议
func GetRequestProtocol(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

// ValidateFilter 验证过滤器语法
func ValidateFilter(c *gin.Context, filter string, errorHandler func(*gin.Context, error, int, string)) error {
	if filter == "" {
		return nil
	}

	if err := util.ValidateFilter(filter); err != nil {
		return err
	}

	return nil
}
