package util

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"scim-go/model"
	"strconv"
	"strings"
	"time"
)

// SetValueByPath
// obj 必须是指针（&user）
// path 如 "active" / "name.givenName" / "members[0].value" / "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.employeeNumber"
// value 要设置的值
func SetValueByPath(obj any, path string, value any) error {
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
		if key == model.EnterpriseUserSchema.String() {
			// 企业扩展属性是通过嵌入 EnterpriseUserExtension 结构体实现的
			finalField, _ = fieldByNameIgnoreCase(finalField, "EnterpriseUserExtension")
			// 如果 EnterpriseUserExtension 为 nil，需要初始化
			if finalField.Kind() == reflect.Ptr && finalField.IsNil() {
				finalField.Set(reflect.New(finalField.Type().Elem()))
				finalField = finalField.Elem()
			}
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

		// 最后一段才赋值
		if i == len(keys)-1 {
			// 处理 manager 字段的特殊情况（map[string]any -> Manager 结构体）
			// 需要在解引用之前检查，因为 Manager 是指针类型
			if finalField.Type() == reflect.TypeOf(&model.Manager{}) || finalField.Type() == reflect.TypeOf(model.Manager{}) {
				if managerMap, ok := value.(map[string]any); ok {
					manager := &model.Manager{}
					if v, ok := managerMap["value"].(string); ok {
						manager.Value = v
					}
					if ref, ok := managerMap["$ref"].(string); ok {
						manager.Ref = ref
					}
					// 如果 finalField 是值类型，需要设置指针
					if finalField.Type() == reflect.TypeOf(model.Manager{}) {
						finalField.Set(reflect.ValueOf(*manager))
					} else {
						finalField.Set(reflect.ValueOf(manager))
					}
					return nil
				}
			}

			val := reflect.ValueOf(value)
			if val.Type().AssignableTo(finalField.Type()) {
				finalField.Set(val)
			} else if finalField.Kind() == reflect.Bool && val.Kind() == reflect.String {
				b, _ := strconv.ParseBool(val.String())
				finalField.SetBool(b)
			} else if finalField.Kind() == reflect.String && val.Kind() == reflect.Bool {
				finalField.SetString(strconv.FormatBool(val.Bool()))
			} else {
				return fmt.Errorf("type mismatch: cannot set %v to %s", value, path)
			}
		}
	}
	return nil
}

// RemoveByPath 根据属性路径删除/清空值
// obj: 必须是结构体指针（&user）
// path: 支持 active / name.givenName / members[0] / emails[0].value / "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.employeeNumber"
func RemoveByPath(obj any, path string) error {
	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return errors.New("obj must be a non-nil pointer")
	}
	val = val.Elem()

	keys := parsePath(path)
	if len(keys) == 0 {
		return errors.New("invalid path")
	}

	// 处理企业扩展属性的命名空间
	if keys[0] == model.EnterpriseUserSchema.String() {
		// 企业扩展属性是通过嵌入 EnterpriseUserExtension 结构体实现的
		enterpriseField, err := fieldByNameIgnoreCase(val, "EnterpriseUserExtension")
		if err != nil {
			return err
		}
		// 如果 EnterpriseUserExtension 为 nil，无需处理
		if enterpriseField.Kind() == reflect.Ptr && enterpriseField.IsNil() {
			return nil
		}
		// 如果是指针，解引用
		if enterpriseField.Kind() == reflect.Ptr {
			enterpriseField = enterpriseField.Elem()
		}
		return processRemove(enterpriseField, keys[1:])
	}

	return processRemove(val, keys)
}

func processRemove(current reflect.Value, keys []string) error {
	key := keys[0]
	isLast := len(keys) == 1

	// 处理数组下标：members[0]
	if strings.Contains(key, "[") {
		left := strings.Index(key, "[")
		right := strings.Index(key, "]")
		if right <= left+1 {
			return fmt.Errorf("invalid index: %s", key)
		}

		idxStr := key[left+1 : right]
		fieldName := key[:left]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			return err
		}

		// 获取切片字段
		sliceField, _ := fieldByNameIgnoreCase(current, fieldName)
		if !sliceField.IsValid() {
			return fmt.Errorf("field %s not found", fieldName)
		}
		if sliceField.Kind() != reflect.Slice {
			return fmt.Errorf("%s is not a slice", fieldName)
		}
		if idx < 0 || idx >= sliceField.Len() {
			return errors.New("index out of range")
		}

		// 最后一级：删除切片元素
		if isLast {
			newSlice := reflect.MakeSlice(sliceField.Type(), 0, sliceField.Len()-1)
			// 拼接 0~idx-1 和 idx+1~end
			newSlice = reflect.AppendSlice(newSlice, sliceField.Slice(0, idx))
			newSlice = reflect.AppendSlice(newSlice, sliceField.Slice(idx+1, sliceField.Len()))
			sliceField.Set(newSlice)
			return nil
		}

		// 进入数组元素继续处理
		return processRemove(sliceField.Index(idx), keys[1:])
	}

	// 处理普通字段
	field, _ := fieldByNameIgnoreCase(current, key)
	if !field.IsValid() {
		return fmt.Errorf("field %s not found", key)
	}

	if !isLast {
		// 递归处理子字段
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				return errors.New("nil pointer")
			}
			field = field.Elem()
		}
		return processRemove(field, keys[1:])
	}

	// 最后一级：清空字段值
	if field.CanSet() {
		field.Set(reflect.Zero(field.Type()))
	}
	return nil
}

