package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ApplyAttributeSelection 应用属性选择到资源对象
// attributes: 只返回指定的属性
// excludedAttributes: 排除指定的属性
func ApplyAttributeSelection(obj interface{}, attributes, excludedAttributes string) (map[string]interface{}, error) {
	// 转换为 map
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// 处理 attributes 参数
	if attributes != "" {
		attrs := ParseAttributeList(attributes)
		result = filterAttributes(result, attrs, true)
	}

	// 处理 excludedAttributes 参数
	if excludedAttributes != "" {
		attrs := ParseAttributeList(excludedAttributes)
		result = filterAttributes(result, attrs, false)
	}

	return result, nil
}

// ApplyAttributeSelectionWithSpecialRules 应用属性选择到资源对象，支持特殊规则
// 当 attributes 只包含 "members" 时，返回所有 Group 字段
// 当 attributes 只包含 "groups" 时，返回所有 User 字段
// 当 attributes 包含 "members" 或 "groups" 以及其他属性时，只返回指定属性
func ApplyAttributeSelectionWithSpecialRules(obj interface{}, attributes, excludedAttributes string, specialAttr string) (map[string]interface{}, error) {
	// 转换为 map
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// 处理 attributes 参数
	if attributes != "" {
		attrs := ParseAttributeList(attributes)

		// 检查是否只包含特殊属性（members 或 groups）或其通配符模式
		if len(attrs) == 1 && (attrs[0] == specialAttr || attrs[0] == specialAttr+".*") {
			// 只包含特殊属性或其通配符模式，返回所有字段
			// 确保特殊属性总是被包含（即使它是空的）
			if _, ok := result[specialAttr]; !ok {
				result[specialAttr] = []interface{}{}
			}
		} else {
			// 包含多个属性或其他属性，应用正常过滤
			// 确保 meta 属性总是被包含（SCIM 2.0 规范要求）
			metaIncluded := false
			for _, attr := range attrs {
				if strings.EqualFold(attr, "meta") || strings.HasPrefix(strings.ToLower(attr), "meta.") {
					metaIncluded = true
					break
				}
			}
			if !metaIncluded {
				attrs = append(attrs, "meta")
			}
			result = filterAttributes(result, attrs, true)
		}
	}

	// 处理 excludedAttributes 参数
	if excludedAttributes != "" {
		attrs := ParseAttributeList(excludedAttributes)
		result = filterAttributes(result, attrs, false)
	}

	return result, nil
}

// ParseAttributeList 解析属性列表
// 支持以下格式：
//   - 简单属性: userName, displayName
//   - 嵌套属性: name.givenName, emails.value
//   - 通配符模式: members.*, groups.*
//   - 多属性组合: members.value,members.type,displayName
//   - 混合模式: members.*,displayName,name.givenName
//
// 示例：
//
//	"userName"                    -> ["userName"]
//	"userName,displayName"        -> ["userName", "displayName"]
//	"members.*"                   -> ["members.*"]
//	"members.value,members.type"  -> ["members.value", "members.type"]
//	"members.*,displayName"       -> ["members.*", "displayName"]
func ParseAttributeList(attrs string) []string {
	return parseAttributeList(attrs)
}

