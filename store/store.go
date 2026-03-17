package store

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"scim-go/model"
	"scim-go/util"
)

// 全局处理器实例
var (
	emailHandler     = EmailHandler{}
	phoneHandler     = PhoneHandler{}
	addressHandler   = AddressHandler{}
	roleHandler      = RoleHandler{}
	emailProcessor   = NewGenericMultiValueProcessor(emailHandler)
	phoneProcessor   = NewGenericMultiValueProcessor(phoneHandler)
	addressProcessor = NewGenericMultiValueProcessor(addressHandler)
	roleProcessor    = NewGenericMultiValueProcessor(roleHandler)
)

// generateVersion 生成 SCIM 版本标识符（ETag 格式）
func generateVersion() string {
	timestamp := time.Now().UnixNano()
	hash := sha1.New()
	hash.Write([]byte(fmt.Sprintf("%d", timestamp)))
	return fmt.Sprintf("W/\"%s\"", hex.EncodeToString(hash.Sum(nil)))
}

// Store 存储层通用接口（所有存储实现需实现此接口）
type Store interface {
	// ---------------------- User 相关 ----------------------
	CreateUser(u *model.User) error
	GetUser(id string) (*model.User, error)
	ListUsers(q *model.ResourceQuery) ([]model.User, int64, error)
	UpdateUser(u *model.User) error                        // 全量更新（PUT）
	PatchUser(id string, ops []model.PatchOperation) error // 补丁更新（PATCH）
	DeleteUser(id string) error

	// ---------------------- Group 相关 ----------------------
	CreateGroup(g *model.Group) error
	GetGroup(id string, preloadMembers bool) (*model.Group, error)
	ListGroups(q *model.ResourceQuery, preloadMembers bool) ([]model.Group, int64, error)
	UpdateGroup(g *model.Group) error                       // 全量更新（PUT）
	PatchGroup(id string, ops []model.PatchOperation) error // 补丁更新（PATCH）
	DeleteGroup(id string) error

	// ---------------------- Group 成员管理 ----------------------
	IsMemberInGroup(groupID, memberID string, memberType ...model.MemberType) (bool, error)                             // 检查成员是否在组中（支持用户和组）
	AddMemberToGroup(groupID, memberID string, memberType ...model.MemberType) error                                    // 添加成员到组（支持用户和组）
	RemoveMemberFromGroup(groupID, memberID string, memberType ...model.MemberType) error                               // 从组中移除成员（支持用户和组）
	GetGroupMembers(groupID string, memberType model.MemberType, q *model.ResourceQuery) ([]model.Member, int64, error) // 获取组成员（支持分页和类型过滤）

	// ---------------------- 反向查询 ----------------------
	GetMemberGroups(memberID string, memberType ...model.MemberType) ([]model.UserGroup, error) // 获取成员所属的所有组（支持用户和组）

	// ---------------------- Email/Role 管理 ----------------------
	RemoveEmailFromUser(userID, emailValue string) error // 从用户中移除指定邮箱
	RemoveRoleFromUser(userID, roleValue string) error   // 从用户中移除指定角色

	// ---------------------- 自定义资源类型相关 ----------------------
	CreateCustomResourceType(crt *model.CustomResourceType) error
	GetCustomResourceType(id string) (*model.CustomResourceType, error)
	ListCustomResourceTypes() ([]model.CustomResourceType, error)
	UpdateCustomResourceType(crt *model.CustomResourceType) error
	DeleteCustomResourceType(id string) error

	// ---------------------- 自定义资源相关 ----------------------
	CreateCustomResource(cr *model.CustomResource) error
	GetCustomResource(id, resourceType string) (*model.CustomResource, error)
	ListCustomResources(q *model.CustomResourceQuery) ([]model.CustomResource, int64, error)
	UpdateCustomResource(cr *model.CustomResource) error
	PatchCustomResource(id, resourceType string, ops []model.PatchOperation) error
	DeleteCustomResource(id, resourceType string) error
}

// PatchResource 处理 SCIM Patch 操作
// 支持 add/replace/remove 操作，处理成员和群组关联关系
func PatchResource(s Store, id string, data any, ops []model.PatchOperation) error {
	if len(ops) == 0 {
		return nil
	}
	var isGroup bool
	if _, ok := data.(*model.Group); ok {
		isGroup = true
	}
	var isUser bool
	if _, ok := data.(*model.User); ok {
		isUser = true
	}

	for _, op := range ops {
		// 验证操作类型
		if err := op.Validate(); err != nil {
			return fmt.Errorf("invalid patch operation: %w", err)
		}

		switch op.Op {
		case "add", "replace":
			if err := handleAddOrReplace(s, id, data, op, isGroup, isUser); err != nil {
				return err
			}
		case "remove":
			if err := handleRemove(s, id, data, op, isGroup, isUser); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported patch operation: %s", op.Op)
		}
	}
	return nil
}

