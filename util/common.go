package util

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// SetValueByPath
// obj 必须是指针（&user）
// path 如 "active" / "name.givenName" / "members[0].value"
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
		// 处理数组下标：members[0]
		if strings.Contains(key, "[") {
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
// path: 支持 active / name.givenName / members[0] / emails[0].value
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
func parsePath(path string) []string {
	return strings.Split(path, ".")
}
