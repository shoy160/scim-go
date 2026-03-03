package store

import (
	"testing"

	"scim-go/model"
)

func TestMemoryStore_UserCRUD(t *testing.T) {
	store := NewMemory()

	// 测试创建用户
	user := &model.User{
		ID:       "user-1",
		UserName: "john.doe",
		Name: struct {
			GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
			FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "John",
			FamilyName: "Doe",
		},
		Active: true,
	}

	err := store.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	// 测试获取用户
	retrieved, err := store.GetUser("user-1")
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if retrieved.UserName != "john.doe" {
		t.Errorf("GetUser() UserName = %v, want %v", retrieved.UserName, "john.doe")
	}

	// 测试重复创建（应该失败）
	err = store.CreateUser(user)
	if err != model.ErrUniqueness {
		t.Errorf("CreateUser() duplicate error = %v, want %v", err, model.ErrUniqueness)
	}

	// 测试更新用户
	user.DisplayName = "John Doe Updated"
	err = store.UpdateUser(user)
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	retrieved, _ = store.GetUser("user-1")
	if retrieved.DisplayName != "John Doe Updated" {
		t.Errorf("UpdateUser() DisplayName = %v, want %v", retrieved.DisplayName, "John Doe Updated")
	}

	// 测试删除用户
	err = store.DeleteUser("user-1")
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	// 测试获取已删除用户
	_, err = store.GetUser("user-1")
	if err != model.ErrNotFound {
		t.Errorf("GetUser() after delete error = %v, want %v", err, model.ErrNotFound)
	}
}

func TestMemoryStore_GroupCRUD(t *testing.T) {
	store := NewMemory()

	// 测试创建组
	group := &model.Group{
		ID:          "group-1",
		DisplayName: "Admins",
	}

	err := store.CreateGroup(group)
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}

	// 测试获取组
	retrieved, err := store.GetGroup("group-1", false)
	if err != nil {
		t.Fatalf("GetGroup() error = %v", err)
	}
	if retrieved.DisplayName != "Admins" {
		t.Errorf("GetGroup() DisplayName = %v, want %v", retrieved.DisplayName, "Admins")
	}

	// 测试重复创建（应该失败）
	err = store.CreateGroup(group)
	if err != model.ErrUniqueness {
		t.Errorf("CreateGroup() duplicate error = %v, want %v", err, model.ErrUniqueness)
	}

	// 测试更新组
	group.DisplayName = "Administrators"
	err = store.UpdateGroup(group)
	if err != nil {
		t.Fatalf("UpdateGroup() error = %v", err)
	}

	retrieved, _ = store.GetGroup("group-1", false)
	if retrieved.DisplayName != "Administrators" {
		t.Errorf("UpdateGroup() DisplayName = %v, want %v", retrieved.DisplayName, "Administrators")
	}

	// 测试删除组
	err = store.DeleteGroup("group-1")
	if err != nil {
		t.Fatalf("DeleteGroup() error = %v", err)
	}

	// 测试获取已删除组
	_, err = store.GetGroup("group-1", false)
	if err != model.ErrNotFound {
		t.Errorf("GetGroup() after delete error = %v, want %v", err, model.ErrNotFound)
	}
}

func TestMemoryStore_ListUsers(t *testing.T) {
	store := NewMemory()

	// 创建测试用户
	users := []*model.User{
		{ID: "user-1", UserName: "alice", Active: true},
		{ID: "user-2", UserName: "bob", Active: false},
		{ID: "user-3", UserName: "charlie", Active: true},
	}

	for _, u := range users {
		u.Name.GivenName = u.UserName
		u.Name.FamilyName = "Test"
		store.CreateUser(u)
	}

	// 测试列表
	query := &model.ResourceQuery{
		StartIndex: 1,
		Count:      10,
	}

	list, total, err := store.ListUsers(query)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if total != 3 {
		t.Errorf("ListUsers() total = %v, want %v", total, 3)
	}
	if len(list) != 3 {
		t.Errorf("ListUsers() count = %v, want %v", len(list), 3)
	}

	// 测试分页
	query.Count = 2
	list, total, err = store.ListUsers(query)
	if err != nil {
		t.Fatalf("ListUsers() pagination error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListUsers() pagination count = %v, want %v", len(list), 2)
	}

	// 测试过滤器
	query = &model.ResourceQuery{
		Filter:     `userName eq "alice"`,
		StartIndex: 1,
		Count:      10,
	}
	list, total, err = store.ListUsers(query)
	if err != nil {
		t.Fatalf("ListUsers() filter error = %v", err)
	}
	if total != 1 {
		t.Errorf("ListUsers() filter total = %v, want %v", total, 1)
	}
	if len(list) != 1 || list[0].UserName != "alice" {
		t.Errorf("ListUsers() filter result = %v, want alice", list)
	}

	// 测试排序
	query = &model.ResourceQuery{
		StartIndex: 1,
		Count:      10,
		SortBy:     "userName",
		SortOrder:  "ascending",
	}
	list, _, _ = store.ListUsers(query)
	if len(list) > 0 && list[0].UserName != "alice" {
		t.Errorf("ListUsers() sort ascending first = %v, want alice", list[0].UserName)
	}

	query.SortOrder = "descending"
	list, _, _ = store.ListUsers(query)
	if len(list) > 0 && list[0].UserName != "charlie" {
		t.Errorf("ListUsers() sort descending first = %v, want charlie", list[0].UserName)
	}
}

