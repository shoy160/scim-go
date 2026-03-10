package model

import (
	"strings"
)

// @Description 自定义资源类型定义
// @Produce json
type CustomResourceType struct {
	ID               string            `json:"id" gorm:"primaryKey"`
	Schemas          []string          `json:"schemas" gorm:"type:json;serializer:json"`
	Name             string            `json:"name" gorm:"not null;uniqueIndex"`
	Endpoint         string            `json:"endpoint" gorm:"not null;uniqueIndex"`
	Description      string            `json:"description,omitempty"`
	Schema           string            `json:"schema" gorm:"not null"` // 自定义资源的 schema URI
	SchemaExtensions []SchemaExtension `json:"schemaExtensions,omitempty" gorm:"-"`
	Meta             *Meta             `json:"meta,omitempty" gorm:"-"`
	CreatedAt        string            `json:"createdAt,omitempty" gorm:"autoCreateTime"`
	UpdatedAt        string            `json:"updatedAt,omitempty" gorm:"autoUpdateTime"`
}

// @Description 自定义资源数据定义
// @Produce json
type CustomResource struct {
	ID           string         `json:"id" gorm:"primaryKey"`
	Schemas      []string       `json:"schemas" gorm:"type:json;serializer:json"`
	ResourceType string         `json:"resourceType" gorm:"not null;index"` // 关联到 CustomResourceType 的 ID
	ExternalID   string         `json:"externalId,omitempty" gorm:"index"`
	Attributes   map[string]any `json:"attributes" gorm:"type:jsonb"` // 存储自定义资源的属性
	Meta         *Meta          `json:"meta,omitempty" gorm:"-"`
	CreatedAt    string         `json:"createdAt,omitempty" gorm:"autoCreateTime"`
	UpdatedAt    string         `json:"updatedAt,omitempty" gorm:"autoUpdateTime"`
	Version      string         `json:"version,omitempty" gorm:"not null"`
}

// @Description 自定义资源引用
// 用于在自定义资源中引用其他 SCIM 资源（如 User、Group）
// @Produce json
type CustomResourceReference struct {
	Value   string `json:"value"`   // 引用的资源 ID
	Ref     string `json:"$ref"`    // 引用的资源 URI
	Display string `json:"display"` // 引用的资源显示名称
	Type    string `json:"type"`    // 引用的资源类型（如 User、Group 或自定义资源类型）
}

// @Description 自定义资源查询参数
// @Produce json
type CustomResourceQuery struct {
	ResourceType string
	Filter       string
	SortBy       string
	SortOrder    string
	StartIndex   int
	Count        int
}

// ValidateCustomResourceType 验证自定义资源类型
func ValidateCustomResourceType(crt *CustomResourceType) error {
	// 验证必需字段
	if crt.ID == "" {
		return ErrInvalidValue
	}
	if crt.Name == "" {
		return ErrInvalidValue
	}
	if crt.Endpoint == "" {
		return ErrInvalidValue
	}
	if crt.Schema == "" {
		return ErrInvalidValue
	}

	// 验证 Endpoint 格式
	if !strings.HasPrefix(crt.Endpoint, "/") {
		return ErrInvalidValue
	}

	// 验证 Schema 格式（简单验证，实际应根据 SCIM 2.0 规范进行更严格的验证）
	if !strings.HasPrefix(crt.Schema, "urn:") {
		return ErrInvalidValue
	}

	return nil
}

// ValidateCustomResource 验证自定义资源
func ValidateCustomResource(cr *CustomResource) error {
	// 验证必需字段
	if cr.ID == "" {
		return ErrInvalidValue
	}
	if cr.ResourceType == "" {
		return ErrInvalidValue
	}

	// 验证 Attributes 不为 nil
	if cr.Attributes == nil {
		cr.Attributes = make(map[string]any)
	}

	return nil
}
