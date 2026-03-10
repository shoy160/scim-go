package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"scim-go/model"
)

// PathProcessor 路径处理器接口
type PathProcessor interface {
	Process(path string, obj interface{}) error
}

// NestedPathHandler 嵌套路径处理器
type NestedPathHandler struct {
	SetValueFunc    func(current reflect.Value, value interface{}) error
	RemoveValueFunc func(current reflect.Value) error
}

// Process 处理路径
func (h *NestedPathHandler) Process(path string, obj interface{}) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("obj must be a non-nil pointer")
	}
	v = v.Elem()

	keys := parsePath(path)
	var finalField reflect.Value
	finalField = v

	for i, key := range keys {
		// 处理企业扩展属性的命名空间
		if key == string(model.EnterpriseUserSchema) {
			// 企业扩展属性是通过嵌入 EnterpriseUserExtension 结构体实现的
			finalField, _ = fieldByNameIgnoreCase(finalField, "EnterpriseUserExtension")
		} else if strings.Contains(key, "[") {
			// 处理数组下标：members[0]
			idxStr := key[strings.Index(key, "[")+1 : strings.Index(key, "]")]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return err
			}
			fieldName := key[:strings.Index(key, "[")]
			finalField, _ = fieldByNameIgnoreCase(finalField, fieldName)
			if !finalField.IsValid() {
				return fmt.Errorf("field %s not found", fieldName)
			}
			if idx >= finalField.Len() {
				return errors.New("index out of range")
			}
			finalField = finalField.Index(idx)
		} else {
			finalField, _ = fieldByNameIgnoreCase(finalField, key)
		}

		if !finalField.IsValid() {
			return fmt.Errorf("path %s not found", path)
		}

		// 如果是指针，解引用
		if finalField.Kind() == reflect.Ptr {
			if finalField.IsNil() {
				finalField.Set(reflect.New(finalField.Type().Elem()))
			}
			finalField = finalField.Elem()
		}

		// 最后一段才执行操作
		if i == len(keys)-1 {
			if h.SetValueFunc != nil {
				return h.SetValueFunc(finalField, nil)
			} else if h.RemoveValueFunc != nil {
				return h.RemoveValueFunc(finalField)
			}
		}
	}
	return nil
}

// StructConverter 结构体转换器
type StructConverter struct{}

// ToMap 将结构体转换为 map[string]interface{}
func (c *StructConverter) ToMap(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// FromMap 将 map[string]interface{} 转换为结构体
func (c *StructConverter) FromMap(m map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// AttributeValidator 属性验证器
type AttributeValidator struct{}

// ValidateEmail 验证邮箱格式
func (v *AttributeValidator) ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email cannot be empty")
	}
	// 简单的 email 格式验证
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format: %s", email)
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format: %s", email)
	}
	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email format: %s", email)
	}
	if !strings.Contains(parts[1], ".") {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

// ValidateRole 验证角色定义
func (v *AttributeValidator) ValidateRole(role string) error {
	if role == "" {
		return errors.New("role cannot be empty")
	}
	// 可以在这里添加更多的角色验证逻辑
	// 例如检查角色是否在预定义的角色列表中
	return nil
}

// ValidateAttribute 验证属性格式
func (v *AttributeValidator) ValidateAttribute(attr string) error {
	if attr == "" {
		return errors.New("attribute cannot be empty")
	}

	// 检查属性是否以点开头
	if strings.HasPrefix(attr, ".") {
		return fmt.Errorf("invalid attribute format: %s (cannot start with dot)", attr)
	}

	// 检查属性是否包含连续的点
	if strings.Contains(attr, "..") {
		return fmt.Errorf("invalid attribute format: %s (cannot contain consecutive dots)", attr)
	}

	// 检查属性是否包含非法字符（只允许字母、数字、下划线、点和星号）
	for _, ch := range attr {
		if !isAlphaNumeric(ch) && ch != '_' && ch != '.' && ch != '*' {
			return fmt.Errorf("invalid attribute format: %s (contains invalid character: %c)", attr, ch)
		}
	}

	// 检查通配符模式
	if strings.Contains(attr, "*") {
		// 通配符只能出现在 ".*" 形式中
		if !strings.HasSuffix(attr, ".*") {
			return fmt.Errorf("invalid attribute format: %s (wildcard can only be used as '.*' at the end)", attr)
		}
		// 检查通配符模式的前缀
		prefix := strings.TrimSuffix(attr, ".*")
		if prefix == "" {
			return fmt.Errorf("invalid attribute format: %s (wildcard must have a parent attribute)", attr)
		}
		// 验证前缀部分
		if err := validateAttributePart(prefix); err != nil {
			return fmt.Errorf("invalid attribute format: %s (%v)", attr, err)
		}
	} else {
		// 检查嵌套属性的每一部分
		parts := strings.Split(attr, ".")
		for _, part := range parts {
			if err := validateAttributePart(part); err != nil {
				return fmt.Errorf("invalid attribute format: %s (%v)", attr, err)
			}
		}
	}

	return nil
}

// ListProcessor 列表处理器
type ListProcessor struct{}

// Filter 过滤列表
func (p *ListProcessor) Filter(items interface{}, filter string, toMap func(interface{}) (map[string]interface{}, error)) (interface{}, error) {
	if filter == "" {
		return items, nil
	}

	node, err := ParseFilter(filter)
	if err != nil {
		return nil, err
	}

	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil, errors.New("items must be a slice")
	}

	// 预分配切片容量
	result := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		obj, err := toMap(item)
		if err != nil {
			return nil, err
		}

		match, err := MatchFilter(node, obj)
		if err != nil {
			return nil, err
		}

		if match {
			result = reflect.Append(result, v.Index(i))
		}
	}

	return result.Interface(), nil
}

// Paginate 分页列表
func (p *ListProcessor) Paginate(items interface{}, startIndex, count int) interface{} {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil
	}

	start := startIndex - 1
	if start < 0 {
		start = 0
	}
	if start >= v.Len() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}

	end := start + count
	if end > v.Len() {
		end = v.Len()
	}

	return v.Slice(start, end).Interface()
}

// Sort 排序列表
func (p *ListProcessor) Sort(items interface{}, sortBy, sortOrder string, compare func(i, j int) bool) {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return
	}

	less := func(i, j int) bool {
		if sortOrder == "descending" {
			return !compare(i, j)
		}
		return compare(i, j)
	}

	sort.Slice(items, less)
}
