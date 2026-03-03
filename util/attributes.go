package util

import (
	"encoding/json"
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
		attrs := parseAttributeList(attributes)
		result = filterAttributes(result, attrs, true)
	}

	// 处理 excludedAttributes 参数
	if excludedAttributes != "" {
		attrs := parseAttributeList(excludedAttributes)
		result = filterAttributes(result, attrs, false)
	}

	return result, nil
}

// parseAttributeList 解析属性列表
// 支持: userName,name.givenName,emails
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
func filterAttributes(obj map[string]interface{}, attrs []string, include bool) map[string]interface{} {
	if include {
		// 只保留指定属性
		result := make(map[string]interface{})
		for _, attr := range attrs {
			if strings.Contains(attr, ".") {
				// 嵌套属性，如 name.givenName
				parts := strings.SplitN(attr, ".", 2)
				if val, ok := obj[parts[0]]; ok {
					if nested, ok := val.(map[string]interface{}); ok {
						filtered := filterAttributes(nested, []string{parts[1]}, true)
						if existing, ok := result[parts[0]]; ok {
							// 合并已存在的嵌套属性
							if existingMap, ok := existing.(map[string]interface{}); ok {
								for k, v := range filtered {
									existingMap[k] = v
								}
							}
						} else {
							result[parts[0]] = filtered
						}
					}
				}
			} else {
				if val, ok := obj[attr]; ok {
					result[attr] = val
				}
			}
		}
		// 始终保留 schemas
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
				if k == parts[0] {
					if nested, ok := v.(map[string]interface{}); ok {
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

	requested := parseAttributeList(attrs)
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
