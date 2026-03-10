package store

import (
	"encoding/json"
	"fmt"
	"scim-go/model"
	"scim-go/util"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryStore 内存存储实现
// 优化：添加用户名索引以提升查询性能
type MemoryStore struct {
	users                        map[string]*model.User
	groups                       map[string]*model.Group
	customResourceTypes          map[string]*model.CustomResourceType
	customResources              map[string]map[string]*model.CustomResource     // 资源类型 -> 资源ID -> 资源
	userNameIdx                  map[string]string                               // 用户名 -> 用户ID 索引
	groupNameIdx                 map[string]string                               // 组名 -> 组ID 索引
	customResourceTypeIdx        map[string]string                               // 资源类型名称 -> 资源类型ID 索引
	userTimestamps               map[string]struct{ CreatedAt, UpdatedAt int64 } // 用户时间戳
	groupTimestamps              map[string]struct{ CreatedAt, UpdatedAt int64 } // 组时间戳
	customResourceTypeTimestamps map[string]struct{ CreatedAt, UpdatedAt int64 } // 自定义资源类型时间戳
	mu                           sync.RWMutex                                    // 读写锁，保证并发安全
}

// NewMemory 创建内存存储实例
func NewMemory() Store {
	return &MemoryStore{
		users:                        make(map[string]*model.User),
		groups:                       make(map[string]*model.Group),
		customResourceTypes:          make(map[string]*model.CustomResourceType),
		customResources:              make(map[string]map[string]*model.CustomResource),
		userNameIdx:                  make(map[string]string),
		groupNameIdx:                 make(map[string]string),
		customResourceTypeIdx:        make(map[string]string),
		userTimestamps:               make(map[string]struct{ CreatedAt, UpdatedAt int64 }),
		groupTimestamps:              make(map[string]struct{ CreatedAt, UpdatedAt int64 }),
		customResourceTypeTimestamps: make(map[string]struct{ CreatedAt, UpdatedAt int64 }),
	}
}

// ---------------------- User 实现 ----------------------

// CreateUser 创建用户
// 优化：使用索引检查用户名唯一性，时间复杂度从 O(n) 降低到 O(1)
func (m *MemoryStore) CreateUser(u *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查用户名唯一性（使用索引）
	lowerUserName := strings.ToLower(u.UserName)
	if _, exists := m.userNameIdx[lowerUserName]; exists {
		return model.ErrUniqueness
	}

	// 记录时间戳
	now := time.Now()
	timestamp := struct{ CreatedAt, UpdatedAt int64 }{now.Unix(), now.Unix()}
	m.userTimestamps[u.ID] = timestamp

	// 生成 meta 属性
	createdAt := time.Unix(timestamp.CreatedAt, 0)
	updatedAt := time.Unix(timestamp.UpdatedAt, 0)
	created := createdAt.Format(time.RFC3339Nano)
	lastModified := updatedAt.Format(time.RFC3339Nano)
	version := util.GenerateVersion()

	u.Meta = model.Meta{
		ResourceType: "User",
		Created:      created,
		LastModified: lastModified,
		Location:     "", // 由API层动态生成
		Version:      version,
	}

	// 设置数据库存储的 meta 字段（ResourceType 不持久化，由API层动态生成）
	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt
	u.Version = version

	// 设置默认 schemas（如果未提供）
	if len(u.Schemas) == 0 {
		u.Schemas = []string{string(model.UserSchema)}
	}

	m.users[u.ID] = u
	m.userNameIdx[lowerUserName] = u.ID
	return nil
}

// GetUser 获取用户
// 优化：直接通过 ID 访问，时间复杂度 O(1)
func (m *MemoryStore) GetUser(id string) (*model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	u, ok := m.users[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

// ListUsers 列出用户
// 优化：预分配切片容量，减少内存分配次数
func (m *MemoryStore) ListUsers(q *model.ResourceQuery) ([]model.User, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 预分配切片容量
	list := make([]model.User, 0, len(m.users))
	for _, u := range m.users {
		list = append(list, *u)
	}

	// 应用过滤器
	if q.Filter != "" {
		filtered, err := m.filterUsers(list, q.Filter)
		if err != nil {
			return nil, 0, err
		}
		list = filtered
	}

	total := int64(len(list))

	// 排序
	if q.SortBy != "" {
		m.sortUsers(list, q.SortBy, q.SortOrder)
	}

	// 分页
	return m.paginateUsers(list, q.StartIndex, q.Count), total, nil
}

// UpdateUser 更新用户
// 优化：使用索引检查用户名唯一性，并更新索引
func (m *MemoryStore) UpdateUser(u *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.users[u.ID]
	if !ok {
		return model.ErrNotFound
	}

	// 检查用户名唯一性（排除自己）
	lowerUserName := strings.ToLower(u.UserName)
	if existingID, exists := m.userNameIdx[lowerUserName]; exists && existingID != u.ID {
		return model.ErrUniqueness
	}

	// 更新索引：如果用户名改变，删除旧索引并添加新索引
	oldLowerUserName := strings.ToLower(existing.UserName)
	if oldLowerUserName != lowerUserName {
		delete(m.userNameIdx, oldLowerUserName)
		m.userNameIdx[lowerUserName] = u.ID
	}

	// 更新时间戳和 meta 属性
	now := time.Now()
	if timestamp, exists := m.userTimestamps[u.ID]; exists {
		timestamp.UpdatedAt = now.Unix()
		m.userTimestamps[u.ID] = timestamp
	} else {
		// 如果 timestamp 不存在，创建新的
		timestamp = struct{ CreatedAt, UpdatedAt int64 }{
			CreatedAt: now.Unix(),
			UpdatedAt: now.Unix(),
		}
		m.userTimestamps[u.ID] = timestamp
	}

	// 更新 meta 属性（在现有对象上）
	existing.Meta.LastModified = now.Format(time.RFC3339Nano)
	existing.Meta.Version = util.GenerateVersion()
	// Location 由API层动态生成

	// 更新数据库存储的 meta 字段
	existing.UpdatedAt = now
	existing.Version = existing.Meta.Version

	// 从请求体复制字段到现有对象
	existing.UserName = u.UserName
	existing.Name = u.Name
	existing.Active = u.Active
	existing.DisplayName = u.DisplayName
	existing.NickName = u.NickName
	existing.ProfileUrl = u.ProfileUrl

	// 处理 emails 的全量覆盖
	existing.Emails = u.Emails

	// 处理 roles 的全量覆盖
	existing.Roles = u.Roles

	existing.EnterpriseUserExtension = u.EnterpriseUserExtension
	// 更新 schemas（支持自定义 schemas）
	if len(u.Schemas) > 0 {
		existing.Schemas = u.Schemas
	}

	// ResourceType 不持久化，由API层动态生成
	// 保留原有的 CreatedAt
	if !existing.CreatedAt.IsZero() {
		// 已在 CreateUser 中设置
	}

	m.users[u.ID] = existing
	return nil
}

// PatchUser 补丁更新用户
func (m *MemoryStore) PatchUser(id string, ops []model.PatchOperation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.users[id]
	if !ok {
		return model.ErrNotFound
	}

	// 创建用户对象的深拷贝，避免在错误时修改原始数据
	uCopy, err := deepCopyUser(u)
	if err != nil {
		return fmt.Errorf("failed to copy user: %w", err)
	}

	// 如果更新了用户名，需要更新索引
	oldUserName := strings.ToLower(u.UserName)
	err = PatchResource(m, id, uCopy, ops)
	if err != nil {
		return err
	}

	newUserName := strings.ToLower(uCopy.UserName)
	if oldUserName != newUserName {
		delete(m.userNameIdx, oldUserName)
		m.userNameIdx[newUserName] = id
	}

	// 更新时间戳和 meta 属性
	if timestamp, exists := m.userTimestamps[id]; exists {
		now := time.Now()
		timestamp.UpdatedAt = now.Unix()
		m.userTimestamps[id] = timestamp

		// 更新 meta 属性
		uCopy.Meta.LastModified = now.Format(time.RFC3339)
		uCopy.Meta.Version = util.GenerateVersion()
		// Location 由API层动态生成
	}

	// 只有在没有错误时才更新原始用户对象
	m.users[id] = uCopy
	return nil
}

// DeleteUser 删除用户
// 优化：删除用户时同时清理索引
func (m *MemoryStore) DeleteUser(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.users[id]
	if !ok {
		return model.ErrNotFound
	}

	// 清理索引和时间戳
	delete(m.userNameIdx, strings.ToLower(u.UserName))
	delete(m.userTimestamps, id)
	delete(m.users, id)
	return nil
}

// deepCopyUser 创建用户对象的深拷贝
func deepCopyUser(u *model.User) (*model.User, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}
	var copy model.User
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	return &copy, nil
}

// ---------------------- Group 实现 ----------------------

// CreateGroup 创建组
// 优化：使用索引检查组名唯一性，并验证成员是否存在
func (m *MemoryStore) CreateGroup(g *model.Group) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查组名唯一性（使用索引）
	lowerGroupName := strings.ToLower(g.DisplayName)
	if _, exists := m.groupNameIdx[lowerGroupName]; exists {
		return model.ErrUniqueness
	}

	// 验证所有成员是否存在
	for _, member := range g.Members {
		if _, exists := m.users[member.Value]; !exists {
			return model.ErrNotFound
		}
	}

	// 记录时间戳
	now := time.Now()
	timestamp := struct{ CreatedAt, UpdatedAt int64 }{now.Unix(), now.Unix()}
	m.groupTimestamps[g.ID] = timestamp

	// 生成 meta 属性
	createdAt := time.Unix(timestamp.CreatedAt, 0)
	updatedAt := time.Unix(timestamp.UpdatedAt, 0)
	created := createdAt.Format(time.RFC3339Nano)
	lastModified := updatedAt.Format(time.RFC3339Nano)
	version := util.GenerateVersion()

	g.Meta = model.Meta{
		ResourceType: "Group",
		Created:      created,
		LastModified: lastModified,
		Location:     "", // 由API层动态生成
		Version:      version,
	}

	// 设置数据库存储的 meta 字段（ResourceType 不持久化，由API层动态生成）
	g.CreatedAt = createdAt
	g.UpdatedAt = updatedAt
	g.Version = version

	// 设置默认 schemas（如果未提供）
	if len(g.Schemas) == 0 {
		g.Schemas = []string{string(model.GroupSchema)}
	}

	m.groups[g.ID] = g
	m.groupNameIdx[lowerGroupName] = g.ID
	return nil
}

// GetGroup 获取组
func (m *MemoryStore) GetGroup(id string, preloadMembers bool) (*model.Group, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	g, ok := m.groups[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return g, nil
}

// ListGroups 列出组
// 优化：预分配切片容量
func (m *MemoryStore) ListGroups(q *model.ResourceQuery, preloadMembers bool) ([]model.Group, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 预分配切片容量
	list := make([]model.Group, 0, len(m.groups))
	for _, g := range m.groups {
		list = append(list, *g)
	}

	// 应用过滤器
	if q.Filter != "" {
		filtered, err := m.filterGroups(list, q.Filter)
		if err != nil {
			return nil, 0, err
		}
		list = filtered
	}

	total := int64(len(list))

	// 排序
	if q.SortBy != "" {
		m.sortGroups(list, q.SortBy, q.SortOrder)
	}

	// 分页
	return m.paginateGroups(list, q.StartIndex, q.Count), total, nil
}

// UpdateGroup 更新组
// 优化：使用索引检查组名唯一性，并更新索引
func (m *MemoryStore) UpdateGroup(g *model.Group) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.groups[g.ID]
	if !ok {
		return model.ErrNotFound
	}

	// 检查组名唯一性（排除自己）
	lowerGroupName := strings.ToLower(g.DisplayName)
	if existingID, exists := m.groupNameIdx[lowerGroupName]; exists && existingID != g.ID {
		return model.ErrUniqueness
	}

	// 更新索引：如果组名改变，删除旧索引并添加新索引
	oldLowerGroupName := strings.ToLower(existing.DisplayName)
	if oldLowerGroupName != lowerGroupName {
		delete(m.groupNameIdx, oldLowerGroupName)
		m.groupNameIdx[lowerGroupName] = g.ID
	}

	// 更新时间戳和 meta 属性
	now := time.Now()
	if timestamp, exists := m.groupTimestamps[g.ID]; exists {
		timestamp.UpdatedAt = now.Unix()
		m.groupTimestamps[g.ID] = timestamp
	} else {
		// 如果 timestamp 不存在，创建新的
		timestamp = struct{ CreatedAt, UpdatedAt int64 }{
			CreatedAt: now.Unix(),
			UpdatedAt: now.Unix(),
		}
		m.groupTimestamps[g.ID] = timestamp
	}

	// 更新 meta 属性（在现有对象上）
	existing.Meta.LastModified = now.Format(time.RFC3339Nano)
	existing.Meta.Version = util.GenerateVersion()
	// Location 由API层动态生成

	// 更新数据库存储的 meta 字段
	existing.UpdatedAt = now
	existing.Version = existing.Meta.Version

	// 从请求体复制字段到现有对象
	existing.DisplayName = g.DisplayName
	existing.Members = g.Members
	// 更新 schemas（支持自定义 schemas）
	if len(g.Schemas) > 0 {
		existing.Schemas = g.Schemas
	}

	// ResourceType 不持久化，由API层动态生成
	// 保留原有的 CreatedAt
	if !existing.CreatedAt.IsZero() {
		// 已在 CreateGroup 中设置
	}

	m.groups[g.ID] = existing
	return nil
}

// PatchGroup 补丁更新组
func (m *MemoryStore) PatchGroup(id string, ops []model.PatchOperation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	g, ok := m.groups[id]
	if !ok {
		return model.ErrNotFound
	}

	// 如果更新了组名，需要更新索引
	oldGroupName := strings.ToLower(g.DisplayName)
	err := PatchResource(m, id, g, ops)
	if err != nil {
		return err
	}

	newGroupName := strings.ToLower(g.DisplayName)
	if oldGroupName != newGroupName {
		delete(m.groupNameIdx, oldGroupName)
		m.groupNameIdx[newGroupName] = id
	}

	// 更新时间戳和 meta 属性
	if timestamp, exists := m.groupTimestamps[id]; exists {
		now := time.Now()
		timestamp.UpdatedAt = now.Unix()
		m.groupTimestamps[id] = timestamp

		// 更新 meta 属性
		g.Meta.LastModified = now.Format(time.RFC3339)
		g.Meta.Version = util.GenerateVersion()
		// Location 由API层动态生成
	}

	return nil
}

// DeleteGroup 删除组
// 优化：删除组时同时清理索引
func (m *MemoryStore) DeleteGroup(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	g, ok := m.groups[id]
	if !ok {
		return model.ErrNotFound
	}

	// 清理索引和时间戳
	delete(m.groupNameIdx, strings.ToLower(g.DisplayName))
	delete(m.groupTimestamps, id)
	delete(m.groups, id)
	return nil
}

// ---------------------- Group 成员管理 ----------------------

// AddMemberToGroup 添加成员到组（支持用户和组）
// 优化：使用 map 检查成员是否存在，时间复杂度从 O(n) 降低到 O(1)
func (m *MemoryStore) AddMemberToGroup(groupID, memberID string, memberType ...model.MemberType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	// 验证组是否存在
	group, ok := m.groups[groupID]
	if !ok {
		return model.ErrNotFound
	}

	// 验证成员是否存在
	if mt == "User" {
		if _, ok := m.users[memberID]; !ok {
			return model.ErrNotFound
		}
	} else if mt == "Group" {
		if _, ok := m.groups[memberID]; !ok {
			return model.ErrNotFound
		}
	} else {
		return model.ErrInvalidValue
	}

	// 检查成员是否已在组中（使用 map 优化）
	memberMap := make(map[string]bool)
	for _, member := range group.Members {
		memberMap[member.Value] = true
	}
	if memberMap[memberID] {
		return model.ErrMemberAlreadyInGroup
	}

	// 添加成员
	group.Members = append(group.Members, model.Member{
		GroupID: groupID,
		Value:   memberID,
		Type:    mt,
	})

	m.groups[groupID] = group
	return nil
}

// RemoveMemberFromGroup 从组中移除成员（支持用户和组）
// 优化：使用切片过滤，避免多次内存分配
func (m *MemoryStore) RemoveMemberFromGroup(groupID, memberID string, memberType ...model.MemberType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	// 验证组是否存在
	group, ok := m.groups[groupID]
	if !ok {
		return model.ErrNotFound
	}

	// 查找并移除成员
	found := false
	newMembers := make([]model.Member, 0, len(group.Members))
	for _, member := range group.Members {
		memberType := member.Type
		if memberType == "" {
			memberType = model.MemberTypeUser
		}
		if member.Value == memberID && (mt == "" || memberType == mt) {
			found = true
		} else {
			newMembers = append(newMembers, member)
		}
	}

	if !found {
		return model.ErrMemberNotInGroup
	}

	group.Members = newMembers
	m.groups[groupID] = group
	return nil
}

// IsMemberInGroup 检查成员是否在组中（支持用户和组）
// 优化：直接遍历，时间复杂度 O(n)
func (m *MemoryStore) IsMemberInGroup(groupID, memberID string, memberType ...model.MemberType) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	group, ok := m.groups[groupID]
	if !ok {
		return false, model.ErrNotFound
	}

	for _, member := range group.Members {
		memberType := member.Type
		if memberType == "" {
			memberType = model.MemberTypeUser
		}
		if member.Value == memberID && (mt == "" || memberType == mt) {
			return true, nil
		}
	}

	return false, nil
}