func TestMemoryStore_ListGroups(t *testing.T) {
	store := NewMemory()

	// 创建测试组
	groups := []*model.Group{
		{ID: "group-1", DisplayName: "Admins"},
		{ID: "group-2", DisplayName: "Users"},
		{ID: "group-3", DisplayName: "Guests"},
	}

	for _, g := range groups {
		store.CreateGroup(g)
	}

	// 测试列表
	query := &model.ResourceQuery{
		StartIndex: 1,
		Count:      10,
	}

	list, total, err := store.ListGroups(query, false)
	if err != nil {
		t.Fatalf("ListGroups() error = %v", err)
	}
	if total != 3 {
		t.Errorf("ListGroups() total = %v, want %v", total, 3)
	}
	if len(list) != 3 {
		t.Errorf("ListGroups() count = %v, want %v", len(list), 3)
	}

	// 测试过滤器
	query = &model.ResourceQuery{
		Filter:     `displayName eq "Admins"`,
		StartIndex: 1,
		Count:      10,
	}
	list, total, err = store.ListGroups(query, false)
	if err != nil {
		t.Fatalf("ListGroups() filter error = %v", err)
	}
	if total != 1 {
		t.Errorf("ListGroups() filter total = %v, want %v", total, 1)
	}
	if len(list) != 1 || list[0].DisplayName != "Admins" {
		t.Errorf("ListGroups() filter result = %v, want Admins", list)
	}
}

func TestMemoryStore_PatchUser(t *testing.T) {
	store := NewMemory()

	// 创建用户
	user := &model.User{
		ID:       "user-1",
		UserName: "john.doe",
		Active:   true,
	}
	user.Name.GivenName = "John"
	user.Name.FamilyName = "Doe"
	store.CreateUser(user)

	// 测试替换操作
	ops := []model.PatchOperation{
		{
			Op:    "replace",
			Path:  "displayName",
			Value: "Johnny Doe",
		},
	}

	err := store.PatchUser("user-1", ops)
	if err != nil {
		t.Fatalf("PatchUser() error = %v", err)
	}

	retrieved, _ := store.GetUser("user-1")
	if retrieved.DisplayName != "Johnny Doe" {
		t.Errorf("PatchUser() DisplayName = %v, want %v", retrieved.DisplayName, "Johnny Doe")
	}

	// 测试不存在的用户
	err = store.PatchUser("nonexistent", ops)
	if err != model.ErrNotFound {
		t.Errorf("PatchUser() not found error = %v, want %v", err, model.ErrNotFound)
	}
}

func TestMemoryStore_PatchGroup(t *testing.T) {
	store := NewMemory()

	// 创建组
	group := &model.Group{
		ID:          "group-1",
		DisplayName: "Admins",
	}
	store.CreateGroup(group)

	// 测试替换操作
	ops := []model.PatchOperation{
		{
			Op:    "replace",
			Path:  "displayName",
			Value: "Administrators",
		},
	}

	err := store.PatchGroup("group-1", ops)
	if err != nil {
		t.Fatalf("PatchGroup() error = %v", err)
	}

	retrieved, _ := store.GetGroup("group-1", false)
	if retrieved.DisplayName != "Administrators" {
		t.Errorf("PatchGroup() DisplayName = %v, want %v", retrieved.DisplayName, "Administrators")
	}
}
