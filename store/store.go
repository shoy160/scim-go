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

// generateVersion 生成 SCIM 版本标识符（ETag 格式）
func generateVersion() string {
	// 使用当前时间戳生成唯一版本标识
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
	// 检查重复的 email（使用小写作为键）
	existingEmails := make(map[string]bool)
	for _, e := range user.Emails {
		existingEmails[strings.ToLower(e.Value)] = true
	}
	for _, email := range emails {
		parsedEmail, err := parseEmail(email, user.ID)
		if err != nil {
			return err
		}
		// 检查是否与现有 email 重复（不区分大小写）
		if existingEmails[strings.ToLower(parsedEmail.Value)] {
			return fmt.Errorf("duplicate email address: %s", parsedEmail.Value)
		}
		user.Emails = append(user.Emails, parsedEmail)
		existingEmails[strings.ToLower(parsedEmail.Value)] = true
	}
	return nil
}

// handleReplaceEmails 处理替换用户的 emails
func handleReplaceEmails(data any, value any) error {
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
	user.Emails = make([]model.Email, 0, len(emails))
	// 检查重复的 email
	existingEmails := make(map[string]bool)
	for _, email := range emails {
		parsedEmail, err := parseEmail(email, user.ID)
		if err != nil {
			return err
		}
		// 检查是否重复
		if existingEmails[parsedEmail.Value] {
			return fmt.Errorf("duplicate email address in request: %s", parsedEmail.Value)
		}
		user.Emails = append(user.Emails, parsedEmail)
		existingEmails[parsedEmail.Value] = true
	}
	return nil
}

// parseRole 解析单个role对象
func parseRole(role any, userID string) (model.Role, error) {
	roleMap, ok := role.(map[string]any)
	if !ok {
		return model.Role{}, fmt.Errorf("role must be an object")
	}
	roleValueVal, ok := roleMap["value"]
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
	if t, ok := roleMap["type"].(string); ok {
		roleType = t
	}
	display := ""
	if d, ok := roleMap["display"].(string); ok {
		display = d
	}
	primary := false
	if p, ok := roleMap["primary"].(bool); ok {
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

// handleAddRoles 处理添加 roles 到用户
func handleAddRoles(data any, value any) error {
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
	// 检查重复的 role（使用小写作为键）
	existingRoles := make(map[string]bool)
	for _, r := range user.Roles {
		existingRoles[strings.ToLower(r.Value)] = true
	}
	for _, role := range roles {
		parsedRole, err := parseRole(role, user.ID)
		if err != nil {
			return err
		}
		// 检查是否与现有 role 重复（不区分大小写）
		if existingRoles[strings.ToLower(parsedRole.Value)] {
			return fmt.Errorf("duplicate role: %s", parsedRole.Value)
		}
		user.Roles = append(user.Roles, parsedRole)
		existingRoles[strings.ToLower(parsedRole.Value)] = true
	}
	return nil
}

// handleReplaceRoles 处理替换用户的 roles
func handleReplaceRoles(data any, value any) error {
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
	user.Roles = make([]model.Role, 0, len(roles))
	// 检查重复的 role（使用小写作为键）
	existingRoles := make(map[string]bool)
	for _, role := range roles {
		parsedRole, err := parseRole(role, user.ID)
		if err != nil {
			return err
		}
		// 检查是否重复（不区分大小写）
		if existingRoles[strings.ToLower(parsedRole.Value)] {
			return fmt.Errorf("duplicate role in request: %s", parsedRole.Value)
		}
		user.Roles = append(user.Roles, parsedRole)
		existingRoles[strings.ToLower(parsedRole.Value)] = true
	}
	return nil
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