// GetMemberGroups 获取成员所属的所有组（支持用户和组）
// 优化：预分配切片容量
func (m *MemoryStore) GetMemberGroups(memberID string, memberType ...model.MemberType) ([]model.UserGroup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	// 预分配切片容量（假设平均每个成员在 3 个组中）
	groups := make([]model.UserGroup, 0, 3)
	for _, group := range m.groups {
		for _, member := range group.Members {
			memberType := member.Type
			if memberType == "" {
				memberType = "User"
			}
			if member.Value == memberID && (mt == "" || memberType == mt) {
				groups = append(groups, model.UserGroup{
					Value:   group.ID,
					Display: group.DisplayName,
				})
				break
			}
		}
	}

	return groups, nil
}

// GetGroupMembers 获取组成员（支持分页和类型过滤）
func (m *MemoryStore) GetGroupMembers(groupID string, memberType model.MemberType, q *model.ResourceQuery) ([]model.Member, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 验证组是否存在
	group, ok := m.groups[groupID]
	if !ok {
		return nil, 0, model.ErrNotFound
	}

	// 过滤成员
	var filteredMembers []model.Member
	for _, member := range group.Members {
		if memberType == "" || member.Type == memberType {
			filteredMembers = append(filteredMembers, member)
		}
	}

	total := int64(len(filteredMembers))

	// 分页处理
	startIndex := q.StartIndex
	count := q.Count
	if startIndex < 1 {
		startIndex = 1
	}
	if count <= 0 {
		count = len(filteredMembers)
	}

	start := startIndex - 1
	end := start + count
	if start > len(filteredMembers) {
		return []model.Member{}, total, nil
	}
	if end > len(filteredMembers) {
		end = len(filteredMembers)
	}

	return filteredMembers[start:end], total, nil
}