// parseAttributeList 解析属性列表（内部实现）
func parseAttributeList(attrs string) []string {
	var result []string
	parts := strings.Split(attrs, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// filterAttributes 过滤属性
// include: true 表示只保留指定属性, false 表示排除指定属性
// 支持不区分大小写的属性匹配，支持通配符模式（如 members.*）
func filterAttributes(obj map[string]interface{}, attrs []string, include bool) map[string]interface{} {
	if include {
		// 只保留指定属性
		result := make(map[string]interface{})

		// 首先处理所有属性，收集通配符模式
		wildcardPatterns := make(map[string]string) // 记录哪些父属性需要通配符（小写键 -> 原始属性名）
		specificAttrs := make([]string, 0)          // 非通配符属性

		for _, attr := range attrs {
			if strings.HasSuffix(attr, ".*") {
				// 通配符模式，如 members.*
				parentAttr := strings.TrimSuffix(attr, ".*")
				wildcardPatterns[strings.ToLower(parentAttr)] = parentAttr
			} else {
				specificAttrs = append(specificAttrs, attr)
			}
		}

		// 收集同一父属性的所有子属性
		nestedAttrs := make(map[string][]string) // 父属性 -> 子属性列表
		for _, attr := range specificAttrs {
			if strings.Contains(attr, ".") {
				parts := strings.SplitN(attr, ".", 2)
				parentKey := strings.ToLower(parts[0])
				nestedAttrs[parentKey] = append(nestedAttrs[parentKey], parts[1])
			}
		}

		// 处理特定属性
		for _, attr := range specificAttrs {
			if strings.Contains(attr, ".") {
				// 嵌套属性，如 name.givenName 或 members.value
				parts := strings.SplitN(attr, ".", 2)
				parentKey := strings.ToLower(parts[0])
				found := false
				var foundKey string
				for k := range obj {
					if strings.EqualFold(k, parts[0]) {
						found = true
						foundKey = k
						break
					}
				}
				if found {
					// 检查是否为数组类型（如 members 或 groups）
					if arr, ok := obj[foundKey].([]interface{}); ok {
						// 对数组中的每个元素应用嵌套属性过滤
						// 使用收集到的所有子属性进行过滤
						filteredArray := make([]interface{}, 0, len(arr))
						for _, item := range arr {
							if itemMap, ok := item.(map[string]interface{}); ok {
								filtered := filterAttributes(itemMap, nestedAttrs[parentKey], true)
								filteredArray = append(filteredArray, filtered)
							}
						}
						result[foundKey] = filteredArray
					} else if nested, ok := obj[foundKey].(map[string]interface{}); ok {
						// 普通嵌套对象
						filtered := filterAttributes(nested, nestedAttrs[parentKey], true)
						if existing, ok := result[foundKey]; ok {
							// 合并已存在的嵌套属性
							if existingMap, ok := existing.(map[string]interface{}); ok {
								for k, v := range filtered {
									existingMap[k] = v
								}
							}
						} else {
							result[foundKey] = filtered
						}
					}
				} else {
					// 如果父属性不存在，为其添加默认值（空切片）
					// 只在第一次遇到这个父属性时添加
					if _, ok := result[parts[0]]; !ok {
						result[parts[0]] = []interface{}{}
					}
				}
			} else {
				// 不区分大小写匹配属性
				found := false
				for k, v := range obj {
					if strings.EqualFold(k, attr) {
						result[k] = v
						found = true
						break
					}
				}
				// 如果属性不存在，为其添加默认值（空切片）
				if !found {
					result[attr] = []interface{}{}
				}
			}
		}

		// 处理通配符模式
		for _, parentAttr := range wildcardPatterns {
			for k, v := range obj {
				if strings.EqualFold(k, parentAttr) {
					// 通配符模式：包含该属性的所有子属性
					if arr, ok := v.([]interface{}); ok {
						// 数组类型，包含所有元素的所有字段
						result[k] = arr
					} else if nested, ok := v.(map[string]interface{}); ok {
						// 对象类型，包含所有子属性
						result[k] = nested
					} else {
						// 基本类型，直接包含
						result[k] = v
					}
					break
				}
			}
			// 注意：通配符模式不添加默认值，只包含存在的属性
		}

		// 始终保留 schemas（使用原始键名保持一致性）
		if schemas, ok := obj["schemas"]; ok {
			result["schemas"] = schemas
		}
		if id, ok := obj["id"]; ok {
			result["id"] = id
		}
		return result
	}

	// 排除指定属性
	result := make(map[string]interface{})
	for k, v := range obj {
		shouldExclude := false
		for _, attr := range attrs {
			if strings.Contains(attr, ".") {
				parts := strings.SplitN(attr, ".", 2)
				if strings.EqualFold(k, parts[0]) {
					// 检查是否为数组类型
					if arr, ok := v.([]interface{}); ok {
						// 对数组中的每个元素应用排除过滤
						filteredArray := make([]interface{}, 0, len(arr))
						for _, item := range arr {
							if itemMap, ok := item.(map[string]interface{}); ok {
								filtered := filterAttributes(itemMap, []string{parts[1]}, false)
								filteredArray = append(filteredArray, filtered)
							} else {
								filteredArray = append(filteredArray, item)
							}
						}
						v = filteredArray
					} else if nested, ok := v.(map[string]interface{}); ok {
						v = filterAttributes(nested, []string{parts[1]}, false)
					}
				}
			} else if strings.EqualFold(k, attr) {
				shouldExclude = true
				break
			}
		}
		if !shouldExclude {
			result[k] = v
		}
	}
	return result
}

// GetNestedValue 获取嵌套属性值
func GetNestedValue(obj map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := interface{}(obj)

	for _, part := range parts {
		if current == nil {
			return nil
		}
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}

	return current
}

// SetNestedValue 设置嵌套属性值
func SetNestedValue(obj map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := obj

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		if _, ok := current[part]; !ok {
			current[part] = make(map[string]interface{})
		}

		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return
		}
	}
}