// handleAddOrReplace 处理 add 和 replace 操作
func handleAddOrReplace(s Store, id string, data any, op model.PatchOperation, isGroup, isUser bool) error {
	// 检查路径是否包含过滤条件
	if strings.Contains(op.Path, "[") && strings.Contains(op.Path, "]") {
		return handlePathWithFilter(data, op)
	}

	// 成员字段的处理（Group 类型）
	if op.Path == "members" && isGroup {
		return handleAddMembers(s, id, op.Value)
	}
	// 群组字段的处理（User 类型）
	if op.Path == "groups" && isUser {
		return handleAddGroups(s, id, op.Value)
	}
	// emails 字段的处理（User 类型）
	if op.Path == "emails" && isUser {
		if op.Op == "add" {
			return handleAddEmails(data, op.Value)
		}
		return handleReplaceEmails(data, op.Value)
	}
	// phoneNumbers 字段的处理（User 类型）
	if op.Path == "phoneNumbers" && isUser {
		if op.Op == "add" {
			return handleAddPhoneNumbers(data, op.Value)
		}
		return handleReplacePhoneNumbers(data, op.Value)
	}
	// addresses 字段的处理（User 类型）
	if op.Path == "addresses" && isUser {
		if op.Op == "add" {
			return handleAddAddresses(data, op.Value)
		}
		return handleReplaceAddresses(data, op.Value)
	}
	// roles 字段的处理（User 类型）
	if op.Path == "roles" && isUser {
		if op.Op == "add" {
			return handleAddRoles(data, op.Value)
		}
		return handleReplaceRoles(data, op.Value)
	}
	// 企业扩展属性 schema 路径处理
	enterpriseSchema := model.EnterpriseUserSchema.String()
	if op.Path == enterpriseSchema && isUser {
		// 如果 path 是企业扩展属性 schema，value 应该是企业扩展属性对象
		if op.Value != nil {
			return util.MergeValue(data, map[string]any{
				enterpriseSchema: op.Value,
			})
		}
		return nil
	}
	// 通用字段处理
	if op.Path != "" {
		return util.SetValueByPath(data, op.Path, op.Value)
	}
	// 如果 Path 为空，Value 应该是一个对象，需要合并到 data 中
	if op.Value != nil {
		return util.MergeValue(data, op.Value)
	}
	return nil
}

// handlePathWithFilter 处理带有过滤条件的路径更新
// 支持如 phoneNumbers[type eq "mobile"] 或 addresses[type eq "work"] 的路径
// 对于 Add 操作：如果找不到匹配项，添加新项
// 对于 Replace 操作：如果找不到匹配项，返回错误
func handlePathWithFilter(data any, op model.PatchOperation) error {
	// 解析路径
	parsedPath, err := util.ParsePathWithFilter(op.Path)
	if err != nil {
		return fmt.Errorf("failed to parse path with filter: %w", err)
	}

	// 获取多值属性
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}

	// 根据属性名称处理不同的多值属性
	switch parsedPath.AttributeName {
	case "phoneNumbers":
		return handlePhoneNumbersWithFilter(user, parsedPath, op)
	case "addresses":
		return handleAddressesWithFilter(user, parsedPath, op)
	case "emails":
		return handleEmailsWithFilter(user, parsedPath, op)
	case "roles":
		return handleRolesWithFilter(user, parsedPath, op)
	default:
		return fmt.Errorf("unsupported multi-valued attribute: %s", parsedPath.AttributeName)
	}
}

