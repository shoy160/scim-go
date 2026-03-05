package util

import (
	"scim-go/model"
)

// NewListResponse 创建SCIM标准列表响应
func NewListResponse(schemas string, totalResults int, startIndex int, itemsPerPage int, resources []interface{}) model.ListResponse {
	return model.ListResponse{
		Schemas:      []string{schemas},
		TotalResults: totalResults,
		StartIndex:   startIndex,
		ItemsPerPage: itemsPerPage,
		Resources:    resources,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(schemas string, status int, scimType string, detail string) model.ErrorResponse {
	return model.ErrorResponse{
		Schemas:  schemas,
		Status:   status,
		ScimType: scimType,
		Detail:   detail,
	}
}