func fieldByNameIgnoreCase(v reflect.Value, name string) (reflect.Value, error) {
	nameLower := strings.ToLower(name)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		fieldName := t.Field(i).Name
		if strings.ToLower(fieldName) == nameLower {
			return v.Field(i), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("field %s not found (case-insensitive)", name)
}

// parsePath 把 "name.givenName" 拆分成 ["name", "givenName"]
// "members[0].value" → ["members[0]", "value"]
// "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.employeeNumber" → ["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User", "employeeNumber"]
func parsePath(path string) []string {
	// 处理企业扩展属性的特殊路径
	enterpriseSchema := model.EnterpriseUserSchema.String()
	if strings.HasPrefix(path, enterpriseSchema+".") {
		// 移除 schema 前缀，返回剩余部分
		remaining := path[len(enterpriseSchema)+1:]
		return []string{enterpriseSchema, remaining}
	}
	return strings.Split(path, ".")
}

// MergeValue 将 value 合并到 obj 中
// 支持将 map[string]any 合并到结构体中
func MergeValue(obj any, value any) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("obj must be a non-nil pointer")
	}
	v = v.Elem()

	// 如果 value 是 map[string]any，遍历并设置每个字段
	valueMap, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("value must be a map[string]any")
	}

	for key, val := range valueMap {
		// 处理企业扩展属性的特殊情况
		enterpriseSchema := model.EnterpriseUserSchema.String()
		if key == enterpriseSchema {
			// 如果 value 是企业扩展属性对象，需要遍历其字段并设置
			if enterpriseMap, ok := val.(map[string]any); ok {
				for attrKey, attrVal := range enterpriseMap {
					fullPath := enterpriseSchema + "." + attrKey
					if err := SetValueByPath(obj, fullPath, attrVal); err != nil {
						// 如果某个字段设置失败，继续处理其他字段
						continue
					}
				}
			}
		} else if nestedMap, ok := val.(map[string]any); ok {
			// 处理嵌套对象（如 name 字段）
			// 递归处理嵌套对象的每个字段
			for nestedKey, nestedVal := range nestedMap {
				fullPath := key + "." + nestedKey
				if err := SetValueByPath(obj, fullPath, nestedVal); err != nil {
					// 如果某个字段设置失败，继续处理其他字段
					continue
				}
			}
		} else {
			if err := SetValueByPath(obj, key, val); err != nil {
				// 如果某个字段设置失败，继续处理其他字段
				continue
			}
		}
	}
	return nil
}

// RemoveValue 处理 path 为空的 remove 操作
// value 应该是一个对象，包含要删除的字段名（值被忽略，只要字段名存在就删除）
func RemoveValue(obj any, value any) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("obj must be a non-nil pointer")
	}

	// 如果 value 是 map[string]any，遍历并删除每个字段
	valueMap, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("value must be a map[string]any for remove operation with empty path")
	}

	for key := range valueMap {
		// 处理企业扩展属性的特殊情况
		enterpriseSchema := model.EnterpriseUserSchema.String()
		if key == enterpriseSchema {
			// 如果 value 是企业扩展属性对象，需要遍历其字段并删除
			if enterpriseMap, ok := valueMap[key].(map[string]any); ok {
				for attrKey := range enterpriseMap {
					fullPath := enterpriseSchema + "." + attrKey
					if err := RemoveByPath(obj, fullPath); err != nil {
						// 如果某个字段删除失败，继续处理其他字段
						continue
					}
				}
			}
		} else if nestedMap, ok := valueMap[key].(map[string]any); ok {
			// 处理嵌套对象（如 name 字段）
			// 递归处理嵌套对象的每个字段
			for nestedKey := range nestedMap {
				fullPath := key + "." + nestedKey
				if err := RemoveByPath(obj, fullPath); err != nil {
					// 如果某个字段删除失败，继续处理其他字段
					continue
				}
			}
		} else {
			// 删除单个字段
			if err := RemoveByPath(obj, key); err != nil {
				// 如果某个字段删除失败，继续处理其他字段
				continue
			}
		}
	}
	return nil
}

// ValidateEmailFormat 验证 email 格式是否符合 SCIM 2.0 规范
func ValidateEmailFormat(email string) error {
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

// ValidateRoleDefinition 验证 role 定义是否符合系统规范
func ValidateRoleDefinition(role string) error {
	if role == "" {
		return errors.New("role cannot be empty")
	}
	// 可以在这里添加更多的角色验证逻辑
	// 例如检查角色是否在预定义的角色列表中
	return nil
}

// GenerateID 生成唯一ID
func GenerateID() string {
	// 使用时间戳和随机数生成唯一ID
	timestamp := time.Now().UnixNano()
	hash := sha1.New()
	hash.Write([]byte(fmt.Sprintf("%d", timestamp)))
	return hex.EncodeToString(hash.Sum(nil))
}

// ContainsSchema 检查 schemas 列表中是否包含指定的 schema
func ContainsSchema(schemas []string, schema string) bool {
	for _, s := range schemas {
		if s == schema {
			return true
		}
	}
	return false
}

// RemoveSchema 从 schemas 列表中移除指定的 schema
func RemoveSchema(schemas []string, schema string) []string {
	result := make([]string, 0, len(schemas))
	for _, s := range schemas {
		if s != schema {
			result = append(result, s)
		}
	}
	return result
}