// handlePhoneNumbersWithFilter 处理带有过滤条件的电话号码更新
// 对于 Add 操作：如果找不到匹配项，添加新项；如果找到匹配项，更新匹配项
// 对于 Replace 操作: 如果找不到匹配项，返回错误；如果找到匹配项，更新匹配项
func handlePhoneNumbersWithFilter(user *model.User, parsedPath *util.ParsedPath, op model.PatchOperation) error {
	// 将 phoneNumbers 转换为 map[string]interface{} 数组
	items := make([]map[string]interface{}, len(user.PhoneNumbers))
	for i, phone := range user.PhoneNumbers {
		items[i] = map[string]interface{}{
			"value":   phone.Value,
			"type":    phone.Type,
			"primary": phone.Primary,
		}
	}

	// 查找匹配的索引
	indices, err := util.FindMatchingIndices(items, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching phone numbers: %w", err)
	}

	// 对于 Add 操作，
	if op.Op == "add" {
		// Add 操作: 如果找不到匹配项，添加新项
		if len(indices) == 0 {
			// 解析 value 并添加新项
			valueMap, ok := op.Value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("value must be a map for add operation")
			}
			newPhone, err := phoneHandler.Parse(valueMap, user.ID)
			if err != nil {
				return err
			}
			user.PhoneNumbers = append(user.PhoneNumbers, newPhone)
			return nil
		}
		// 如果找到匹配项，更新匹配项
	} else {
		// Replace 操作: 如果找不到匹配项，返回错误
		if len(indices) == 0 {
			return util.ErrNoMatchingItems
		}
	}

	// 如果有子路径，更新匹配项的子属性
	if parsedPath.SubPath != "" {
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for sub-path update")
		}

		for _, idx := range indices {
			if subValue, exists := valueMap[parsedPath.SubPath]; exists {
				items[idx][parsedPath.SubPath] = subValue
			}
		}
	} else {
		// 更新整个匹配项
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for item update")
		}

		for _, idx := range indices {
			for key, val := range valueMap {
				items[idx][key] = val
			}
		}
	}

	// 将更新后的数据写回 user.PhoneNumbers
	for i, item := range items {
		if i < len(user.PhoneNumbers) {
			if value, ok := item["value"].(string); ok {
				user.PhoneNumbers[i].Value = value
			}
			if phoneType, ok := item["type"].(string); ok {
				user.PhoneNumbers[i].Type = phoneType
			}
			if primary, ok := item["primary"].(bool); ok {
				user.PhoneNumbers[i].Primary = primary
			}
		}
	}

	return nil
}

// handleAddressesWithFilter 处理带有过滤条件的地址更新
// 对于 Add 操作: 如果找不到匹配项,添加新项; 如果找到匹配项,更新匹配项
// 对于 Replace 操作: 如果找不到匹配项,返回错误; 如果找到匹配项,更新匹配项
func handleAddressesWithFilter(user *model.User, parsedPath *util.ParsedPath, op model.PatchOperation) error {
	// 将 addresses 转换为 map[string]interface{} 数组
	items := make([]map[string]interface{}, len(user.Addresses))
	for i, addr := range user.Addresses {
		items[i] = map[string]interface{}{
			"value":         addr.Value,
			"display":       addr.Display,
			"streetAddress": addr.StreetAddress,
			"locality":      addr.Locality,
			"region":        addr.Region,
			"postalCode":    addr.PostalCode,
			"country":       addr.Country,
			"formatted":     addr.Formatted,
			"type":          addr.Type,
			"primary":       addr.Primary,
		}
	}

	// 查找匹配的索引
	indices, err := util.FindMatchingIndices(items, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching addresses: %w", err)
	}

	// 对于 Add 操作
	if op.Op == "add" {
		// Add 操作: 如果找不到匹配项,添加新项
		if len(indices) == 0 {
			// 解析 value 并添加新项
			valueMap, ok := op.Value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("value must be a map for add operation")
			}
			newAddress, err := addressHandler.Parse(valueMap, user.ID)
			if err != nil {
				return err
			}
			user.Addresses = append(user.Addresses, newAddress)
			return nil
		}
		// 如果找到匹配项,继续执行更新操作
	} else {
		// Replace 操作: 如果找不到匹配项,返回错误
		if len(indices) == 0 {
			return util.ErrNoMatchingItems
		}
	}

	// 如果有子路径,更新匹配项的子属性
	if parsedPath.SubPath != "" {
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for sub-path update")
		}

		for _, idx := range indices {
			if subValue, exists := valueMap[parsedPath.SubPath]; exists {
				items[idx][parsedPath.SubPath] = subValue
			}
		}
	} else {
		// 更新整个匹配项
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for item update")
		}

		for _, idx := range indices {
			for key, val := range valueMap {
				items[idx][key] = val
			}
		}
	}

	// 将更新后的数据写回 user.Addresses
	for i, item := range items {
		if i < len(user.Addresses) {
			if value, ok := item["value"].(string); ok {
				user.Addresses[i].Value = value
			}
			if display, ok := item["display"].(string); ok {
				user.Addresses[i].Display = display
			}
			if streetAddress, ok := item["streetAddress"].(string); ok {
				user.Addresses[i].StreetAddress = streetAddress
			}
			if locality, ok := item["locality"].(string); ok {
				user.Addresses[i].Locality = locality
			}
			if region, ok := item["region"].(string); ok {
				user.Addresses[i].Region = region
			}
			if postalCode, ok := item["postalCode"].(string); ok {
				user.Addresses[i].PostalCode = postalCode
			}
			if country, ok := item["country"].(string); ok {
				user.Addresses[i].Country = country
			}
			if formatted, ok := item["formatted"].(string); ok {
				user.Addresses[i].Formatted = formatted
			}
			if addrType, ok := item["type"].(string); ok {
				user.Addresses[i].Type = addrType
			}
			if primary, ok := item["primary"].(bool); ok {
				user.Addresses[i].Primary = primary
			}
		}
	}

	return nil
}