// StructToMap 将结构体转换为 map[string]interface{}
func StructToMap(obj interface{}) (map[string]interface{}, error) {
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

// MapToStruct 将 map[string]interface{} 转换为结构体
func MapToStruct(m map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// GetAttributeNames 获取结构体的所有属性名（包括嵌套）
func GetAttributeNames(t reflect.Type, prefix string) []string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	var names []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue // 跳过未导出字段
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		name := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
		}

		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}

		// 总是添加当前字段名
		names = append(names, fullName)

		// 如果是嵌套结构体，也添加其子字段
		if field.Type.Kind() == reflect.Struct && field.Type != reflect.TypeOf((*string)(nil)).Elem() {
			nested := GetAttributeNames(field.Type, fullName)
			names = append(names, nested...)
		}
	}

	return names
}

// ValidateAttributes 验证属性列表是否有效
func ValidateAttributes(validAttrs []string, attrs string) ([]string, []string) {
	if attrs == "" {
		return nil, nil
	}

	requested := ParseAttributeList(attrs)
	var valid []string
	var invalid []string

	validMap := make(map[string]bool)
	for _, v := range validAttrs {
		validMap[strings.ToLower(v)] = true
	}

	for _, attr := range requested {
		// 支持嵌套属性前缀匹配
		found := false
		attrLower := strings.ToLower(attr)
		for validAttr := range validMap {
			if strings.HasPrefix(validAttr, attrLower) || strings.HasPrefix(attrLower, validAttr) {
				found = true
				break
			}
		}
		if found {
			valid = append(valid, attr)
		} else {
			invalid = append(invalid, attr)
		}
	}

	return valid, invalid
}

// ValidateAttributeFormat 验证属性格式是否正确
// 检查属性名称是否符合 SCIM 规范，支持通配符模式（如 members.*）
func ValidateAttributeFormat(attrs string) error {
	if attrs == "" {
		return nil
	}

	requested := ParseAttributeList(attrs)
	for _, attr := range requested {
		// 检查属性是否为空
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
	}

	return nil
}

// validateAttributePart 验证属性部分的格式
func validateAttributePart(part string) error {
	if part == "" {
		return errors.New("empty part in nested attribute")
	}

	// 检查是否以数字开头（SCIM 规范不允许）
	if len(part) > 0 && part[0] >= '0' && part[0] <= '9' {
		return fmt.Errorf("attribute part cannot start with digit: %s", part)
	}

	// 检查是否包含非法字符
	for _, ch := range part {
		if !isAlphaNumeric(ch) && ch != '_' {
			return fmt.Errorf("attribute part contains invalid character: %c", ch)
		}
	}

	return nil
}

// isAlphaNumeric 检查字符是否为字母或数字
func isAlphaNumeric(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}
