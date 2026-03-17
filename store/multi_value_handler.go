package store

import (
	"fmt"
	"scim-go/model"
	"scim-go/util"
	"strings"
)

// MultiValueAttributeHandler 多值属性处理器接口
type MultiValueAttributeHandler[T any] interface {
	Parse(value map[string]interface{}, userID string) (T, error)
	ToMap(item T) map[string]interface{}
	UpdateFromMap(item *T, data map[string]interface{})
	GetUserID(item T) string
	Deduplicate(items []T) []T
}

// PhoneHandler 电话号码处理器
type PhoneHandler struct{}

func (h PhoneHandler) Parse(value map[string]interface{}, userID string) (model.PhoneNumber, error) {
	phoneValueVal, ok := value["value"]
	if !ok {
		return model.PhoneNumber{}, fmt.Errorf("phone number must have a value field")
	}
	phoneValue, ok := phoneValueVal.(string)
	if !ok {
		return model.PhoneNumber{}, fmt.Errorf("phone number value must be a string")
	}
	phoneType := "work"
	if t, ok := value["type"].(string); ok {
		phoneType = t
	}
	primary := false
	if p, ok := value["primary"].(bool); ok {
		primary = p
	}
	return model.PhoneNumber{
		UserID:  userID,
		Value:   phoneValue,
		Type:    phoneType,
		Primary: primary,
	}, nil
}

func (h PhoneHandler) ToMap(item model.PhoneNumber) map[string]interface{} {
	return map[string]interface{}{
		"value":   item.Value,
		"type":    item.Type,
		"primary": item.Primary,
	}
}

func (h PhoneHandler) UpdateFromMap(item *model.PhoneNumber, data map[string]interface{}) {
	if value, ok := data["value"].(string); ok {
		item.Value = value
	}
	if phoneType, ok := data["type"].(string); ok {
		item.Type = phoneType
	}
	if primary, ok := data["primary"].(bool); ok {
		item.Primary = primary
	}
}

func (h PhoneHandler) GetUserID(item model.PhoneNumber) string {
	return item.UserID
}

func (h PhoneHandler) Deduplicate(items []model.PhoneNumber) []model.PhoneNumber {
	return deduplicatePhoneNumbers(items)
}

// AddressHandler 地址处理器
type AddressHandler struct{}

func (h AddressHandler) Parse(value map[string]interface{}, userID string) (model.Address, error) {
	addr := model.Address{UserID: userID}
	if v, ok := value["value"].(string); ok {
		addr.Value = v
	}
	if d, ok := value["display"].(string); ok {
		addr.Display = d
	}
	if s, ok := value["streetAddress"].(string); ok {
		addr.StreetAddress = s
	}
	if l, ok := value["locality"].(string); ok {
		addr.Locality = l
	}
	if r, ok := value["region"].(string); ok {
		addr.Region = r
	}
	if p, ok := value["postalCode"].(string); ok {
		addr.PostalCode = p
	}
	if c, ok := value["country"].(string); ok {
		addr.Country = c
	}
	if f, ok := value["formatted"].(string); ok {
		addr.Formatted = f
	}
	addrType := "work"
	if t, ok := value["type"].(string); ok {
		addrType = t
	}
	addr.Type = addrType
	if p, ok := value["primary"].(bool); ok {
		addr.Primary = p
	}
	return addr, nil
}

func (h AddressHandler) ToMap(item model.Address) map[string]interface{} {
	return map[string]interface{}{
		"value":         item.Value,
		"display":       item.Display,
		"streetAddress": item.StreetAddress,
		"locality":      item.Locality,
		"region":        item.Region,
		"postalCode":    item.PostalCode,
		"country":       item.Country,
		"formatted":     item.Formatted,
		"type":          item.Type,
		"primary":       item.Primary,
	}
}

func (h AddressHandler) UpdateFromMap(item *model.Address, data map[string]interface{}) {
	if value, ok := data["value"].(string); ok {
		item.Value = value
	}
	if display, ok := data["display"].(string); ok {
		item.Display = display
	}
	if streetAddress, ok := data["streetAddress"].(string); ok {
		item.StreetAddress = streetAddress
	}
	if locality, ok := data["locality"].(string); ok {
		item.Locality = locality
	}
	if region, ok := data["region"].(string); ok {
		item.Region = region
	}
	if postalCode, ok := data["postalCode"].(string); ok {
		item.PostalCode = postalCode
	}
	if country, ok := data["country"].(string); ok {
		item.Country = country
	}
	if formatted, ok := data["formatted"].(string); ok {
		item.Formatted = formatted
	}
	if addrType, ok := data["type"].(string); ok {
		item.Type = addrType
	}
	if primary, ok := data["primary"].(bool); ok {
		item.Primary = primary
	}
}