// RemoveEmailFromUser 从用户中移除指定邮箱
// 注意：此方法不在 PatchUser 的锁外层调用，因此不需要再次加锁
func (m *MemoryStore) RemoveEmailFromUser(userID, emailValue string) error {
	u, ok := m.users[userID]
	if !ok {
		return model.ErrNotFound
	}

	for i, email := range u.Emails {
		if strings.EqualFold(email.Value, emailValue) {
			u.Emails = append(u.Emails[:i], u.Emails[i+1:]...)
			return nil
		}
	}
	// 如果在内存中找不到，直接返回 nil（可能已经被 handleRemoveEmails 移除了）
	return nil
}

// RemoveRoleFromUser 从用户中移除指定角色
// 注意：此方法不在 PatchUser 的锁外层调用，因此不需要再次加锁
func (m *MemoryStore) RemoveRoleFromUser(userID, roleValue string) error {
	u, ok := m.users[userID]
	if !ok {
		return model.ErrNotFound
	}

	for i, role := range u.Roles {
		if strings.EqualFold(role.Value, roleValue) {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			return nil
		}
	}
	// 如果在内存中找不到，直接返回 nil（可能已经被 handleRemoveRoles 移除了）
	return nil
}

// ---------------------- 辅助方法 ----------------------

// paginateUsers 用户分页
// 优化：边界检查，避免不必要的切片操作
func (m *MemoryStore) paginateUsers(users []model.User, startIndex, count int) []model.User {
	result := Paginate(users, startIndex, count)
	if result == nil {
		return []model.User{}
	}
	return result.([]model.User)
}

