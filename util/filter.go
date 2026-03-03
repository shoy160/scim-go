package util

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FilterNode 过滤器抽象语法树节点
type FilterNode struct {
	Op       string        // eq/ne/gt/ge/lt/le/co/sw/ew/pr/and/or/not
	Attr     string        // 属性路径，如 userName/name.givenName
	Value    string        // 比较值
	Children []*FilterNode // 子节点（用于 and/or/not）
}

// ParseFilter 解析 SCIM 过滤表达式
// 支持: eq, ne, gt, ge, lt, le, co(contains), sw(startsWith), ew(endsWith), pr(present)
// 支持逻辑操作: and, or, not
func ParseFilter(filter string) (*FilterNode, error) {
	if strings.TrimSpace(filter) == "" {
		return nil, nil
	}
	parser := &filterParser{input: filter, pos: 0}
	return parser.parseExpression()
}

// filterParser 过滤器解析器
type filterParser struct {
	input string
	pos   int
}

// 解析表达式（处理 or）
func (p *filterParser) parseExpression() (*FilterNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.peekKeyword("or") {
		p.consumeKeyword("or")
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &FilterNode{Op: "or", Children: []*FilterNode{left, right}}
	}

	return left, nil
}

// 解析 and
func (p *filterParser) parseAnd() (*FilterNode, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.peekKeyword("and") {
		p.consumeKeyword("and")
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &FilterNode{Op: "and", Children: []*FilterNode{left, right}}
	}

	return left, nil
}

// 解析 not
func (p *filterParser) parseNot() (*FilterNode, error) {
	if p.peekKeyword("not") {
		p.consumeKeyword("not")
		child, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &FilterNode{Op: "not", Children: []*FilterNode{child}}, nil
	}
	return p.parsePrimary()
}

// 解析基本表达式
func (p *filterParser) parsePrimary() (*FilterNode, error) {
	p.skipWhitespace()

	// 处理括号分组
	if p.peek() == '(' {
		p.consume()
		node, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.peek() != ')' {
			return nil, errors.New("expected closing parenthesis")
		}
		p.consume()
		return node, nil
	}

	// 解析属性路径
	attr, err := p.parseAttribute()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()

	// 解析操作符
	ops := []string{"eq", "ne", "gt", "ge", "lt", "le", "co", "sw", "ew", "pr"}
	for _, op := range ops {
		if p.peekKeyword(op) {
			p.consumeKeyword(op)
			if op == "pr" {
				return &FilterNode{Op: op, Attr: attr}, nil
			}
			p.skipWhitespace()
			value, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			return &FilterNode{Op: op, Attr: attr, Value: value}, nil
		}
	}

	return nil, fmt.Errorf("expected operator after attribute %s", attr)
}

// 解析属性路径
func (p *filterParser) parseAttribute() (string, error) {
	p.skipWhitespace()
	start := p.pos

	for p.pos < len(p.input) && (isAlphaNumByte(p.peek()) || p.peek() == '.' || p.peek() == '[' || p.peek() == ']') {
		p.consume()
	}

	if start == p.pos {
		return "", errors.New("expected attribute name")
	}

	return strings.TrimSpace(p.input[start:p.pos]), nil
}

// 解析值
func (p *filterParser) parseValue() (string, error) {
	p.skipWhitespace()

	// 字符串值（带引号）
	if p.peek() == '"' {
		return p.parseQuotedString()
	}

	// 布尔值或 null
	start := p.pos
	for p.pos < len(p.input) && !isWhitespace(p.peek()) && p.peek() != ')' {
		p.consume()
	}

	if start == p.pos {
		return "", errors.New("expected value")
	}

	return p.input[start:p.pos], nil
}

// 解析带引号的字符串
func (p *filterParser) parseQuotedString() (string, error) {
	if p.peek() != '"' {
		return "", errors.New("expected opening quote")
	}
	p.consume() // 跳过开头的 "

	var result strings.Builder
	for p.pos < len(p.input) && p.peek() != '"' {
		if p.peek() == '\\' && p.pos+1 < len(p.input) {
			p.consume()
			switch p.peek() {
			case '"', '\\':
				result.WriteByte(p.peek())
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			default:
				result.WriteByte(p.peek())
			}
			p.consume()
		} else {
			result.WriteByte(p.peek())
			p.consume()
		}
	}

	if p.peek() != '"' {
		return "", errors.New("expected closing quote")
	}
	p.consume() // 跳过后面的 "

	return result.String(), nil
}

