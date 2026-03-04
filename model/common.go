package model

import (
	"errors"
	"fmt"
)

// 预定义错误
// 这些错误用于在存储层和业务逻辑层之间传递标准化的错误信息
var (
	// ErrNotFound 资源未找到错误
	ErrNotFound = errors.New("resource not found")
	// ErrInvalidValue 无效值错误
	ErrInvalidValue = errors.New("invalid value")
	// ErrUniqueness 唯一性约束错误
	ErrUniqueness = errors.New("uniqueness error")
	// ErrInternal 内部服务器错误
	ErrInternal = errors.New("internal server error")
	// ErrUnauthorized 未授权错误
	ErrUnauthorized = errors.New("unauthorized")
	// ErrBadRequest 错误请求错误
	ErrBadRequest = errors.New("bad request")
	// ErrUserAlreadyInGroup 用户已在组中错误
	// 用于在 AddUserToGroup 操作中检测重复添加用户的情况
	// 修复日期: 2025-03-03
	// 修复原因: 统一错误类型，使 API 层能够正确识别并返回 409 Conflict 状态码
	ErrUserAlreadyInGroup = errors.New("user already in group")
	// ErrUserNotInGroup 用户不在组中错误
	// 用于在 RemoveUserFromGroup 操作中检测用户不在组中的情况
	// 修复日期: 2025-03-03
	// 修复原因: 统一错误类型，使 API 层能够正确识别并返回 404 Not Found 状态码
	ErrUserNotInGroup = errors.New("user not in group")
)

// SCIMSchema 定义SCIM标准Schema常量
type SCIMSchema string

const (
	// PatchSchema SCIM Patch操作Schema
	PatchSchema SCIMSchema = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	// ErrorSchema SCIM错误响应Schema
	ErrorSchema SCIMSchema = "urn:ietf:params:scim:api:messages:2.0:Error"
	// ListSchema SCIM列表响应Schema
	ListSchema SCIMSchema = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	// UserSchema SCIM用户Schema
	UserSchema SCIMSchema = "urn:ietf:params:scim:schemas:core:2.0:User"
	// GroupSchema SCIM组Schema
	GroupSchema SCIMSchema = "urn:ietf:params:scim:schemas:core:2.0:Group"
)

// String 返回Schema的字符串表示
func (s SCIMSchema) String() string {
	return string(s)
}

// ListResponse SCIM标准列表响应
// 用于返回资源列表，符合RFC 7644规范
type ListResponse struct {
	// Schemas SCIM Schema标识符数组
	Schemas []string `json:"schemas"`
	// TotalResults 总结果数量
	TotalResults int `json:"totalResults"`
	// StartIndex 起始索引（从1开始）
	StartIndex int `json:"startIndex,omitempty"`
	// ItemsPerPage 每页项目数
	ItemsPerPage int `json:"itemsPerPage,omitempty"`
	// Resources 资源列表
	Resources interface{} `json:"Resources"`
}

// NewListResponse 创建新的列表响应
// 简化列表响应的创建过程
func NewListResponse(schemas []string, total, startIndex, itemsPerPage int, resources interface{}) *ListResponse {
	return &ListResponse{
		Schemas:      schemas,
		TotalResults: total,
		StartIndex:   startIndex,
		ItemsPerPage: itemsPerPage,
		Resources:    resources,
	}
}

// PatchRequest SCIM标准PATCH请求体（RFC 7644：JSON Patch）
type PatchRequest struct {
	// Schemas SCIM Schema标识符数组
	Schemas []string `json:"schemas" validate:"required"`
	// Operations 补丁操作列表
	Operations []PatchOperation `json:"Operations" validate:"required"`
}

// Validate 验证Patch请求的有效性
func (p *PatchRequest) Validate() error {
	if len(p.Schemas) == 0 {
		return errors.New("schemas is required")
	}
	if p.Schemas[0] != PatchSchema.String() {
		return fmt.Errorf("invalid schema: %s, expected: %s", p.Schemas[0], PatchSchema)
	}
	if len(p.Operations) == 0 {
		return errors.New("operations is required")
	}
	return nil
}

// PatchOperation PATCH操作项：add/remove/replace
type PatchOperation struct {
	// Op 操作类型，必须是 add/remove/replace 之一
	Op string `json:"op" validate:"required,oneof=add remove replace"`
	// Path 属性路径，如 name.givenName 或 emails[0].value
	Path string `json:"path,omitempty"`
	// Value 操作值
	Value any `json:"value,omitempty"`
}

// Validate 验证Patch操作的有效性
func (p *PatchOperation) Validate() error {
	validOps := map[string]bool{"add": true, "remove": true, "replace": true}
	if !validOps[p.Op] {
		return fmt.Errorf("invalid operation: %s", p.Op)
	}
	return nil
}

// ErrorResponse SCIM标准错误响应
type ErrorResponse struct {
	// Schemas SCIM错误Schema
	Schemas string `json:"schemas"`
	// Detail 错误详情
	Detail string `json:"detail"`
	// Status HTTP状态码
	Status int `json:"status"`
	// ScimType SCIM错误类型
	ScimType string `json:"scimType,omitempty"`
}

// NewErrorResponse 创建新的错误响应
func NewErrorResponse(detail string, status int, scimType string) *ErrorResponse {
	return &ErrorResponse{
		Schemas:  ErrorSchema.String(),
		Detail:   detail,
		Status:   status,
		ScimType: scimType,
	}
}

// ResourceQuery 资源查询参数
// 用于列表查询和搜索操作
type ResourceQuery struct {
	// Filter 过滤条件表达式
	Filter string `form:"filter"`
	// StartIndex 起始索引（从1开始）
	StartIndex int `form:"startIndex,default=1"`
	// Count 每页数量
	Count int `form:"count"`
	// SortBy 排序字段
	SortBy string `form:"sortBy"`
	// SortOrder 排序顺序：ascending/descending
	SortOrder string `form:"sortOrder,default=ascending"`
	// Attributes 要返回的属性列表
	Attributes string `form:"attributes"`
	// ExcludedAttributes 要排除的属性列表
	ExcludedAttributes string `form:"excludedAttributes"`
	// Cursor 游标（用于游标分页）
	Cursor string `form:"cursor"`
}

// Validate 验证查询参数的有效性
func (q *ResourceQuery) Validate() error {
	if q.StartIndex < 1 {
		q.StartIndex = 1
	}
	if q.Count < 0 {
		q.Count = 0
	}
	if q.SortOrder != "" && q.SortOrder != "ascending" && q.SortOrder != "descending" {
		return fmt.Errorf("invalid sortOrder: %s", q.SortOrder)
	}
	return nil
}

// ScimConfig SCIM服务配置
// 包含服务级别的配置参数
type ScimConfig struct {
	// DefaultSchema 默认Schema
	DefaultSchema string
	// GroupSchema 组Schema
	GroupSchema string
	// ErrorSchema 错误Schema
	ErrorSchema string
	// ListSchema 列表Schema
	ListSchema string
}

// NewScimConfig 创建默认的SCIM配置
func NewScimConfig() *ScimConfig {
	return &ScimConfig{
		DefaultSchema: UserSchema.String(),
		GroupSchema:   GroupSchema.String(),
		ErrorSchema:   ErrorSchema.String(),
		ListSchema:    ListSchema.String(),
	}
}
