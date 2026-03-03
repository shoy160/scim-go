package model

import "errors"

// 预定义错误
var (
	ErrNotFound     = errors.New("resource not found")
	ErrInvalidValue = errors.New("invalid value")
	ErrUniqueness   = errors.New("uniqueness error")
	ErrInternal     = errors.New("internal server error")
)

// ListResponse SCIM标准列表响应
type ListResponse struct {
	Schemas      []string    `json:"schemas"`
	TotalResults int         `json:"totalResults"`
	StartIndex   int         `json:"startIndex,omitempty"`
	ItemsPerPage int         `json:"itemsPerPage,omitempty"`
	Resources    interface{} `json:"Resources"`
}

// PatchRequest SCIM标准PATCH请求体（RFC 7644：JSON Patch）
type PatchRequest struct {
	Schemas    []string         `json:"schemas" validate:"required"`
	Operations []PatchOperation `json:"Operations" validate:"required"`
}

// PatchOperation PATCH操作项：add/remove/replace
type PatchOperation struct {
	Op    string `json:"op" validate:"required,oneof=add remove replace"` // 操作类型
	Path  string `json:"path,omitempty"`                                  // 属性路径：如name.givenName/emails[0].value
	Value any    `json:"value,omitempty"`                                 // 操作值
}

// ErrorResponse SCIM标准错误响应
type ErrorResponse struct {
	Schemas  string `json:"schemas"`
	Detail   string `json:"detail"`
	Status   int    `json:"status"`
	ScimType string `json:"scimType,omitempty"`
}

// ResourceQuery 查询参数
type ResourceQuery struct {
	Filter             string `form:"filter"`
	StartIndex         int    `form:"startIndex,default=1"`
	Count              int    `form:"count"`
	SortBy             string `form:"sortBy"`
	SortOrder          string `form:"sortOrder,default=ascending"`
	Attributes         string `form:"attributes"`
	ExcludedAttributes string `form:"excludedAttributes"`
	Cursor             string `form:"cursor"`
}

// ScimConfig 【这里是你缺失的！已补上】
type ScimConfig struct {
	DefaultSchema string
	GroupSchema   string
	ErrorSchema   string
	ListSchema    string
}

const (
	PatchSchema = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	ErrorSchema = "urn:ietf:params:scim:api:messages:2.0:Error"
	ListSchema  = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	UserSchema  = "urn:ietf:params:scim:schemas:core:2.0:User"
	GroupSchema = "urn:ietf:params:scim:schemas:core:2.0:Group"
)
