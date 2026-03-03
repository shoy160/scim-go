package api

import (
	"net/http"
	"scim-go/model"

	"github.com/gin-gonic/gin"
)

// Auth SCIM Bearer Token认证中间件
func Auth(validToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "Bearer "+validToken {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Schemas:  model.ErrorSchema,
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
func BindQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		var q model.ResourceQuery
		if err := c.ShouldBindQuery(&q); err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Schemas:  model.ErrorSchema,
				Detail:   "Invalid query parameters: " + err.Error(),
				Status:   http.StatusBadRequest,
				ScimType: "invalidValue",
			})
			c.Abort()
			return
		}
		c.Set("scim_query", &q)
		c.Next()
	}
}

// Pagination 分页参数校验中间件（限制最大条数）
func Pagination(defaultCount, maxCount int) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, ok := c.Get("scim_query")
		if !ok {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{
				Schemas:  model.ErrorSchema,
				Detail:   "Query parameters not found",
				Status:   http.StatusInternalServerError,
				ScimType: "internalError",
			})
			c.Abort()
			return
		}
		query := q.(*model.ResourceQuery)

		// 校验StartIndex
		if query.StartIndex < 1 {
			query.StartIndex = 1
		}
		// 校验Count
		if query.Count <= 0 {
			query.Count = defaultCount
		}
		if query.Count > maxCount {
			query.Count = maxCount
		}

		c.Set("scim_query", query)
		c.Next()
	}
}

// ErrorHandler 通用错误处理函数（简化代码）
func ErrorHandler(c *gin.Context, err error, status int, scimType string) {
	c.JSON(status, model.ErrorResponse{
		Schemas:  model.ErrorSchema,
		Detail:   err.Error(),
		Status:   status,
		ScimType: scimType,
	})
}
