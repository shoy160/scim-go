package store

import (
	"testing"

	"scim-go/model"
	"scim-go/store"
)

func TestMemoryStore_GetUserGroups(t *testing.T) {
	store := store.NewMemory()

	// 创建测试用户
	user1 := &model.User{
		ID:       "user-1",
		UserName: "alice",
	}
	user1.Name.GivenName = "Alice"
	user1.Name.FamilyName = "Smith"
	store.CreateUser(user1)

	user2 := &model.User{
		ID:       "user-2",
		UserName: "bob",
	}
	user2.Name.GivenName = "Bob"
	user2.Name.FamilyName = "Jones"
	store.CreateUser(user2)

	// 创建测试组
	group1 := &model.Group{
		ID:          "group-1",
		DisplayName: "Engineering",
	}
	store.CreateGroup(group1)

	group2 := &model.Group{
		ID:          "group-2",
		DisplayName: "Product",
	}
	store.CreateGroup(group2)

	group3 := &model.Group{
		ID:          "group-3",
		DisplayName: "Marketing",
	}
	store.CreateGroup(group3)

	// 添加用户到组
	store.AddUserToGroup("group-1", "user-1")
	store.AddUserToGroup("group-2", "user-1")
	store.AddUserToGroup("group-1", "user-2")

	// 测试获取用户所属组
	t.Run("Get user groups - user in multiple groups", func(t *testing.T) {
		groups, err := store.GetUserGroups("user-1")
		if err != nil {
			t.Fatalf("GetUserGroups() error = %v", err)
		}
		if len(groups) != 2 {
			t.Errorf("Expected 2 groups, got %d", len(groups))
		}

		// 检查组信息
		groupMap := make(map[string]string)
		for _, g := range groups {
			groupMap[g.Value] = g.Display
		}

		if _, ok := groupMap["group-1"]; !ok {
			t.Error("User should be in group-1")
		}
		if _, ok := groupMap["group-2"]; !ok {
			t.Error("User should be in group-2")
		}
		if groupMap["group-1"] != "Engineering" {
			t.Errorf("Group-1 display name should be 'Engineering', got %s", groupMap["group-1"])
		}
	})

	t.Run("Get user groups - user in single group", func(t *testing.T) {
		groups, err := store.GetUserGroups("user-2")
		if err != nil {
			t.Fatalf("GetUserGroups() error = %v", err)
		}
		if len(groups) != 1 {
			t.Errorf("Expected 1 group, got %d", len(groups))
		}
		if groups[0].Value != "group-1" {
			t.Errorf("Expected group-1, got %s", groups[0].Value)
		}
	})

	t.Run("Get user groups - user in no groups", func(t *testing.T) {
		// 创建新用户，不添加到任何组
		user3 := &model.User{
			ID:       "user-3",
			UserName: "charlie",
		}
		user3.Name.GivenName = "Charlie"
		user3.Name.FamilyName = "Brown"
		store.CreateUser(user3)

		groups, err := store.GetUserGroups("user-3")
		if err != nil {
			t.Fatalf("GetUserGroups() error = %v", err)
		}
		if len(groups) != 0 {
			t.Errorf("Expected 0 groups, got %d", len(groups))
		}
	})

	t.Run("Get user groups - non-existent user", func(t *testing.T) {
		groups, err := store.GetUserGroups("non-existent")
		if err != nil {
			t.Fatalf("GetUserGroups() error = %v", err)
		}
		if len(groups) != 0 {
			t.Errorf("Expected 0 groups for non-existent user, got %d", len(groups))
		}
	})
}

func TestMemoryStore_GetUserGroupsAfterRemoval(t *testing.T) {
	store := store.NewMemory()

	// 创建测试数据
	user := &model.User{
		ID:       "user-1",
		UserName: "alice",
	}
	user.Name.GivenName = "Alice"
	user.Name.FamilyName = "Smith"
	store.CreateUser(user)

	group := &model.Group{
		ID:          "group-1",
		DisplayName: "Test Group",
	}
	store.CreateGroup(group)

	// 添加然后移除用户
	store.AddUserToGroup("group-1", "user-1")

	groups, _ := store.GetUserGroups("user-1")
	if len(groups) != 1 {
		t.Errorf("Expected 1 group before removal, got %d", len(groups))
	}

	store.RemoveUserFromGroup("group-1", "user-1")

	groups, _ = store.GetUserGroups("user-1")
	if len(groups) != 0 {
		t.Errorf("Expected 0 groups after removal, got %d", len(groups))
	}
}