// paginateGroups 组分页
func (m *MemoryStore) paginateGroups(groups []model.Group, startIndex, count int) []model.Group {
	result := Paginate(groups, startIndex, count)
	if result == nil {
		return []model.Group{}
	}
	return result.([]model.Group)
}

// filterUsers 过滤用户列表
func (m *MemoryStore) filterUsers(users []model.User, filter string) ([]model.User, error) {
	node, err := util.ParseFilter(filter)
	if err != nil {
		return nil, err
	}

	// 预分配切片容量
	result := make([]model.User, 0, len(users))
	for _, u := range users {
		obj, err := userToMap(&u)
		if err != nil {
			return nil, err
		}

		match, err := util.MatchFilter(node, obj)
		if err != nil {
			return nil, err
		}

		if match {
			result = append(result, u)
		}
	}

	return result, nil
}

// filterGroups 过滤组列表
func (m *MemoryStore) filterGroups(groups []model.Group, filter string) ([]model.Group, error) {
	node, err := util.ParseFilter(filter)
	if err != nil {
		return nil, err
	}

	// 预分配切片容量
	result := make([]model.Group, 0, len(groups))
	for _, g := range groups {
		obj, err := groupToMap(&g)
		if err != nil {
			return nil, err
		}

		match, err := util.MatchFilter(node, obj)
		if err != nil {
			return nil, err
		}

		if match {
			result = append(result, g)
		}
	}

	return result, nil
}