// 辅助方法
func (p *filterParser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *filterParser) consume() {
	p.pos++
}

func (p *filterParser) skipWhitespace() {
	for p.pos < len(p.input) && isWhitespace(p.peek()) {
		p.consume()
	}
}

func (p *filterParser) peekKeyword(keyword string) bool {
	p.skipWhitespace()
	if p.pos+len(keyword) > len(p.input) {
		return false
	}
	return strings.EqualFold(p.input[p.pos:p.pos+len(keyword)], keyword) &&
		(p.pos+len(keyword) >= len(p.input) || !isAlphaNum(rune(p.input[p.pos+len(keyword)])))
}

func (p *filterParser) consumeKeyword(keyword string) {
	p.skipWhitespace()
	p.pos += len(keyword)
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isAlphaNum(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func isAlphaNumByte(c byte) bool {
	return isAlphaNum(rune(c))
}

// MatchFilter 评估过滤器是否匹配对象
func MatchFilter(node *FilterNode, obj map[string]interface{}) (bool, error) {
	if node == nil {
		return true, nil
	}

	switch node.Op {
	case "and":
		for _, child := range node.Children {
			match, err := MatchFilter(child, obj)
			if err != nil {
				return false, err
			}
			if !match {
				return false, nil
			}
		}
		return true, nil
	case "or":
		for _, child := range node.Children {
			match, err := MatchFilter(child, obj)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
			}
		}
		return false, nil
	case "not":
		match, err := MatchFilter(node.Children[0], obj)
		if err != nil {
			return false, err
		}
		return !match, nil
	case "eq":
		return compareEq(node.Attr, node.Value, obj)
	case "ne":
		match, err := compareEq(node.Attr, node.Value, obj)
		return !match, err
	case "co":
		return compareCo(node.Attr, node.Value, obj)
	case "sw":
		return compareSw(node.Attr, node.Value, obj)
	case "ew":
		return compareEw(node.Attr, node.Value, obj)
	case "pr":
		return comparePr(node.Attr, obj)
	case "gt", "ge", "lt", "le":
		return compareNumeric(node.Op, node.Attr, node.Value, obj)
	default:
		return false, fmt.Errorf("unsupported operator: %s", node.Op)
	}
}

// 比较函数
func compareEq(attr, value string, obj map[string]interface{}) (bool, error) {
	actual := getValueByPath(obj, attr)
	if actual == nil {
		return value == "", nil
	}
	return fmt.Sprintf("%v", actual) == value, nil
}

func compareCo(attr, value string, obj map[string]interface{}) (bool, error) {
	actual := getValueByPath(obj, attr)
	if actual == nil {
		return false, nil
	}
	return strings.Contains(fmt.Sprintf("%v", actual), value), nil
}

func compareSw(attr, value string, obj map[string]interface{}) (bool, error) {
	actual := getValueByPath(obj, attr)
	if actual == nil {
		return false, nil
	}
	return strings.HasPrefix(fmt.Sprintf("%v", actual), value), nil
}

func compareEw(attr, value string, obj map[string]interface{}) (bool, error) {
	actual := getValueByPath(obj, attr)
	if actual == nil {
		return false, nil
	}
	return strings.HasSuffix(fmt.Sprintf("%v", actual), value), nil
}

func comparePr(attr string, obj map[string]interface{}) (bool, error) {
	actual := getValueByPath(obj, attr)
	return actual != nil && fmt.Sprintf("%v", actual) != "", nil
}

func compareNumeric(op, attr, value string, obj map[string]interface{}) (bool, error) {
	actual := getValueByPath(obj, attr)
	if actual == nil {
		return false, nil
	}

	actualNum, err1 := toFloat64(actual)
	valueNum, err2 := strconv.ParseFloat(value, 64)

	if err1 != nil || err2 != nil {
		// 字符串比较
		actualStr := fmt.Sprintf("%v", actual)
		switch op {
		case "gt":
			return actualStr > value, nil
		case "ge":
			return actualStr >= value, nil
		case "lt":
			return actualStr < value, nil
		case "le":
			return actualStr <= value, nil
		}
	}

	switch op {
	case "gt":
		return actualNum > valueNum, nil
	case "ge":
		return actualNum >= valueNum, nil
	case "lt":
		return actualNum < valueNum, nil
	case "le":
		return actualNum <= valueNum, nil
	}
	return false, nil
}

func toFloat64(v interface{}) (float64, error) {
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
		return 0, errors.New("not a number")
	}
}