func (h AddressHandler) GetUserID(item model.Address) string {
	return item.UserID
}

func (h AddressHandler) Deduplicate(items []model.Address) []model.Address {
	return deduplicateAddresses(items)
}

// EmailHandler 邮箱处理器
type EmailHandler struct{}

func (h EmailHandler) Parse(value map[string]interface{}, userID string) (model.Email, error) {
	emailValueVal, ok := value["value"]
	if !ok {
		return model.Email{}, fmt.Errorf("email must have a value field")
	}
	emailValue, ok := emailValueVal.(string)
	if !ok {
		return model.Email{}, fmt.Errorf("email value must be a string")
	}
	if err := util.ValidateEmailFormat(emailValue); err != nil {
		return model.Email{}, fmt.Errorf("invalid email format: %w", err)
	}
	emailType := "work"
	if t, ok := value["type"].(string); ok {
		emailType = t
	}
	primary := false
	if p, ok := value["primary"].(bool); ok {
		primary = p
	}
	return model.Email{
		UserID:  userID,
		Value:   emailValue,
		Type:    emailType,
		Primary: primary,
	}, nil
}

func (h EmailHandler) ToMap(item model.Email) map[string]interface{} {
	return map[string]interface{}{
		"value":   item.Value,
		"type":    item.Type,
		"primary": item.Primary,
	}
}

func (h EmailHandler) UpdateFromMap(item *model.Email, data map[string]interface{}) {
	if value, ok := data["value"].(string); ok {
		item.Value = value
	}
	if emailType, ok := data["type"].(string); ok {
		item.Type = emailType
	}
	if primary, ok := data["primary"].(bool); ok {
		item.Primary = primary
	}
}

func (h EmailHandler) GetUserID(item model.Email) string {
	return item.UserID
}

func (h EmailHandler) Deduplicate(items []model.Email) []model.Email {
	return deduplicateEmails(items)
}

// RoleHandler 角色处理器
type RoleHandler struct{}

func (h RoleHandler) Parse(value map[string]interface{}, userID string) (model.Role, error) {
	roleValueVal, ok := value["value"]
	if !ok {
		return model.Role{}, fmt.Errorf("role must have a value field")
	}
	roleValue, ok := roleValueVal.(string)
	if !ok {
		return model.Role{}, fmt.Errorf("role value must be a string")
	}
	if err := util.ValidateRoleDefinition(roleValue); err != nil {
		return model.Role{}, fmt.Errorf("invalid role definition: %w", err)
	}
	roleType := ""
	if t, ok := value["type"].(string); ok {
		roleType = t
	}
	display := ""
	if d, ok := value["display"].(string); ok {
		display = d
	}
	primary := false
	if p, ok := value["primary"].(bool); ok {
		primary = p
	}
	return model.Role{
		UserID:  userID,
		Value:   roleValue,
		Type:    roleType,
		Display: display,
		Primary: primary,
	}, nil
}

func (h RoleHandler) ToMap(item model.Role) map[string]interface{} {
	return map[string]interface{}{
		"value":   item.Value,
		"type":    item.Type,
		"display": item.Display,
		"primary": item.Primary,
	}
}

func (h RoleHandler) UpdateFromMap(item *model.Role, data map[string]interface{}) {
	if value, ok := data["value"].(string); ok {
		item.Value = value
	}
	if roleType, ok := data["type"].(string); ok {
		item.Type = roleType
	}
	if display, ok := data["display"].(string); ok {
		item.Display = display
	}
	if primary, ok := data["primary"].(bool); ok {
		item.Primary = primary
	}
}

func (h RoleHandler) GetUserID(item model.Role) string {
	return item.UserID
}

func (h RoleHandler) Deduplicate(items []model.Role) []model.Role {
	return deduplicateRoles(items)
}

// GenericMultiValueProcessor 通用的多值属性处理器
type GenericMultiValueProcessor[T any] struct {
	handler MultiValueAttributeHandler[T]
}

