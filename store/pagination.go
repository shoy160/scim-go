package store

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"scim-go/model"
	"strings"
	"time"
)

// CursorInfo 游标信息
type CursorInfo struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"created_at"`
}

// EncodeCursor 编码游标
func EncodeCursor(id string, createdAt time.Time) string {
	info := CursorInfo{
		ID:        id,
		CreatedAt: createdAt.Unix(),
	}
	data, _ := json.Marshal(info)
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeCursor 解码游标
func DecodeCursor(cursor string) (*CursorInfo, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	var info CursorInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}
	return &info, nil
}

// PaginationParams 分页参数
type PaginationParams struct {
	StartIndex int    // 起始索引（从1开始）
	Count      int    // 每页数量
	SortBy     string // 排序字段
	SortOrder  string // 排序方向（ascending/descending）
	Cursor     string // 游标（用于游标分页）
}

// PaginationResult 分页结果
type PaginationResult struct {
	TotalResults int64   // 总数量
	StartIndex   int     // 起始索引
	ItemsPerPage int     // 每页数量
	Cursor       string  // 下一页游标
	HasMore      bool    // 是否有更多数据
	Resources    []any   // 资源列表
}

// ApplyOffsetPagination 应用偏移量分页
func ApplyOffsetPagination(query interface{}, params PaginationParams) (offset, limit int) {
	if params.Count <= 0 {
		params.Count = 100
	}
	if params.StartIndex <= 0 {
		params.StartIndex = 1
	}
	offset = (params.StartIndex - 1) * params.Count
	limit = params.Count
	return offset, limit
}

// BuildSortClause 构建排序子句
func BuildSortClause(sortBy, sortOrder, defaultSort string) string {
	if sortBy == "" {
		sortBy = defaultSort
	}
	order := "ASC"
	if strings.ToLower(sortOrder) == "descending" {
		order = "DESC"
	}
	return fmt.Sprintf("%s %s", sortBy, order)
}

// CalculateHasMore 计算是否有更多数据
func CalculateHasMore(total int64, startIndex, count int) bool {
	return int64(startIndex+count) <= total
}

// NextCursor 生成下一页游标
func NextCursor(lastID string, lastCreatedAt time.Time) string {
	return EncodeCursor(lastID, lastCreatedAt)
}

// ParseResourceQuery 解析资源查询参数
func ParseResourceQuery(q *model.ResourceQuery) PaginationParams {
	return PaginationParams{
		StartIndex: q.StartIndex,
		Count:      q.Count,
		SortBy:     q.SortBy,
		SortOrder:  q.SortOrder,
		Cursor:     q.Cursor,
	}
}

// CursorPaginationConfig 游标分页配置
type CursorPaginationConfig struct {
	DefaultLimit    int
	MaxLimit        int
	DefaultSortBy   string
	DefaultSortOrder string
}

// DefaultCursorConfig 默认游标分页配置
var DefaultCursorConfig = CursorPaginationConfig{
	DefaultLimit:     100,
	MaxLimit:         500,
	DefaultSortBy:    "created_at",
	DefaultSortOrder: "descending",
}

// ValidateAndNormalizeCursorParams 验证并规范化游标分页参数
func ValidateAndNormalizeCursorParams(params *PaginationParams, config CursorPaginationConfig) {
	if params.Count <= 0 {
		params.Count = config.DefaultLimit
	}
	if params.Count > config.MaxLimit {
		params.Count = config.MaxLimit
	}
	if params.SortBy == "" {
		params.SortBy = config.DefaultSortBy
	}
	if params.SortOrder == "" {
		params.SortOrder = config.DefaultSortOrder
	}
}