// getValueByPath 从 map 中获取嵌套值
func getValueByPath(obj map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := interface{}(obj)

	for _, part := range parts {
		if current == nil {
			return nil
		}

		// 处理数组索引，如 emails[0]
		if strings.Contains(part, "[") {
			idx := strings.Index(part, "[")
			fieldName := part[:idx]
			idxEnd := strings.Index(part, "]")
			if idxEnd == -1 {
				return nil
			}
			index, err := strconv.Atoi(part[idx+1 : idxEnd])
			if err != nil {
				return nil
			}

			m, ok := current.(map[string]interface{})
			if !ok {
				return nil
			}
			arr, ok := m[fieldName].([]interface{})
			if !ok || index >= len(arr) {
				return nil
			}
			current = arr[index]
		} else {
			m, ok := current.(map[string]interface{})
			if !ok {
				return nil
			}
			current = m[part]
		}
	}

	return current
}

// ValidateFilter 验证过滤器语法是否有效
func ValidateFilter(filter string) error {
	_, err := ParseFilter(filter)
	return err
}

// FilterToSQL 将过滤器转换为 SQL WHERE 子句（简化版）
func FilterToSQL(node *FilterNode, columnMapping map[string]string) (string, []interface{}, error) {
	if node == nil {
		return "1=1", nil, nil
	}

	switch node.Op {
	case "and":
		var conditions []string
		var args []interface{}
		for _, child := range node.Children {
			cond, childArgs, err := FilterToSQL(child, columnMapping)
			if err != nil {
				return "", nil, err
			}
			conditions = append(conditions, cond)
			args = append(args, childArgs...)
		}
		return "(" + strings.Join(conditions, " AND ") + ")", args, nil
	case "or":
		var conditions []string
		var args []interface{}
		for _, child := range node.Children {
			cond, childArgs, err := FilterToSQL(child, columnMapping)
			if err != nil {
				return "", nil, err
			}
			conditions = append(conditions, cond)
			args = append(args, childArgs...)
		}
		return "(" + strings.Join(conditions, " OR ") + ")", args, nil
	case "not":
		cond, args, err := FilterToSQL(node.Children[0], columnMapping)
		if err != nil {
			return "", nil, err
		}
		return "NOT (" + cond + ")", args, nil
	case "eq":
		col := getColumn(node.Attr, columnMapping)
		return col + " = ?", []interface{}{node.Value}, nil
	case "ne":
		col := getColumn(node.Attr, columnMapping)
		return col + " != ?", []interface{}{node.Value}, nil
	case "co":
		col := getColumn(node.Attr, columnMapping)
		return col + " LIKE ?", []interface{}{"%" + node.Value + "%"}, nil
	case "sw":
		col := getColumn(node.Attr, columnMapping)
		return col + " LIKE ?", []interface{}{node.Value + "%"}, nil
	case "ew":
		col := getColumn(node.Attr, columnMapping)
		return col + " LIKE ?", []interface{}{"%" + node.Value}, nil
	case "pr":
		col := getColumn(node.Attr, columnMapping)
		return col + " IS NOT NULL AND " + col + " != ''", nil, nil
	case "gt":
		col := getColumn(node.Attr, columnMapping)
		return col + " > ?", []interface{}{node.Value}, nil
	case "ge":
		col := getColumn(node.Attr, columnMapping)
		return col + " >= ?", []interface{}{node.Value}, nil
	case "lt":
		col := getColumn(node.Attr, columnMapping)
		return col + " < ?", []interface{}{node.Value}, nil
	case "le":
		col := getColumn(node.Attr, columnMapping)
		return col + " <= ?", []interface{}{node.Value}, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", node.Op)
	}
}

func getColumn(attr string, mapping map[string]string) string {
	if col, ok := mapping[attr]; ok {
		return col
	}
	// 默认转换为蛇形命名
	return toSnakeCase(attr)
}

func toSnakeCase(s string) string {
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	s = re.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(s)
}
