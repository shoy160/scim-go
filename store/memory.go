package store

import (
	"encoding/json"
	"errors"
	"scim-go/model"
	"scim-go/util"
	"sort"
	"strings"
	"sync"
)

// MemoryStore 内存存储实现
type MemoryStore struct {
	users  map[string]*model.User
	groups map[string]*model.Group
	mu     sync.RWMutex // 读写锁，保证并发安全
}

// NewMemory 创建内存存储实例
func NewMemory() Store {
	return &MemoryStore{
		users:  make(map[string]*model.User),
		groups: make(map[string]*model.Group),
	}
}

// ---------------------- User 实现 ----------------------
func (m *MemoryStore) CreateUser(u *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查用户名唯一性
	for _, existing := range m.users {
		if strings.EqualFold(existing.UserName, u.UserName) {
			return model.ErrUniqueness
		}
	}

	m.users[u.ID] = u
	return nil
}

func (m *MemoryStore) GetUser(id string) (*model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *MemoryStore) ListUsers(q *model.ResourceQuery) ([]model.User, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []model.User
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
	start := q.StartIndex - 1
	if start < 0 {
		start = 0
	}
	if start > len(list) {
		return []model.User{}, total, nil
	}

	end := start + q.Count
	if end > len(list) {
		end = len(list)
	}

	return list[start:end], total, nil
}

func (m *MemoryStore) UpdateUser(u *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.users[u.ID]; !ok {
		return model.ErrNotFound
	}

	// 检查用户名唯一性（排除自己）
	for id, existing := range m.users {
		if id != u.ID && strings.EqualFold(existing.UserName, u.UserName) {
			return model.ErrUniqueness
		}
	}

	m.users[u.ID] = u
	return nil
}

func (m *MemoryStore) PatchUser(id string, ops []model.PatchOperation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.users[id]
	if !ok {
		return model.ErrNotFound
	}
	err := PatchResource(m, id, u, ops)
	if err != nil {
		return err
	}
	return nil
}

func (m *MemoryStore) DeleteUser(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.users[id]; !ok {
		return model.ErrNotFound
	}

	delete(m.users, id)
	return nil
}

// ---------------------- Group 实现 ----------------------
func (m *MemoryStore) CreateGroup(g *model.Group) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查组名唯一性
	for _, existing := range m.groups {
		if strings.EqualFold(existing.DisplayName, g.DisplayName) {
			return model.ErrUniqueness
		}
	}

	m.groups[g.ID] = g
	return nil
}

func (m *MemoryStore) GetGroup(id string, preloadMembers bool) (*model.Group, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.groups[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return g, nil
}

func (m *MemoryStore) ListGroups(q *model.ResourceQuery, preloadMembers bool) ([]model.Group, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []model.Group
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
	start := q.StartIndex - 1
	if start < 0 {
		start = 0
	}
	if start > len(list) {
		return []model.Group{}, total, nil
	}

	end := start + q.Count
	if end > len(list) {
		end = len(list)
	}

	return list[start:end], total, nil
}

func (m *MemoryStore) UpdateGroup(g *model.Group) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groups[g.ID]; !ok {
		return model.ErrNotFound
	}

	// 检查组名唯一性（排除自己）
	for id, existing := range m.groups {
		if id != g.ID && strings.EqualFold(existing.DisplayName, g.DisplayName) {
			return model.ErrUniqueness
		}
	}

	m.groups[g.ID] = g
	return nil
}

func (m *MemoryStore) PatchGroup(id string, ops []model.PatchOperation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	g, ok := m.groups[id]
	if !ok {
		return model.ErrNotFound
	}

	err := PatchResource(m, id, g, ops)
	if err != nil {
		return err
	}
	return nil
}

func (m *MemoryStore) DeleteGroup(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groups[id]; !ok {
		return model.ErrNotFound
	}

	delete(m.groups, id)
	return nil
}

// ---------------------- Group 成员管理 ----------------------

// AddUserToGroup 添加用户到组
func (m *MemoryStore) AddUserToGroup(groupID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证组是否存在
	group, ok := m.groups[groupID]
	if !ok {
		return model.ErrNotFound
	}

	// 验证用户是否存在
	if _, ok := m.users[userID]; !ok {
		return model.ErrNotFound
	}

	// 检查用户是否已在组中
	for _, member := range group.Members {
		if member.Value == userID {
			return errors.New("user already in group")
		}
	}

	// 添加成员
	group.Members = append(group.Members, model.Member{
		GroupID: groupID,
		Value:   userID,
	})

	m.groups[groupID] = group
	return nil
}

// RemoveUserFromGroup 从组中移除用户
func (m *MemoryStore) RemoveUserFromGroup(groupID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证组是否存在
	group, ok := m.groups[groupID]
	if !ok {
		return model.ErrNotFound
	}

	// 查找并移除成员
	found := false
	var newMembers []model.Member
	for _, member := range group.Members {
		if member.Value == userID {
			found = true
		} else {
			newMembers = append(newMembers, member)
		}
	}

	if !found {
		return errors.New("user not in group")
	}

	group.Members = newMembers
	m.groups[groupID] = group
	return nil
}

// IsUserInGroup 检查用户是否在组中
func (m *MemoryStore) IsUserInGroup(groupID, userID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[groupID]
	if !ok {
		return false, model.ErrNotFound
	}

	for _, member := range group.Members {
		if member.Value == userID {
			return true, nil
		}
	}

	return false, nil
}

// GetUserGroups 获取用户所属的所有组
func (m *MemoryStore) GetUserGroups(userID string) ([]model.UserGroup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var groups []model.UserGroup
	for _, group := range m.groups {
		for _, member := range group.Members {
			if member.Value == userID {
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

// ---------------------- 辅助方法 ----------------------

// filterUsers 过滤用户列表
func (m *MemoryStore) filterUsers(users []model.User, filter string) ([]model.User, error) {
	node, err := util.ParseFilter(filter)
	if err != nil {
		return nil, err
	}

	var result []model.User
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

	var result []model.Group
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
func (m *MemoryStore) sortUsers(users []model.User, sortBy, sortOrder string) {
	less := func(i, j int) bool {
		var vi, vj string
		switch strings.ToLower(sortBy) {
		case "username":
			vi, vj = users[i].UserName, users[j].UserName
		case "displayname":
			vi, vj = users[i].DisplayName, users[j].DisplayName
		case "id":
			vi, vj = users[i].ID, users[j].ID
		default:
			return false
		}

		if strings.ToLower(sortOrder) == "descending" {
			return vi > vj
		}
		return vi < vj
	}

	sort.Slice(users, less)
}

// sortGroups 排序组列表
func (m *MemoryStore) sortGroups(groups []model.Group, sortBy, sortOrder string) {
	less := func(i, j int) bool {
		var vi, vj string
		switch strings.ToLower(sortBy) {
		case "displayname":
			vi, vj = groups[i].DisplayName, groups[j].DisplayName
		case "id":
			vi, vj = groups[i].ID, groups[j].ID
		default:
			return false
		}

		if strings.ToLower(sortOrder) == "descending" {
			return vi > vj
		}
		return vi < vj
	}

	sort.Slice(groups, less)
}

// userToMap 将用户转换为 map
func userToMap(u *model.User) (map[string]interface{}, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// groupToMap 将组转换为 map
func groupToMap(g *model.Group) (map[string]interface{}, error) {
	data, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}