func NewGenericMultiValueProcessor[T any](handler MultiValueAttributeHandler[T]) *GenericMultiValueProcessor[T] {
	return &GenericMultiValueProcessor[T]{handler: handler}
}

// HandleAdd 处理添加操作
func (p *GenericMultiValueProcessor[T]) HandleAdd(items *[]T, value any, userID string, getExistingKey func(T) string) error {
	if value == nil {
		return nil
	}
	newItems, ok := value.([]any)
	if !ok {
		return fmt.Errorf("value must be an array")
	}

	existingKeys := make(map[string]bool)
	for _, item := range *items {
		key := getExistingKey(item)
		if key != "" {
			existingKeys[strings.ToLower(key)] = true
		}
	}

	for _, newItem := range newItems {
		parsed, err := p.handler.Parse(newItem.(map[string]interface{}), userID)
		if err != nil {
			return err
		}
		key := getExistingKey(parsed)
		if key != "" && existingKeys[strings.ToLower(key)] {
			return fmt.Errorf("duplicate item: %s", key)
		}
		*items = append(*items, parsed)
		if key != "" {
			existingKeys[strings.ToLower(key)] = true
		}
	}
	return nil
}

// HandleReplace 处理替换操作
func (p *GenericMultiValueProcessor[T]) HandleReplace(items *[]T, value any, userID string, getExistingKey func(T) string) error {
	if value == nil {
		return nil
	}
	newItems, ok := value.([]any)
	if !ok {
		return fmt.Errorf("value must be an array")
	}

	*items = make([]T, 0, len(newItems))
	existingKeys := make(map[string]bool)

	for _, newItem := range newItems {
		parsed, err := p.handler.Parse(newItem.(map[string]interface{}), userID)
		if err != nil {
			return err
		}
		key := getExistingKey(parsed)
		if key != "" && existingKeys[strings.ToLower(key)] {
			return fmt.Errorf("duplicate item in request: %s", key)
		}
		*items = append(*items, parsed)
		if key != "" {
			existingKeys[strings.ToLower(key)] = true
		}
	}
	return nil
}

// HandleFilterOperation 处理带过滤条件的操作
func (p *GenericMultiValueProcessor[T]) HandleFilterOperation(
	items *[]T,
	parsedPath *util.ParsedPath,
	op model.PatchOperation,
	userID string,
) error {
	// 转换为 map 数组
	mapItems := make([]map[string]interface{}, len(*items))
	for i, item := range *items {
		mapItems[i] = p.handler.ToMap(item)
	}

	// 查找匹配的索引
	indices, err := util.FindMatchingIndices(mapItems, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching items: %w", err)
	}

	// 对于 Add 操作
	if op.Op == "add" {
		if len(indices) == 0 {
			// 添加新项
			valueMap, ok := op.Value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("value must be a map for add operation")
			}
			newItem, err := p.handler.Parse(valueMap, userID)
			if err != nil {
				return err
			}
			*items = append(*items, newItem)
			return nil
		}
	} else {
		// Replace 操作
		if len(indices) == 0 {
			return util.ErrNoMatchingItems
		}
	}

	// 更新匹配项
	if parsedPath.SubPath != "" {
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for sub-path update")
		}
		for _, idx := range indices {
			if subValue, exists := valueMap[parsedPath.SubPath]; exists {
				mapItems[idx][parsedPath.SubPath] = subValue
			}
		}
	} else {
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for item update")
		}
		for _, idx := range indices {
			for key, val := range valueMap {
				mapItems[idx][key] = val
			}
		}
	}

	// 写回
	for i, mapItem := range mapItems {
		if i < len(*items) {
			item := &(*items)[i]
			p.handler.UpdateFromMap(item, mapItem)
		}
	}

	return nil
}

// HandleRemove 处理删除操作
func (p *GenericMultiValueProcessor[T]) HandleRemove(
	items *[]T,
	parsedPath *util.ParsedPath,
) error {
	mapItems := make([]map[string]interface{}, len(*items))
	for i, item := range *items {
		mapItems[i] = p.handler.ToMap(item)
	}

	indices, err := util.FindMatchingIndices(mapItems, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching items: %w", err)
	}

	if len(indices) == 0 {
		return util.ErrNoMatchingItems
	}

	// 从后向前删除
	for i := len(indices) - 1; i >= 0; i-- {
		idx := indices[i]
		*items = append((*items)[:idx], (*items)[idx+1:]...)
	}

	return nil
}