// handleEmailsWithFilter 处理带有过滤条件的邮箱更新
// 对于 Add 操作: 如果找不到匹配项,添加新项; 如果找到匹配项,更新匹配项
// 对于 Replace 操作: 如果找不到匹配项,返回错误; 如果找到匹配项,更新匹配项
func handleEmailsWithFilter(user *model.User, parsedPath *util.ParsedPath, op model.PatchOperation) error {
	// 将 emails 转换为 map[string]interface{} 数组
	items := make([]map[string]interface{}, len(user.Emails))
	for i, email := range user.Emails {
		items[i] = map[string]interface{}{
			"value":   email.Value,
			"type":    email.Type,
			"primary": email.Primary,
		}
	}

	// 查找匹配的索引
	indices, err := util.FindMatchingIndices(items, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching emails: %w", err)
	}

	// 对于 Add 操作
	if op.Op == "add" {
		// Add 操作: 如果找不到匹配项,添加新项
		if len(indices) == 0 {
			// 解析 value 并添加新项
			valueMap, ok := op.Value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("value must be a map for add operation")
			}
			newEmail, err := parseEmail(valueMap, user.ID)
			if err != nil {
				return err
			}
			user.Emails = append(user.Emails, newEmail)
			return nil
		}
		// 如果找到匹配项,继续执行更新操作
	} else {
		// Replace 操作: 如果找不到匹配项,返回错误
		if len(indices) == 0 {
			return util.ErrNoMatchingItems
		}
	}

	// 如果有子路径,更新匹配项的子属性
	if parsedPath.SubPath != "" {
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for sub-path update")
		}

		for _, idx := range indices {
			if subValue, exists := valueMap[parsedPath.SubPath]; exists {
				items[idx][parsedPath.SubPath] = subValue
			}
		}
	} else {
		// 更新整个匹配项
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for item update")
		}

		for _, idx := range indices {
			for key, val := range valueMap {
				items[idx][key] = val
			}
		}
	}

	// 将更新后的数据写回 user.Emails
	for i, item := range items {
		if i < len(user.Emails) {
			if value, ok := item["value"].(string); ok {
				user.Emails[i].Value = value
			}
			if emailType, ok := item["type"].(string); ok {
				user.Emails[i].Type = emailType
			}
			if primary, ok := item["primary"].(bool); ok {
				user.Emails[i].Primary = primary
			}
		}
	}

	return nil
}

// handleRolesWithFilter 处理带有过滤条件的角色更新
// 对于 Add 操作: 如果找不到匹配项,添加新项; 如果找到匹配项,更新匹配项
// 对于 Replace 操作: 如果找不到匹配项,返回错误; 如果找到匹配项,更新匹配项
func handleRolesWithFilter(user *model.User, parsedPath *util.ParsedPath, op model.PatchOperation) error {
	// 将 roles 转换为 map[string]interface{} 数组
	items := make([]map[string]interface{}, len(user.Roles))
	for i, role := range user.Roles {
		items[i] = map[string]interface{}{
			"value":   role.Value,
			"type":    role.Type,
			"display": role.Display,
			"primary": role.Primary,
		}
	}

	// 查找匹配的索引
	indices, err := util.FindMatchingIndices(items, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching roles: %w", err)
	}

	// 对于 Add 操作
	if op.Op == "add" {
		// Add 操作: 如果找不到匹配项,添加新项
		if len(indices) == 0 {
			// 解析 value 并添加新项
			valueMap, ok := op.Value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("value must be a map for add operation")
			}
			newRole, err := roleHandler.Parse(valueMap, user.ID)
			if err != nil {
				return err
			}
			user.Roles = append(user.Roles, newRole)
			return nil
		}
		// 如果找到匹配项,继续执行更新操作
	} else {
		// Replace 操作: 如果找不到匹配项,返回错误
		if len(indices) == 0 {
			return util.ErrNoMatchingItems
		}
	}

	// 如果有子路径,更新匹配项的子属性
	if parsedPath.SubPath != "" {
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for sub-path update")
		}

		for _, idx := range indices {
			if subValue, exists := valueMap[parsedPath.SubPath]; exists {
				items[idx][parsedPath.SubPath] = subValue
			}
		}
	} else {
		// 更新整个匹配项
		valueMap, ok := op.Value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("value must be a map for item update")
		}

		for _, idx := range indices {
			for key, val := range valueMap {
				items[idx][key] = val
			}
		}
	}

	// 将更新后的数据写回 user.Roles
	for i, item := range items {
		if i < len(user.Roles) {
			if value, ok := item["value"].(string); ok {
				user.Roles[i].Value = value
			}
			if roleType, ok := item["type"].(string); ok {
				user.Roles[i].Type = roleType
			}
			if display, ok := item["display"].(string); ok {
				user.Roles[i].Display = display
			}
			if primary, ok := item["primary"].(bool); ok {
				user.Roles[i].Primary = primary
			}
		}
	}

	return nil
}

