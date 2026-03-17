package model

import (
	"reflect"
	"strings"
	"time"
)

// SchemaGenerator 用于从Model动态生成Schema
// 实现自动从Model结构体反射生成Schema定义，确保Schema与Model保持同步

// GenerateSchemaFromModel 从Model结构体生成Schema
func GenerateSchemaFromModel(model interface{}, schemaID, schemaName, schemaDescription string) *Schema {
	attrs := generateAttributesFromStruct(reflect.TypeOf(model), "")
	return &Schema{
		Schemas:     []string{SchemaDefinitionSchema},
		ID:          schemaID,
		Name:        schemaName,
		Description: schemaDescription,
		Attributes:  attrs,
	}
}

// generateAttributesFromStruct 从结构体类型生成Schema属性
func generateAttributesFromStruct(t reflect.Type, prefix string) []SchemaAttribute {
	var attrs []SchemaAttribute

	// 如果是指针类型，获取其元素类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 只处理结构体类型
	if t.Kind() != reflect.Struct {
		return attrs
	}

	// 遍历结构体字段
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 跳过未导出字段
		if field.PkgPath != "" {
			continue
		}

		// 获取json标签
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// 解析json标签
		name := getFieldNameFromJSONTag(jsonTag)
		if name == "" {
			continue
		}

		// 构建完整路径
		fullPath := name
		if prefix != "" {
			fullPath = prefix + "." + name
		}

		// 生成属性
		attr := generateAttributeFromField(field, fullPath)

		// 如果是复杂类型，递归处理子属性
		if field.Type.Kind() == reflect.Struct && field.Type != reflect.TypeOf(time.Time{}) {
			// 跳过Meta类型，它是动态生成的
			if field.Type == reflect.TypeOf(Meta{}) {
				continue
			}
			subAttrs := generateAttributesFromStruct(field.Type, fullPath)
			if len(subAttrs) > 0 {
				attr.SubAttributes = subAttrs
				attr.Type = "complex"
			}
		} else if field.Type.Kind() == reflect.Slice {
			// 处理切片类型（多值属性）
			attr.MultiValued = true

			// 检查切片元素类型
			elemType := field.Type.Elem()
			if elemType.Kind() == reflect.Struct {
				// 对于复杂类型的切片，生成子属性
				subAttrs := generateAttributesFromStruct(elemType, fullPath)
				if len(subAttrs) > 0 {
					attr.SubAttributes = subAttrs
					attr.Type = "complex"
				}
			}
		}

		attrs = append(attrs, attr)
	}

	return attrs
}

// generateAttributeFromField 从字段生成Schema属性
func generateAttributeFromField(field reflect.StructField, fullPath string) SchemaAttribute {
	// 初始化属性
	attr := SchemaAttribute{
		Name:        getFieldNameFromJSONTag(field.Tag.Get("json")),
		MultiValued: field.Type.Kind() == reflect.Slice,
		Required:    false,
		CaseExact:   false,
		Mutability:  "readWrite",
		Returned:    "default",
		Uniqueness:  "none",
	}

	// 根据字段类型设置属性类型
	switch field.Type.Kind() {
	case reflect.String:
		attr.Type = "string"
	case reflect.Bool:
		attr.Type = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		attr.Type = "integer"
	case reflect.Float32, reflect.Float64:
		attr.Type = "decimal"
	case reflect.Slice:
		// 对于字符串切片，类型仍然是string
		if field.Type.Elem().Kind() == reflect.String {
			attr.Type = "string"
		} else {
			attr.Type = "complex"
		}
	case reflect.Struct:
		if field.Type == reflect.TypeOf(time.Time{}) {
			attr.Type = "dateTime"
		} else {
			attr.Type = "complex"
		}
	case reflect.Ptr:
		elemType := field.Type.Elem()
		if elemType.Kind() == reflect.Struct {
			attr.Type = "complex"
		} else {
			// 根据指针指向的类型设置
			switch elemType.Kind() {
			case reflect.String:
				attr.Type = "string"
			case reflect.Bool:
				attr.Type = "boolean"
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				attr.Type = "integer"
			case reflect.Float32, reflect.Float64:
				attr.Type = "decimal"
			}
		}
	}

	// 添加描述（可以从注释或其他标签获取）
	attr.Description = getFieldDescription(field)

	// 处理特殊字段
	switch attr.Name {
	case "userName":
		attr.Required = true
		attr.Uniqueness = "server"
	case "id":
		attr.Mutability = "readOnly"
	case "meta":
		attr.Mutability = "readOnly"
	case "groups":
		attr.Mutability = "readOnly"
	}

	// 处理地址类型的特殊字段
	if strings.Contains(fullPath, "addresses") {
		switch attr.Name {
		case "type":
			attr.CanonicalValues = []string{"work", "home", "other"}
		}
	}

	// 处理电话号码类型的特殊字段
	if strings.Contains(fullPath, "phoneNumbers") {
		switch attr.Name {
		case "type":
			attr.CanonicalValues = []string{"work", "home", "mobile", "other"}
		}
	}

	// 处理邮箱类型的特殊字段
	if strings.Contains(fullPath, "emails") {
		switch attr.Name {
		case "type":
			attr.CanonicalValues = []string{"work", "home", "other"}
		}
	}

	return attr
}

// getFieldNameFromJSONTag 从json标签获取字段名
func getFieldNameFromJSONTag(jsonTag string) string {
	if jsonTag == "" {
		return ""
	}
	parts := strings.Split(jsonTag, ",")
	return parts[0]
}

// getFieldDescription 从字段获取描述
func getFieldDescription(field reflect.StructField) string {
	// 可以从注释或其他标签获取描述
	// 这里简单返回空字符串，实际实现可以从注释解析
	return ""
}

// GetDynamicUserSchema 动态生成User Schema
func GetDynamicUserSchema() *Schema {
	return GenerateSchemaFromModel(&User{}, UserSchema.String(), "User", "User Account")
}

// GetDynamicGroupSchema 动态生成Group Schema
func GetDynamicGroupSchema() *Schema {
	return GenerateSchemaFromModel(&Group{}, GroupSchema.String(), "Group", "Group")
}

// GetDynamicEnterpriseUserExtensionSchema 动态生成企业用户扩展Schema
func GetDynamicEnterpriseUserExtensionSchema() *Schema {
	return GenerateSchemaFromModel(&EnterpriseUserExtension{}, EnterpriseUserSchema.String(), "EnterpriseUser", "Enterprise User Extension")
}