// sortUsers 排序用户列表
// 优化：支持多种排序字段
func (m *MemoryStore) sortUsers(users []model.User, sortBy, sortOrder string) {
	less := func(i, j int) bool {
		var cmp int
		switch sortBy {
		case "userName":
			cmp = strings.Compare(strings.ToLower(users[i].UserName), strings.ToLower(users[j].UserName))
		case "displayName":
			cmp = strings.Compare(users[i].DisplayName, users[j].DisplayName)
		case "id":
			cmp = strings.Compare(users[i].ID, users[j].ID)
		default:
			cmp = strings.Compare(strings.ToLower(users[i].UserName), strings.ToLower(users[j].UserName))
		}

		if sortOrder == "descending" {
			return cmp > 0
		}
		return cmp < 0
	}

	sort.Slice(users, less)
}

// sortGroups 排序组列表
func (m *MemoryStore) sortGroups(groups []model.Group, sortBy, sortOrder string) {
	less := func(i, j int) bool {
		var cmp int
		switch sortBy {
		case "displayName":
			cmp = strings.Compare(groups[i].DisplayName, groups[j].DisplayName)
		case "id":
			cmp = strings.Compare(groups[i].ID, groups[j].ID)
		default:
			cmp = strings.Compare(groups[i].DisplayName, groups[j].DisplayName)
		}

		if sortOrder == "descending" {
			return cmp > 0
		}
		return cmp < 0
	}

	sort.Slice(groups, less)
}