// handleAddMembers 处理添加成员到组
func handleAddMembers(s Store, groupID string, value any) error {
	if value == nil {
		return nil
	}
	members, ok := value.([]any)
	if !ok {
		return fmt.Errorf("members value must be an array")
	}
	for _, member := range members {
		memberMap, ok := member.(map[string]any)
		if !ok {
			return fmt.Errorf("member must be an object")
		}
		memberIDVal, ok := memberMap["value"]
		if !ok {
			return fmt.Errorf("member must have a value field")
		}
		memberID, ok := memberIDVal.(string)
		if !ok {
			return fmt.Errorf("member value must be a string")
		}
		memberType := model.MemberTypeUser // 默认类型
		if t, ok := memberMap["type"]; ok {
			if typeStr, ok := t.(string); ok {
				if typeStr == "Group" {
					memberType = model.MemberTypeGroup
				}
			}
		}
		if err := s.AddMemberToGroup(groupID, memberID, memberType); err != nil {
			return fmt.Errorf("failed to add member %s to group: %w", memberID, err)
		}
	}
	return nil
}

// handleAddGroups 处理添加用户到组
func handleAddGroups(s Store, userID string, value any) error {
	if value == nil {
		return nil
	}
	groups, ok := value.([]any)
	if !ok {
		return fmt.Errorf("groups value must be an array")
	}
	for _, group := range groups {
		groupMap, ok := group.(map[string]any)
		if !ok {
			return fmt.Errorf("group must be an object")
		}
		groupIDVal, ok := groupMap["value"]
		if !ok {
			return fmt.Errorf("group must have a value field")
		}
		groupID, ok := groupIDVal.(string)
		if !ok {
			return fmt.Errorf("group value must be a string")
		}
		if err := s.AddMemberToGroup(groupID, userID, model.MemberTypeUser); err != nil {
			return fmt.Errorf("failed to add user to group %s: %w", groupID, err)
		}
	}
	return nil
}

// handleRemove 处理 remove 操作
func handleRemove(s Store, id string, data any, op model.PatchOperation, isGroup, isUser bool) error {
	// 检查路径是否包含过滤条件
	if strings.Contains(op.Path, "[") && strings.Contains(op.Path, "]") {
		return handleRemoveWithFilter(data, op)
	}

	// 成员字段的处理（Group 类型）
	if op.Path == "members" && isGroup {
		return handleRemoveMembers(s, id, op.Value)
	}
	// 群组字段的处理（User 类型）
	if op.Path == "groups" && isUser {
		return handleRemoveGroups(s, id, op.Value)
	}
	// emails 字段的处理（User 类型）
	if op.Path == "emails" && isUser {
		return handleRemoveEmails(s, id, data, op.Value)
	}
	// phoneNumbers 字段的处理（User 类型）
	if op.Path == "phoneNumbers" && isUser {
		return handleRemovePhoneNumbers(s, id, data, op.Value)
	}
	// addresses 字段的处理（User 类型）
	if op.Path == "addresses" && isUser {
		return handleRemoveAddresses(s, id, data, op.Value)
	}
	// roles 字段的处理（User 类型）
	if op.Path == "roles" && isUser {
		return handleRemoveRoles(s, id, data, op.Value)
	}
	// 通用字段处理
	if op.Path != "" {
		return util.RemoveByPath(data, op.Path)
	}
	// 如果 Path 为空，Value 应该是一个对象，需要删除对象中指定的字段
	if op.Value != nil {
		return util.RemoveValue(data, op.Value)
	}
	return nil
}

// handleRemoveWithFilter 处理带有过滤条件的删除操作
func handleRemoveWithFilter(data any, op model.PatchOperation) error {
	// 解析路径
	parsedPath, err := util.ParsePathWithFilter(op.Path)
	if err != nil {
		return fmt.Errorf("failed to parse path with filter: %w", err)
	}

	// 获取多值属性
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}

	// 根据属性名称处理不同的多值属性
	switch parsedPath.AttributeName {
	case "phoneNumbers":
		return handleRemovePhoneNumbersWithFilter(user, parsedPath)
	case "addresses":
		return handleRemoveAddressesWithFilter(user, parsedPath)
	case "emails":
		return handleRemoveEmailsWithFilter(user, parsedPath)
	case "roles":
		return handleRemoveRolesWithFilter(user, parsedPath)
	default:
		return fmt.Errorf("unsupported multi-valued attribute: %s", parsedPath.AttributeName)
	}
}

