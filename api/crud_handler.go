package api

import (
	"errors"
	"net/http"

	"scim-go/model"
	"scim-go/util"

	"github.com/gin-gonic/gin"
)

// CRUDHandler 通用 CRUD 处理器
// 用于处理 SCIM 资源的通用 CRUD 操作

type CRUDHandler interface {
	// 验证过滤器语法
	validateFilter(c *gin.Context, filter string) error
	// 获取请求协议
	getRequestProtocol(c *gin.Context) string
}

// ListRequest 列表请求参数
type ListRequest struct {
	Query          *model.ResourceQuery
	ValidateFilter func(filter string) error
	ListFunc       func(q *model.ResourceQuery) ([]interface{}, int64, error)
	ProcessFunc    func(resources []interface{}, q *model.ResourceQuery, host, proto string) []interface{}
	Schema         string
}

// GetRequest 获取单个资源请求参数
type GetRequest struct {
	ID            string
	Query         *model.ResourceQuery
	GetFunc       func(id string, preload bool) (interface{}, error)
	ProcessFunc   func(resource interface{}, q *model.ResourceQuery, host, proto string) error
	AttributeName string
	Schema        string
}

// CreateRequest 创建资源请求参数
type CreateRequest struct {
	Resource     interface{}
	ValidateFunc func(resource interface{}) error
	CreateFunc   func(resource interface{}) error
	ProcessFunc  func(resource interface{}, host, proto string) error
	Schema       string
}

// UpdateRequest 更新资源请求参数
type UpdateRequest struct {
	ID           string
	Resource     interface{}
	ValidateFunc func(resource interface{}) error
	UpdateFunc   func(resource interface{}) error
	GetFunc      func(id string) (interface{}, error)
	ProcessFunc  func(resource interface{}, host, proto string) error
	Schema       string
}

// PatchRequest 补丁更新资源请求参数
type PatchRequest struct {
	ID          string
	Ops         []model.PatchOperation
	GetFunc     func(id string) (interface{}, error)
	PatchFunc   func(id string, ops []model.PatchOperation) error
	ProcessFunc func(resource interface{}, host, proto string) error
	Schema      string
}

// DeleteRequest 删除资源请求参数
type DeleteRequest struct {
	ID         string
	DeleteFunc func(id string) error
}

// HandleList 处理列表请求
func HandleList(c *gin.Context, req ListRequest) {
	// 验证过滤器语法
	if req.ValidateFilter != nil {
		if err := req.ValidateFilter(req.Query.Filter); err != nil {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidFilter")
			return
		}
	}

	// 获取资源列表
	resources, total, err := req.ListFunc(req.Query)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取当前请求的协议和主机
	proto := GetRequestProtocol(c)
	host := c.Request.Host

	// 处理资源列表并应用属性选择
	processedResources := req.ProcessFunc(resources, req.Query, host, proto)
	if processedResources == nil {
		ErrorHandler(c, errors.New("failed to process resource list"), http.StatusInternalServerError, "internalError")
		return
	}

	// 构造SCIM标准列表响应
	response := util.NewListResponse(req.Schema, int(total), req.Query.StartIndex, req.Query.Count, processedResources)
	c.JSON(http.StatusOK, response)
}

// HandleGet 处理获取单个资源请求
func HandleGet(c *gin.Context, req GetRequest) {
	// 获取资源
	resource, err := req.GetFunc(req.ID, false)
	if err != nil {
		ErrorHandler(c, err, http.StatusNotFound, "notFound")
		return
	}

	// 处理资源
	if req.ProcessFunc != nil {
		if err := req.ProcessFunc(resource, req.Query, c.Request.Host, GetRequestProtocol(c)); err != nil {
			ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
			return
		}
	}

	// 应用属性选择
	if req.Query.Attributes != "" || req.Query.ExcludedAttributes != "" {
		filtered, err := util.ApplyAttributeSelectionWithSpecialRules(resource, req.Query.Attributes, req.Query.ExcludedAttributes, req.AttributeName)
		if err != nil {
			ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
			return
		}
		c.JSON(http.StatusOK, filtered)
		return
	}

	c.JSON(http.StatusOK, resource)
}

// HandleCreate 处理创建资源请求
func HandleCreate(c *gin.Context, req CreateRequest) {
	// 验证资源
	if req.ValidateFunc != nil {
		if err := req.ValidateFunc(req.Resource); err != nil {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
			return
		}
	}

	// 创建资源
	if err := req.CreateFunc(req.Resource); err != nil {
		if errors.Is(err, model.ErrUniqueness) {
			ErrorHandler(c, err, http.StatusConflict, "uniqueness")
			return
		}
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, errors.New("invalid reference"), http.StatusBadRequest, "invalidValue")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 处理资源
	if req.ProcessFunc != nil {
		if err := req.ProcessFunc(req.Resource, c.Request.Host, GetRequestProtocol(c)); err != nil {
			ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
			return
		}
	}

	c.JSON(http.StatusCreated, req.Resource)
}

// HandleUpdate 处理更新资源请求
func HandleUpdate(c *gin.Context, req UpdateRequest) {
	// 验证资源
	if req.ValidateFunc != nil {
		if err := req.ValidateFunc(req.Resource); err != nil {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
			return
		}
	}

	// 更新资源
	if err := req.UpdateFunc(req.Resource); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取更新后的资源
	updatedResource, err := req.GetFunc(req.ID)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 处理资源
	if req.ProcessFunc != nil {
		if err := req.ProcessFunc(updatedResource, c.Request.Host, GetRequestProtocol(c)); err != nil {
			ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
			return
		}
	}

	c.JSON(http.StatusOK, updatedResource)
}

// HandlePatch 处理补丁更新资源请求
func HandlePatch(c *gin.Context, req PatchRequest) {
	// 获取资源
	_, err := req.GetFunc(req.ID)
	if err != nil {
		ErrorHandler(c, err, http.StatusNotFound, "notFound")
		return
	}

	// 应用补丁操作
	if err := req.PatchFunc(req.ID, req.Ops); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 获取更新后的资源
	updatedResource, err := req.GetFunc(req.ID)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 处理资源
	if req.ProcessFunc != nil {
		if err := req.ProcessFunc(updatedResource, c.Request.Host, GetRequestProtocol(c)); err != nil {
			ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
			return
		}
	}

	c.JSON(http.StatusOK, updatedResource)
}

// HandleDelete 处理删除资源请求
func HandleDelete(c *gin.Context, req DeleteRequest) {
	// 删除资源
	if err := req.DeleteFunc(req.ID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	c.Status(http.StatusNoContent)
}
