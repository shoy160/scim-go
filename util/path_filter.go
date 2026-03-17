package util

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// 路径过滤相关错误定义
var (
	ErrInvalidPathFilter     = errors.New("invalid path filter expression")
	ErrNoMatchingItems       = errors.New("no matching items found")
	ErrInvalidFilterSyntax   = errors.New("invalid filter syntax")
	ErrUnsupportedOperator   = errors.New("unsupported filter operator")
	ErrInvalidAttributeValue = errors.New("invalid attribute value")
)

// PathFilter 路径过滤器，用于 PATCH 操作中的路径过滤
type PathFilter struct {
	AttributeName string   // 属性名称
	Operator      Operator // 比较操作符
	Value         string   // 比较值
	IsString      bool     // 值是否为字符串类型
}

// ParsedPath 解析后的路径结构
type ParsedPath struct {
	AttributeName string      // 属性名称
	Filter        *PathFilter // 过滤条件（可选）
	SubPath       string      // 子路径（可选）
}

// ParsePathWithFilter 解析带有过滤条件的路径
// 支持格式：attributeName[filterExpression] 或 attributeName[filterExpression].subPath
// 示例：phoneNumbers[type eq "mobile"], addresses[type eq "work"].streetAddress
func ParsePathWithFilter(path string) (*ParsedPath, error) {
	if path == "" {
		return nil, ErrInvalidPathFilter
	}

	// 匹配带有过滤器的路径：attributeName[filterExpression] 或 attributeName[filterExpression].subPath
	re := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\[([^\]]+)\](?:\.(.*))?$`)
	matches := re.FindStringSubmatch(path)

	if len(matches) == 0 {
		// 没有过滤器，解析简单路径
		parts := strings.SplitN(path, ".", 2)
		result := &ParsedPath{
			AttributeName: parts[0],
		}
		if len(parts) > 1 {
			result.SubPath = parts[1]
		}
		return result, nil
	}

	// 提取各部分
	attributeName := matches[1]
	filterExpr := matches[2]
	subPath := ""
	if len(matches) > 3 {
		subPath = matches[3]
	}

	// 解析过滤表达式
	filter, err := parseFilterExpression(filterExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse filter expression: %w", err)
	}

	return &ParsedPath{
		AttributeName: attributeName,
		Filter:        filter,
		SubPath:       subPath,
	}, nil
}

// parseFilterExpression 解析过滤表达式
// 支持格式：attribute op value，如 type eq "mobile"
func parseFilterExpression(expr string) (*PathFilter, error) {
	expr = strings.TrimSpace(expr)

	// 首先检查 pr 操作符（一元操作符，不需要值）
	prOp := " pr"
	if strings.HasSuffix(strings.ToLower(expr), prOp) {
		attrName := strings.TrimSpace(expr[:len(expr)-len(prOp)])
		if attrName != "" {
			return &PathFilter{
				AttributeName: attrName,
				Operator:      OpPr,
				Value:         "",
				IsString:      false,
			}, nil
		}
	}

	// 检查其他二元操作符
	ops := AllComparisonOperators()
	for _, op := range ops {
		if op == OpPr {
			continue // pr 已在上面处理
		}

		opStr := " " + string(op) + " "
		if strings.Contains(strings.ToLower(expr), opStr) {
			parts := strings.SplitN(expr, opStr, 2)
			if len(parts) != 2 {
				continue
			}

			attrName := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			isString := false
			// 检查值是否被引号包围
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				isString = true
				value = value[1 : len(value)-1]
			}

			return &PathFilter{
				AttributeName: attrName,
				Operator:      op,
				Value:         value,
				IsString:      isString,
			}, nil
		}
	}

	return nil, ErrInvalidFilterSyntax
}

// MatchPathFilter 检查单个项是否匹配路径过滤条件
// 使用统一的 CompareValues 函数进行比较
func MatchPathFilter(item map[string]interface{}, filter *PathFilter) (bool, error) {
	if filter == nil {
		return true, nil
	}

	value, exists := item[filter.AttributeName]
	if !exists {
		if filter.Operator == OpPr {
			return false, nil
		}
		return false, nil
	}

	strValue := FormatValue(value)
	return CompareValues(strValue, filter.Value, filter.Operator, nil)
}

// FindMatchingIndices 查找数组中所有匹配过滤条件的索引
func FindMatchingIndices(items []map[string]interface{}, filter *PathFilter) ([]int, error) {
	if filter == nil {
		// 没有过滤器，返回所有索引
		indices := make([]int, len(items))
		for i := range items {
			indices[i] = i
		}
		return indices, nil
	}

	var matchingIndices []int
	for i, item := range items {
		matches, err := MatchPathFilter(item, filter)
		if err != nil {
			return nil, err
		}
		if matches {
			matchingIndices = append(matchingIndices, i)
		}
	}

	return matchingIndices, nil
}

// ValidatePathFilter 验证路径过滤表达式是否有效
func ValidatePathFilter(path string) error {
	_, err := ParsePathWithFilter(path)
	return err
}

// PathFilterToFilterNode 将路径过滤器转换为通用的 FilterNode
// 用于复用通用的过滤匹配逻辑
func PathFilterToFilterNode(filter *PathFilter) *FilterNode {
	if filter == nil {
		return nil
	}
	return &FilterNode{
		Op:    filter.Operator,
		Attr:  filter.AttributeName,
		Value: filter.Value,
	}
}