// handleRemovePhoneNumbersWithFilter 处理带有过滤条件的电话号码删除
func handleRemovePhoneNumbersWithFilter(user *model.User, parsedPath *util.ParsedPath) error {
	// 将 phoneNumbers 转换为 map[string]interface{} 数组
	items := make([]map[string]interface{}, len(user.PhoneNumbers))
	for i, phone := range user.PhoneNumbers {
		items[i] = map[string]interface{}{
			"value":   phone.Value,
			"type":    phone.Type,
			"primary": phone.Primary,
		}
	}

	// 查找匹配的索引
	indices, err := util.FindMatchingIndices(items, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching phone numbers: %w", err)
	}

	if len(indices) == 0 {
		return util.ErrNoMatchingItems
	}

	// 从后向前删除，避免索引变化
	for i := len(indices) - 1; i >= 0; i-- {
		idx := indices[i]
		user.PhoneNumbers = append(user.PhoneNumbers[:idx], user.PhoneNumbers[idx+1:]...)
	}

	return nil
}

// handleRemoveAddressesWithFilter 处理带有过滤条件的地址删除
func handleRemoveAddressesWithFilter(user *model.User, parsedPath *util.ParsedPath) error {
	// 将 addresses 转换为 map[string]interface{} 数组
	items := make([]map[string]interface{}, len(user.Addresses))
	for i, addr := range user.Addresses {
		items[i] = map[string]interface{}{
			"value":         addr.Value,
			"display":       addr.Display,
			"streetAddress": addr.StreetAddress,
			"locality":      addr.Locality,
			"region":        addr.Region,
			"postalCode":    addr.PostalCode,
			"country":       addr.Country,
			"formatted":     addr.Formatted,
			"type":          addr.Type,
			"primary":       addr.Primary,
		}
	}

	// 查找匹配的索引
	indices, err := util.FindMatchingIndices(items, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching addresses: %w", err)
	}

	if len(indices) == 0 {
		return util.ErrNoMatchingItems
	}

	// 从后向前删除，避免索引变化
	for i := len(indices) - 1; i >= 0; i-- {
		idx := indices[i]
		user.Addresses = append(user.Addresses[:idx], user.Addresses[idx+1:]...)
	}

	return nil
}

// handleRemoveEmailsWithFilter 处理带有过滤条件的邮箱删除
func handleRemoveEmailsWithFilter(user *model.User, parsedPath *util.ParsedPath) error {
	// 将 emails 转换为 map[string]interface{} 数组
	items := make([]map[string]interface{}, len(user.Emails))
	for i, email := range user.Emails {
		items[i] = map[string]interface{}{
			"value":   email.Value,
			"type":    email.Type,
			"primary": email.Primary,
		}
	}

	// 查找匹配的索引
	indices, err := util.FindMatchingIndices(items, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching emails: %w", err)
	}

	if len(indices) == 0 {
		return util.ErrNoMatchingItems
	}

	// 从后向前删除，避免索引变化
	for i := len(indices) - 1; i >= 0; i-- {
		idx := indices[i]
		user.Emails = append(user.Emails[:idx], user.Emails[idx+1:]...)
	}

	return nil
}

// handleRemoveRolesWithFilter 处理带有过滤条件的角色删除
func handleRemoveRolesWithFilter(user *model.User, parsedPath *util.ParsedPath) error {
	// 将 roles 转换为 map[string]interface{} 数组
	items := make([]map[string]interface{}, len(user.Roles))
	for i, role := range user.Roles {
		items[i] = map[string]interface{}{
			"value":   role.Value,
			"type":    role.Type,
			"display": role.Display,
			"primary": role.Primary,
		}
	}

	// 查找匹配的索引
	indices, err := util.FindMatchingIndices(items, parsedPath.Filter)
	if err != nil {
		return fmt.Errorf("failed to find matching roles: %w", err)
	}

	if len(indices) == 0 {
		return util.ErrNoMatchingItems
	}

	// 从后向前删除，避免索引变化
	for i := len(indices) - 1; i >= 0; i-- {
		idx := indices[i]
		user.Roles = append(user.Roles[:idx], user.Roles[idx+1:]...)
	}

	return nil
}

// handleRemoveMembers 处理从组中移除成员
func handleRemoveMembers(s Store, groupID string, value any) error {
	// SCIM Patch remove 操作可以有两种形式：
	// 1. Path 指定具体成员路径，如 "members[value eq \"xxx\"]"
	// 2. Value 包含要移除的成员列表
	if value == nil {
		return nil
	}
	members, ok := value.([]any)
	if !ok {
		return fmt.Errorf("members value must be an array")
	}
	for _, member := range members {
		memberMap, ok := member.(map[string]any)
		if !ok {
			return fmt.Errorf("member must be an object")
		}
		memberIDVal, ok := memberMap["value"]
		if !ok {
			return fmt.Errorf("member must have a value field")
		}
		memberID, ok := memberIDVal.(string)
		if !ok {
			return fmt.Errorf("member value must be a string")
		}
		if err := s.RemoveMemberFromGroup(groupID, memberID); err != nil {
			return fmt.Errorf("failed to remove member %s from group: %w", memberID, err)
		}
	}
	return nil
}

