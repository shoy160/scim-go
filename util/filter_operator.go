package util

import (
	"fmt"
	"strconv"
	"strings"
)

// Operator 定义 SCIM 过滤操作符类型
type Operator string

const (
	OpEq  Operator = "eq"  // 等于
	OpNe  Operator = "ne"  // 不等于
	OpGt  Operator = "gt"  // 大于
	OpGe  Operator = "ge"  // 大于等于
	OpLt  Operator = "lt"  // 小于
	OpLe  Operator = "le"  // 小于等于
	OpCo  Operator = "co"  // 包含
	OpSw  Operator = "sw"  // 以...开始
	OpEw  Operator = "ew"  // 以...结束
	OpPr  Operator = "pr"  // 存在
	OpAnd Operator = "and" // 逻辑与
	OpOr  Operator = "or"  // 逻辑或
	OpNot Operator = "not" // 逻辑非
)

// CompareOptions 包含比较操作的配置选项
type CompareOptions struct {
	CaseInsensitive bool // 是否忽略大小写
}

// DefaultCompareOptions 默认比较选项（忽略大小写）
var DefaultCompareOptions = &CompareOptions{
	CaseInsensitive: true,
}

// CompareValues 执行两个字符串值之间的比较操作
// 这是所有比较操作的统一入口，支持大小写敏感配置
func CompareValues(actual, expected string, op Operator, opts *CompareOptions) (bool, error) {
	if opts == nil {
		opts = DefaultCompareOptions
	}

	switch op {
	case OpEq:
		return compareEqual(actual, expected, opts.CaseInsensitive), nil
	case OpNe:
		return !compareEqual(actual, expected, opts.CaseInsensitive), nil
	case OpCo:
		return compareContains(actual, expected, opts.CaseInsensitive), nil
	case OpSw:
		return compareStartsWith(actual, expected, opts.CaseInsensitive), nil
	case OpEw:
		return compareEndsWith(actual, expected, opts.CaseInsensitive), nil
	case OpPr:
		return actual != "", nil
	case OpGt, OpGe, OpLt, OpLe:
		return compareOrdered(actual, expected, op)
	default:
		return false, fmt.Errorf("unsupported comparison operator: %s", op)
	}
}

// compareEqual 比较两个值是否相等
func compareEqual(actual, expected string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.EqualFold(actual, expected)
	}
	return actual == expected
}

// compareContains 检查 actual 是否包含 expected
func compareContains(actual, expected string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.Contains(strings.ToLower(actual), strings.ToLower(expected))
	}
	return strings.Contains(actual, expected)
}

// compareStartsWith 检查 actual 是否以 expected 开头
func compareStartsWith(actual, expected string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.HasPrefix(strings.ToLower(actual), strings.ToLower(expected))
	}
	return strings.HasPrefix(actual, expected)
}

// compareEndsWith 检查 actual 是否以 expected 结尾
func compareEndsWith(actual, expected string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.HasSuffix(strings.ToLower(actual), strings.ToLower(expected))
	}
	return strings.HasSuffix(actual, expected)
}

// compareOrdered 执行有序比较（大于、小于等）
// 首先尝试数值比较，如果失败则使用字符串比较
func compareOrdered(actual, expected string, op Operator) (bool, error) {
	actualNum, actualErr := strconv.ParseFloat(actual, 64)
	expectedNum, expectedErr := strconv.ParseFloat(expected, 64)

	if actualErr == nil && expectedErr == nil {
		return compareNumeric(actualNum, expectedNum, op), nil
	}

	return compareStringOrdered(actual, expected, op), nil
}

// compareNumeric 执行数值比较
func compareNumeric(actual, expected float64, op Operator) bool {
	switch op {
	case OpGt:
		return actual > expected
	case OpGe:
		return actual >= expected
	case OpLt:
		return actual < expected
	case OpLe:
		return actual <= expected
	default:
		return false
	}
}

// compareStringOrdered 执行字符串有序比较
func compareStringOrdered(actual, expected string, op Operator) bool {
	switch op {
	case OpGt:
		return actual > expected
	case OpGe:
		return actual >= expected
	case OpLt:
		return actual < expected
	case OpLe:
		return actual <= expected
	default:
		return false
	}
}

// ToFloat64 将各种类型转换为 float64
// 支持 int, int32, int64, float32, float64, string
func ToFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// FormatValue 将任意值格式化为字符串用于比较
func FormatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// IsValidOperator 检查操作符是否有效
func IsValidOperator(op string) bool {
	switch Operator(op) {
	case OpEq, OpNe, OpGt, OpGe, OpLt, OpLe, OpCo, OpSw, OpEw, OpPr:
		return true
	case OpAnd, OpOr, OpNot:
		return true
	default:
		return false
	}
}

// IsLogicalOperator 检查是否为逻辑操作符
func IsLogicalOperator(op Operator) bool {
	return op == OpAnd || op == OpOr || op == OpNot
}

// IsComparisonOperator 检查是否为比较操作符
func IsComparisonOperator(op Operator) bool {
	return op == OpEq || op == OpNe || op == OpGt || op == OpGe ||
		op == OpLt || op == OpLe || op == OpCo || op == OpSw || op == OpEw || op == OpPr
}

// AllComparisonOperators 返回所有比较操作符列表
func AllComparisonOperators() []Operator {
	return []Operator{OpEq, OpNe, OpGt, OpGe, OpLt, OpLe, OpCo, OpSw, OpEw, OpPr}
}
