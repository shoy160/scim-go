package util

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// FilterNode 过滤器抽象语法树节点
type FilterNode struct {
	Op       Operator      // 操作符
	Attr     string        // 属性路径，如 userName/name.givenName
	Value    string        // 比较值
	Children []*FilterNode // 子节点（用于 and/or/not）
}

// filterCache 过滤器解析结果缓存
var filterCache = &sync.Map{}

// ParseFilter 解析 SCIM 过滤表达式
func ParseFilter(filter string) (*FilterNode, error) {
	if strings.TrimSpace(filter) == "" {
		return nil, nil
	}

	cacheKey := strings.TrimSpace(strings.ToLower(filter))

	if cached, ok := filterCache.Load(cacheKey); ok {
		return cached.(*FilterNode), nil
	}

	parser := &filterParser{input: filter, pos: 0}
	node, err := parser.parseExpression()
	if err != nil {
		return nil, err
	}

	filterCache.Store(cacheKey, node)
	return node, nil
}

// ClearFilterCache 清空过滤器缓存
func ClearFilterCache() {
	filterCache = &sync.Map{}
}

// filterParser 过滤器解析器
type filterParser struct {
	input string
	pos   int
}

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
		left = &FilterNode{Op: OpOr, Children: []*FilterNode{left, right}}
	}

	return left, nil
}

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
		left = &FilterNode{Op: OpAnd, Children: []*FilterNode{left, right}}
	}

	return left, nil
}

func (p *filterParser) parseNot() (*FilterNode, error) {
	if p.peekKeyword("not") {
		p.consumeKeyword("not")
		child, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &FilterNode{Op: OpNot, Children: []*FilterNode{child}}, nil
	}
	return p.parsePrimary()
}

func (p *filterParser) parsePrimary() (*FilterNode, error) {
	p.skipWhitespace()

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

	attr, err := p.parseAttribute()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()

	ops := AllComparisonOperators()
	for _, op := range ops {
		if p.peekKeyword(string(op)) {
			p.consumeKeyword(string(op))
			if op == OpPr {
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

func (p *filterParser) parseValue() (string, error) {
	p.skipWhitespace()

	if p.peek() == '"' {
		return p.parseQuotedString()
	}

	start := p.pos
	for p.pos < len(p.input) && !isWhitespace(p.peek()) && p.peek() != ')' {
		p.consume()
	}

	if start == p.pos {
		return "", errors.New("expected value")
	}

	return p.input[start:p.pos], nil
}

func (p *filterParser) parseQuotedString() (string, error) {
	if p.peek() != '"' {
		return "", errors.New("expected opening quote")
	}
	p.consume()

	var result strings.Builder
	result.Grow(32)

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
	p.consume()

	return result.String(), nil
}

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

// ValidateFilter 验证过滤器语法是否有效
func ValidateFilter(filter string) error {
	_, err := ParseFilter(filter)
	return err
}

// MatchFilter 评估过滤器是否匹配对象
func MatchFilter(node *FilterNode, obj map[string]interface{}) (bool, error) {
	if node == nil {
		return true, nil
	}

	switch node.Op {
	case OpAnd:
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
	case OpOr:
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
	case OpNot:
		match, err := MatchFilter(node.Children[0], obj)
		if err != nil {
			return false, err
		}
		return !match, nil
	case OpEq, OpNe, OpCo, OpSw, OpEw, OpPr, OpGt, OpGe, OpLt, OpLe:
		return matchComparison(node, obj)
	default:
		return false, fmt.Errorf("unsupported operator: %s", node.Op)
	}
}

// matchComparison 执行比较操作
func matchComparison(node *FilterNode, obj map[string]interface{}) (bool, error) {
	actual := getValueByPath(obj, node.Attr)
	if actual == nil {
		if node.Op == OpPr {
			return false, nil
		}
		return node.Value == "", nil
	}

	actualStr := FormatValue(actual)
	return CompareValues(actualStr, node.Value, node.Op, nil)
}

// getValueByPath 从 map 中获取嵌套值
func getValueByPath(obj map[string]interface{}, path string) interface{} {
	if !strings.Contains(path, ".") && !strings.Contains(path, "[") {
		return obj[path]
	}

	parts := strings.Split(path, ".")
	current := interface{}(obj)

	for _, part := range parts {
		if current == nil {
			return nil
		}

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

// FilterToSQL 将过滤器转换为 SQL WHERE 子句
func FilterToSQL(node *FilterNode, columnMapping map[string]string) (string, []interface{}, error) {
	if node == nil {
		return "1=1", nil, nil
	}

	switch node.Op {
	case OpAnd:
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
	case OpOr:
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
	case OpNot:
		cond, args, err := FilterToSQL(node.Children[0], columnMapping)
		if err != nil {
			return "", nil, err
		}
		return "NOT (" + cond + ")", args, nil
	case OpEq:
		col := getColumn(node.Attr, columnMapping)
		return col + " = ?", []interface{}{node.Value}, nil
	case OpNe:
		col := getColumn(node.Attr, columnMapping)
		return col + " != ?", []interface{}{node.Value}, nil
	case OpCo:
		col := getColumn(node.Attr, columnMapping)
		return col + " LIKE ?", []interface{}{"%" + node.Value + "%"}, nil
	case OpSw:
		col := getColumn(node.Attr, columnMapping)
		return col + " LIKE ?", []interface{}{node.Value + "%"}, nil
	case OpEw:
		col := getColumn(node.Attr, columnMapping)
		return col + " LIKE ?", []interface{}{"%" + node.Value}, nil
	case OpPr:
		col := getColumn(node.Attr, columnMapping)
		return col + " IS NOT NULL AND " + col + " != ''", nil, nil
	case OpGt:
		col := getColumn(node.Attr, columnMapping)
		return col + " > ?", []interface{}{node.Value}, nil
	case OpGe:
		col := getColumn(node.Attr, columnMapping)
		return col + " >= ?", []interface{}{node.Value}, nil
	case OpLt:
		col := getColumn(node.Attr, columnMapping)
		return col + " < ?", []interface{}{node.Value}, nil
	case OpLe:
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
	return toSnakeCase(attr)
}

// toSnakeCase 将驼峰命名转换为蛇形命名
var (
	camelCaseRe1 = regexp.MustCompile("([a-z0-9])([A-Z])")
	camelCaseRe2 = regexp.MustCompile("([A-Z]+)([A-Z][a-z])")
)

func toSnakeCase(s string) string {
	s = camelCaseRe2.ReplaceAllString(s, "${1}_${2}")
	s = camelCaseRe1.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(s)
}