// handleRemoveGroups 处理从组中移除用户
func handleRemoveGroups(s Store, userID string, value any) error {
	if value == nil {
		return nil
	}
	groups, ok := value.([]any)
	if !ok {
		return fmt.Errorf("groups value must be an array")
	}
	for _, group := range groups {
		groupMap, ok := group.(map[string]any)
		if !ok {
			return fmt.Errorf("group must be an object")
		}
		groupIDVal, ok := groupMap["value"]
		if !ok {
			return fmt.Errorf("group must have a value field")
		}
		groupID, ok := groupIDVal.(string)
		if !ok {
			return fmt.Errorf("group value must be a string")
		}
		if err := s.RemoveMemberFromGroup(groupID, userID); err != nil {
			return fmt.Errorf("failed to remove user from group %s: %w", groupID, err)
		}
	}
	return nil
}

// parseEmail 解析单个email对象
func parseEmail(email any, userID string) (model.Email, error) {
	emailMap, ok := email.(map[string]any)
	if !ok {
		return model.Email{}, fmt.Errorf("email must be an object")
	}
	emailValueVal, ok := emailMap["value"]
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
	if t, ok := emailMap["type"].(string); ok {
		emailType = t
	}
	primary := false
	if p, ok := emailMap["primary"].(bool); ok {
		primary = p
	}
	return model.Email{
		UserID:  userID,
		Value:   emailValue,
		Type:    emailType,
		Primary: primary,
	}, nil
}

// handleAddEmails 处理添加 emails 到用户
func handleAddEmails(data any, value any) error {
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	return emailProcessor.HandleAdd(
		&user.Emails,
		value,
		user.ID,
		func(e model.Email) string { return e.Value },
	)
}

// handleReplaceEmails 处理替换用户的 emails
func handleReplaceEmails(data any, value any) error {
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	return emailProcessor.HandleReplace(
		&user.Emails,
		value,
		user.ID,
		func(e model.Email) string { return e.Value },
	)
}

// handleAddRoles 处理添加 roles 到用户
func handleAddRoles(data any, value any) error {
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	return roleProcessor.HandleAdd(
		&user.Roles,
		value,
		user.ID,
		func(r model.Role) string { return r.Value },
	)
}

// handleReplaceRoles 处理替换用户的 roles
func handleReplaceRoles(data any, value any) error {
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	return roleProcessor.HandleReplace(
		&user.Roles,
		value,
		user.ID,
		func(r model.Role) string { return r.Value },
	)
}

// handleAddPhoneNumbers 处理添加 phoneNumbers 到用户
func handleAddPhoneNumbers(data any, value any) error {
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	return phoneProcessor.HandleAdd(
		&user.PhoneNumbers,
		value,
		user.ID,
		func(p model.PhoneNumber) string { return p.Value },
	)
}

// handleReplacePhoneNumbers 处理替换用户的 phoneNumbers
func handleReplacePhoneNumbers(data any, value any) error {
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	return phoneProcessor.HandleReplace(
		&user.PhoneNumbers,
		value,
		user.ID,
		func(p model.PhoneNumber) string { return p.Value },
	)
}

// handleAddAddresses 处理添加 addresses 到用户
func handleAddAddresses(data any, value any) error {
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	return addressProcessor.HandleAdd(
		&user.Addresses,
		value,
		user.ID,
		func(a model.Address) string {
			return a.StreetAddress + "," + a.Locality + "," + a.Region + "," + a.PostalCode + "," + a.Country
		},
	)
}

// handleReplaceAddresses 处理替换用户的 addresses
func handleReplaceAddresses(data any, value any) error {
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	return addressProcessor.HandleReplace(
		&user.Addresses,
		value,
		user.ID,
		func(a model.Address) string {
			return a.StreetAddress + "," + a.Locality + "," + a.Region + "," + a.PostalCode + "," + a.Country
		},
	)
}