// userToMap 将用户转换为map
func userToMap(u *model.User) (map[string]interface{}, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// groupToMap 将组转换为map
func groupToMap(g *model.Group) (map[string]interface{}, error) {
	data, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// ---------------------- 自定义资源类型实现 ----------------------

// CreateCustomResourceType 创建自定义资源类型
func (m *MemoryStore) CreateCustomResourceType(crt *model.CustomResourceType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证自定义资源类型
	if err := model.ValidateCustomResourceType(crt); err != nil {
		return err
	}

	// 检查资源类型名称唯一性
	lowerName := strings.ToLower(crt.Name)
	if _, exists := m.customResourceTypeIdx[lowerName]; exists {
		return model.ErrUniqueness
	}

	// 记录时间戳
	now := time.Now()
	timestamp := struct{ CreatedAt, UpdatedAt int64 }{now.Unix(), now.Unix()}
	m.customResourceTypeTimestamps[crt.ID] = timestamp

	// 生成 meta 属性
	createdAt := time.Unix(timestamp.CreatedAt, 0)
	updatedAt := time.Unix(timestamp.UpdatedAt, 0)
	created := createdAt.Format(time.RFC3339Nano)
	lastModified := updatedAt.Format(time.RFC3339Nano)

	crt.Meta = &model.Meta{
		ResourceType: "ResourceType",
		Created:      created,
		LastModified: lastModified,
		Location:     "", // 由API层动态生成
	}

	// 设置默认 schemas
	if len(crt.Schemas) == 0 {
		crt.Schemas = []string{model.ResourceTypeSchema}
	}

	m.customResourceTypes[crt.ID] = crt
	m.customResourceTypeIdx[lowerName] = crt.ID

	// 初始化该资源类型的自定义资源存储
	m.customResources[crt.ID] = make(map[string]*model.CustomResource)

	return nil
}

// GetCustomResourceType 获取自定义资源类型
func (m *MemoryStore) GetCustomResourceType(id string) (*model.CustomResourceType, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	crt, ok := m.customResourceTypes[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return crt, nil
}

// ListCustomResourceTypes 列出所有自定义资源类型
func (m *MemoryStore) ListCustomResourceTypes() ([]model.CustomResourceType, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]model.CustomResourceType, 0, len(m.customResourceTypes))
	for _, crt := range m.customResourceTypes {
		list = append(list, *crt)
	}
	return list, nil
}

// UpdateCustomResourceType 更新自定义资源类型
func (m *MemoryStore) UpdateCustomResourceType(crt *model.CustomResourceType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.customResourceTypes[crt.ID]
	if !ok {
		return model.ErrNotFound
	}

	// 验证自定义资源类型
	if err := model.ValidateCustomResourceType(crt); err != nil {
		return err
	}

	// 检查资源类型名称唯一性（排除自己）
	lowerName := strings.ToLower(crt.Name)
	if existingID, exists := m.customResourceTypeIdx[lowerName]; exists && existingID != crt.ID {
		return model.ErrUniqueness
	}

	// 更新索引：如果名称改变，删除旧索引并添加新索引
	oldLowerName := strings.ToLower(existing.Name)
	if oldLowerName != lowerName {
		delete(m.customResourceTypeIdx, oldLowerName)
		m.customResourceTypeIdx[lowerName] = crt.ID
	}

	// 更新时间戳和 meta 属性
	now := time.Now()
	if timestamp, exists := m.customResourceTypeTimestamps[crt.ID]; exists {
		timestamp.UpdatedAt = now.Unix()
		m.customResourceTypeTimestamps[crt.ID] = timestamp
	}

	// 更新 meta 属性
	existing.Meta.LastModified = now.Format(time.RFC3339Nano)
	// Location 由API层动态生成

	// 从请求体复制字段到现有对象
	existing.Name = crt.Name
	existing.Endpoint = crt.Endpoint
	existing.Description = crt.Description
	existing.Schema = crt.Schema
	existing.SchemaExtensions = crt.SchemaExtensions
	// 更新 schemas
	if len(crt.Schemas) > 0 {
		existing.Schemas = crt.Schemas
	}

	m.customResourceTypes[crt.ID] = existing
	return nil
}

// DeleteCustomResourceType 删除自定义资源类型
func (m *MemoryStore) DeleteCustomResourceType(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	crt, ok := m.customResourceTypes[id]
	if !ok {
		return model.ErrNotFound
	}

	// 清理索引和时间戳
	delete(m.customResourceTypeIdx, strings.ToLower(crt.Name))
	delete(m.customResourceTypeTimestamps, id)
	delete(m.customResourceTypes, id)
	delete(m.customResources, id) // 同时删除该资源类型的所有自定义资源

	return nil
}

// ---------------------- 自定义资源实现 ----------------------

// CreateCustomResource 创建自定义资源
func (m *MemoryStore) CreateCustomResource(cr *model.CustomResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证资源类型是否存在
	if _, ok := m.customResourceTypes[cr.ResourceType]; !ok {
		return model.ErrNotFound
	}

	// 验证自定义资源
	if err := model.ValidateCustomResource(cr); err != nil {
		return err
	}

	// 初始化该资源类型的自定义资源存储（如果不存在）
	if _, ok := m.customResources[cr.ResourceType]; !ok {
		m.customResources[cr.ResourceType] = make(map[string]*model.CustomResource)
	}

	// 记录时间戳
	now := time.Now()

	// 生成 meta 属性
	created := now.Format(time.RFC3339Nano)
	lastModified := now.Format(time.RFC3339Nano)
	version := util.GenerateVersion()

	cr.Meta = &model.Meta{
		ResourceType: cr.ResourceType,
		Created:      created,
		LastModified: lastModified,
		Location:     "", // 由API层动态生成
		Version:      version,
	}

	// 设置版本
	cr.Version = version

	m.customResources[cr.ResourceType][cr.ID] = cr
	return nil
}

// GetCustomResource 获取自定义资源
func (m *MemoryStore) GetCustomResource(id, resourceType string) (*model.CustomResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 验证资源类型是否存在
	if _, ok := m.customResourceTypes[resourceType]; !ok {
		return nil, model.ErrNotFound
	}

	// 验证资源是否存在
	if resources, ok := m.customResources[resourceType]; ok {
		if cr, ok := resources[id]; ok {
			return cr, nil
		}
	}

	return nil, model.ErrNotFound
}

// ListCustomResources 列出自定义资源
func (m *MemoryStore) ListCustomResources(q *model.CustomResourceQuery) ([]model.CustomResource, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 验证资源类型是否存在
	if _, ok := m.customResourceTypes[q.ResourceType]; !ok {
		return nil, 0, model.ErrNotFound
	}

	// 获取该资源类型的所有资源
	resources, ok := m.customResources[q.ResourceType]
	if !ok {
		return []model.CustomResource{}, 0, nil
	}

	// 预分配切片容量
	list := make([]model.CustomResource, 0, len(resources))
	for _, cr := range resources {
		list = append(list, *cr)
	}

	// 应用过滤器
	if q.Filter != "" {
		filtered, err := m.filterCustomResources(list, q.Filter)
		if err != nil {
			return nil, 0, err
		}
		list = filtered
	}

	total := int64(len(list))

	// 排序
	if q.SortBy != "" {
		m.sortCustomResources(list, q.SortBy, q.SortOrder)
	}

	// 分页
	return m.paginateCustomResources(list, q.StartIndex, q.Count), total, nil
}

// UpdateCustomResource 更新自定义资源
func (m *MemoryStore) UpdateCustomResource(cr *model.CustomResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证资源类型是否存在
	if _, ok := m.customResourceTypes[cr.ResourceType]; !ok {
		return model.ErrNotFound
	}

	// 验证自定义资源
	if err := model.ValidateCustomResource(cr); err != nil {
		return err
	}

	// 验证资源是否存在
	if resources, ok := m.customResources[cr.ResourceType]; ok {
		if existing, ok := resources[cr.ID]; ok {
			// 更新时间戳和 meta 属性
			now := time.Now()
			existing.Meta.LastModified = now.Format(time.RFC3339Nano)
			existing.Meta.Version = util.GenerateVersion()
			// Location 由API层动态生成

			// 更新版本
			existing.Version = existing.Meta.Version

			// 从请求体复制字段到现有对象
			existing.Attributes = cr.Attributes
			existing.ExternalID = cr.ExternalID

			m.customResources[cr.ResourceType][cr.ID] = existing
			return nil
		}
	}

	return model.ErrNotFound
}

// PatchCustomResource 补丁更新自定义资源
func (m *MemoryStore) PatchCustomResource(id, resourceType string, ops []model.PatchOperation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证资源类型是否存在
	if _, ok := m.customResourceTypes[resourceType]; !ok {
		return model.ErrNotFound
	}

	// 验证资源是否存在
	if resources, ok := m.customResources[resourceType]; ok {
		if cr, ok := resources[id]; ok {
			// 应用补丁操作
			err := PatchResource(m, id, cr, ops)
			if err != nil {
				return err
			}

			// 更新时间戳和 meta 属性
			now := time.Now()
			cr.Meta.LastModified = now.Format(time.RFC3339Nano)
			cr.Meta.Version = util.GenerateVersion()
			// Location 由API层动态生成

			// 更新版本
			cr.Version = cr.Meta.Version

			m.customResources[resourceType][id] = cr
			return nil
		}
	}

	return model.ErrNotFound
}

// DeleteCustomResource 删除自定义资源
func (m *MemoryStore) DeleteCustomResource(id, resourceType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证资源类型是否存在
	if _, ok := m.customResourceTypes[resourceType]; !ok {
		return model.ErrNotFound
	}

	// 验证资源是否存在
	if resources, ok := m.customResources[resourceType]; ok {
		if _, ok := resources[id]; ok {
			delete(resources, id)
			return nil
		}
	}

	return model.ErrNotFound
}

// ---------------------- 自定义资源辅助方法 ----------------------

// paginateCustomResources 自定义资源分页
func (m *MemoryStore) paginateCustomResources(resources []model.CustomResource, startIndex, count int) []model.CustomResource {
	result := Paginate(resources, startIndex, count)
	if result == nil {
		return []model.CustomResource{}
	}
	return result.([]model.CustomResource)
}

// filterCustomResources 过滤自定义资源列表
func (m *MemoryStore) filterCustomResources(resources []model.CustomResource, filter string) ([]model.CustomResource, error) {
	node, err := util.ParseFilter(filter)
	if err != nil {
		return nil, err
	}

	// 预分配切片容量
	result := make([]model.CustomResource, 0, len(resources))
	for _, cr := range resources {
		obj, err := customResourceToMap(&cr)
		if err != nil {
			return nil, err
		}

		match, err := util.MatchFilter(node, obj)
		if err != nil {
			return nil, err
		}

		if match {
			result = append(result, cr)
		}
	}

	return result, nil
}

// sortCustomResources 排序自定义资源列表
func (m *MemoryStore) sortCustomResources(resources []model.CustomResource, sortBy, sortOrder string) {
	less := func(i, j int) bool {
		var cmp int
		switch sortBy {
		case "id":
			cmp = strings.Compare(resources[i].ID, resources[j].ID)
		default:
			// 对于其他字段，尝试从 Attributes 中获取
			val1, ok1 := resources[i].Attributes[sortBy]
			val2, ok2 := resources[j].Attributes[sortBy]
			if ok1 && ok2 {
				if str1, ok := val1.(string); ok {
					if str2, ok := val2.(string); ok {
						cmp = strings.Compare(str1, str2)
					}
				}
			}
		}

		if sortOrder == "descending" {
			return cmp > 0
		}
		return cmp < 0
	}

	sort.Slice(resources, less)
}

// customResourceToMap 将自定义资源转换为map
func customResourceToMap(cr *model.CustomResource) (map[string]interface{}, error) {
	data, err := json.Marshal(cr)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	// 将 Attributes 中的字段合并到顶层
	if attributes, ok := obj["attributes"].(map[string]interface{}); ok {
		for k, v := range attributes {
			obj[k] = v
		}
		delete(obj, "attributes")
	}

	return obj, nil
}