// handleRemoveEmails 处理从用户中移除 emails
func handleRemoveEmails(s Store, userID string, data any, value any) error {
	if value == nil {
		return nil
	}
	emails, ok := value.([]any)
	if !ok {
		return fmt.Errorf("emails value must be an array")
	}
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	for _, email := range emails {
		emailMap, ok := email.(map[string]any)
		if !ok {
			return fmt.Errorf("email must be an object")
		}
		emailValueVal, ok := emailMap["value"]
		if !ok {
			return fmt.Errorf("email must have a value field")
		}
		emailValue, ok := emailValueVal.(string)
		if !ok {
			return fmt.Errorf("email value must be a string")
		}
		// 从内存中移除所有匹配的邮箱（不区分大小写）
		var newEmails []model.Email
		for _, e := range user.Emails {
			if !strings.EqualFold(e.Value, emailValue) {
				newEmails = append(newEmails, e)
			}
		}
		user.Emails = newEmails
		// 对于 DBStore，删除操作已在 PatchUser 中处理，这里不需要重复删除
		// 对于 MemoryStore 和其他存储，需要调用 Store 方法删除
		if _, isDB := s.(*DBStore); !isDB {
			if err := s.RemoveEmailFromUser(userID, emailValue); err != nil {
				return fmt.Errorf("failed to remove email %s from user: %w", emailValue, err)
			}
		}
	}
	return nil
}

// handleRemoveRoles 处理从用户中移除 roles
func handleRemoveRoles(s Store, userID string, data any, value any) error {
	if value == nil {
		return nil
	}
	roles, ok := value.([]any)
	if !ok {
		return fmt.Errorf("roles value must be an array")
	}
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	for _, role := range roles {
		roleMap, ok := role.(map[string]any)
		if !ok {
			return fmt.Errorf("role must be an object")
		}
		roleValueVal, ok := roleMap["value"]
		if !ok {
			return fmt.Errorf("role must have a value field")
		}
		roleValue, ok := roleValueVal.(string)
		if !ok {
			return fmt.Errorf("role value must be a string")
		}
		// 从内存中移除
		for i, r := range user.Roles {
			if strings.EqualFold(r.Value, roleValue) {
				user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
				break
			}
		}
		// 对于 DBStore，删除操作已在 PatchUser 中处理，这里不需要重复删除
		// 对于 MemoryStore 和其他存储，需要调用 Store 方法删除
		if _, isDB := s.(*DBStore); !isDB {
			if err := s.RemoveRoleFromUser(userID, roleValue); err != nil {
				return fmt.Errorf("failed to remove role %s from user: %w", roleValue, err)
			}
		}
	}
	return nil
}

// handleRemovePhoneNumbers 处理从用户中移除 phoneNumbers
func handleRemovePhoneNumbers(s Store, userID string, data any, value any) error {
	if value == nil {
		return nil
	}
	phoneNumbers, ok := value.([]any)
	if !ok {
		return fmt.Errorf("phoneNumbers value must be an array")
	}
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	for _, phone := range phoneNumbers {
		phoneMap, ok := phone.(map[string]any)
		if !ok {
			return fmt.Errorf("phone number must be an object")
		}
		phoneValueVal, ok := phoneMap["value"]
		if !ok {
			return fmt.Errorf("phone number must have a value field")
		}
		phoneValue, ok := phoneValueVal.(string)
		if !ok {
			return fmt.Errorf("phone number value must be a string")
		}
		// 从内存中移除所有匹配的电话号码（不区分大小写）
		var newPhoneNumbers []model.PhoneNumber
		for _, p := range user.PhoneNumbers {
			if !strings.EqualFold(p.Value, phoneValue) {
				newPhoneNumbers = append(newPhoneNumbers, p)
			}
		}
		user.PhoneNumbers = newPhoneNumbers
	}
	return nil
}

// handleRemoveAddresses 处理从用户中移除 addresses
func handleRemoveAddresses(s Store, userID string, data any, value any) error {
	if value == nil {
		return nil
	}
	addresses, ok := value.([]any)
	if !ok {
		return fmt.Errorf("addresses value must be an array")
	}
	user, ok := data.(*model.User)
	if !ok {
		return fmt.Errorf("data must be a User pointer")
	}
	for _, address := range addresses {
		addressMap, ok := address.(map[string]any)
		if !ok {
			return fmt.Errorf("address must be an object")
		}
		// 从内存中移除所有匹配的地址
		var newAddresses []model.Address
		for _, a := range user.Addresses {
			// 比较地址的关键字段
			match := true
			if streetAddress, ok := addressMap["streetAddress"].(string); ok {
				if !strings.EqualFold(a.StreetAddress, streetAddress) {
					match = false
				}
			}
			if locality, ok := addressMap["locality"].(string); ok {
				if !strings.EqualFold(a.Locality, locality) {
					match = false
				}
			}
			if region, ok := addressMap["region"].(string); ok {
				if !strings.EqualFold(a.Region, region) {
					match = false
				}
			}
			if postalCode, ok := addressMap["postalCode"].(string); ok {
				if !strings.EqualFold(a.PostalCode, postalCode) {
					match = false
				}
			}
			if country, ok := addressMap["country"].(string); ok {
				if !strings.EqualFold(a.Country, country) {
					match = false
				}
			}
			if !match {
				newAddresses = append(newAddresses, a)
			}
		}
		user.Addresses = newAddresses
	}
	return nil
}
