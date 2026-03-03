package store

import (
	"testing"

	"scim-go/model"
)

func TestMemoryStore_AddUserToGroup(t *testing.T) {
	store := NewMemory()

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
	group := &model.Group{
		ID:          "group-1",
		DisplayName: "Test Group",
	}
	store.CreateGroup(group)

	// 测试添加用户到组
	t.Run("Add user to group", func(t *testing.T) {
		err := store.AddUserToGroup("group-1", "user-1")
		if err != nil {
			t.Fatalf("AddUserToGroup() error = %v", err)
		}

		// 验证用户已添加
		inGroup, _ := store.IsUserInGroup("group-1", "user-1")
		if !inGroup {
			t.Error("User should be in group")
		}

		// 验证组包含该成员
		group, _ := store.GetGroup("group-1", false)
		if len(group.Members) != 1 {
			t.Errorf("Group should have 1 member, got %d", len(group.Members))
		}
		if group.Members[0].Value != "user-1" {
			t.Errorf("Member value should be user-1, got %s", group.Members[0].Value)
		}
	})

	// 测试添加多个用户
	t.Run("Add multiple users to group", func(t *testing.T) {
		err := store.AddUserToGroup("group-1", "user-2")
		if err != nil {
			t.Fatalf("AddUserToGroup() error = %v", err)
		}

		group, _ := store.GetGroup("group-1", false)
		if len(group.Members) != 2 {
			t.Errorf("Group should have 2 members, got %d", len(group.Members))
		}
	})

	// 测试重复添加
	t.Run("Add duplicate user to group", func(t *testing.T) {
		err := store.AddUserToGroup("group-1", "user-1")
		if err == nil {
			t.Error("Should return error for duplicate user")
		}
		if err.Error() != "user already in group" {
			t.Errorf("Expected 'user already in group' error, got %v", err)
		}
	})

	// 测试添加到不存在的组
	t.Run("Add user to non-existent group", func(t *testing.T) {
		err := store.AddUserToGroup("nonexistent", "user-1")
		if err != model.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	// 测试添加不存在的用户
	t.Run("Add non-existent user to group", func(t *testing.T) {
		err := store.AddUserToGroup("group-1", "nonexistent")
		if err != model.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}

func TestMemoryStore_RemoveUserFromGroup(t *testing.T) {
	store := NewMemory()

	// 创建测试数据
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

	group := &model.Group{
		ID:          "group-1",
		DisplayName: "Test Group",
	}
	store.CreateGroup(group)

	// 添加用户到组
	store.AddUserToGroup("group-1", "user-1")
	store.AddUserToGroup("group-1", "user-2")

	// 测试移除用户
	t.Run("Remove user from group", func(t *testing.T) {
		err := store.RemoveUserFromGroup("group-1", "user-1")
		if err != nil {
			t.Fatalf("RemoveUserFromGroup() error = %v", err)
		}

		// 验证用户已移除
		inGroup, _ := store.IsUserInGroup("group-1", "user-1")
		if inGroup {
			t.Error("User should not be in group")
		}

		// 验证其他用户仍在组中
		inGroup, _ = store.IsUserInGroup("group-1", "user-2")
		if !inGroup {
			t.Error("Other user should still be in group")
		}

		group, _ := store.GetGroup("group-1", false)
		if len(group.Members) != 1 {
			t.Errorf("Group should have 1 member, got %d", len(group.Members))
		}
	})

	// 测试移除不存在的用户
	t.Run("Remove non-existent user from group", func(t *testing.T) {
		err := store.RemoveUserFromGroup("group-1", "nonexistent")
		if err == nil {
			t.Error("Should return error for non-existent user")
		}
		if err.Error() != "user not in group" {
			t.Errorf("Expected 'user not in group' error, got %v", err)
		}
	})

	// 测试从不存在的组移除
	t.Run("Remove user from non-existent group", func(t *testing.T) {
		err := store.RemoveUserFromGroup("nonexistent", "user-2")
		if err != model.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}

func TestMemoryStore_IsUserInGroup(t *testing.T) {
	store := NewMemory()

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

	// 测试用户不在组中
	t.Run("User not in group", func(t *testing.T) {
		inGroup, err := store.IsUserInGroup("group-1", "user-1")
		if err != nil {
			t.Fatalf("IsUserInGroup() error = %v", err)
		}
		if inGroup {
			t.Error("User should not be in group initially")
		}
	})

	// 添加用户到组
	store.AddUserToGroup("group-1", "user-1")

	// 测试用户在组中
	t.Run("User in group", func(t *testing.T) {
		inGroup, err := store.IsUserInGroup("group-1", "user-1")
		if err != nil {
			t.Fatalf("IsUserInGroup() error = %v", err)
		}
		if !inGroup {
			t.Error("User should be in group after adding")
		}
	})

	// 测试不存在的组
	t.Run("Check user in non-existent group", func(t *testing.T) {
		_, err := store.IsUserInGroup("nonexistent", "user-1")
		if err != model.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}

func TestMemoryStore_CreateGroupWithMembers(t *testing.T) {
	store := NewMemory()

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

	// 创建带成员的组
	group := &model.Group{
		ID:          "group-1",
		DisplayName: "Test Group",
		Members: []model.Member{
			{Value: "user-1"},
			{Value: "user-2"},
		},
	}

	err := store.CreateGroup(group)
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}

	// 验证成员已添加
	retrieved, _ := store.GetGroup("group-1", false)
	if len(retrieved.Members) != 2 {
		t.Errorf("Group should have 2 members, got %d", len(retrieved.Members))
	}

	// 验证用户在组中
	inGroup1, _ := store.IsUserInGroup("group-1", "user-1")
	inGroup2, _ := store.IsUserInGroup("group-1", "user-2")
	if !inGroup1 || !inGroup2 {
		t.Error("Both users should be in group")
	}
}

func TestMemoryStore_GroupMemberIntegrity(t *testing.T) {
	store := NewMemory()

	// 创建测试数据
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

	group1 := &model.Group{
		ID:          "group-1",
		DisplayName: "Group 1",
	}
	store.CreateGroup(group1)

	group2 := &model.Group{
		ID:          "group-2",
		DisplayName: "Group 2",
	}
	store.CreateGroup(group2)

	// 将用户添加到两个组
	store.AddUserToGroup("group-1", "user-1")
	store.AddUserToGroup("group-2", "user-1")
	store.AddUserToGroup("group-1", "user-2")

	// 验证用户在两个组中
	inGroup1_1, _ := store.IsUserInGroup("group-1", "user-1")
	inGroup2_1, _ := store.IsUserInGroup("group-2", "user-1")
	if !inGroup1_1 || !inGroup2_1 {
		t.Error("User should be in both groups")
	}

	// 从一个组中移除用户
	store.RemoveUserFromGroup("group-1", "user-1")

	// 验证用户已从第一个组移除，但仍存在于第二个组
	inGroup1_1, _ = store.IsUserInGroup("group-1", "user-1")
	inGroup2_1, _ = store.IsUserInGroup("group-2", "user-1")
	if inGroup1_1 {
		t.Error("User should not be in group-1 after removal")
	}
	if !inGroup2_1 {
		t.Error("User should still be in group-2")
	}

	// 验证其他用户不受影响
	inGroup1_2, _ := store.IsUserInGroup("group-1", "user-2")
	if !inGroup1_2 {
		t.Error("Other user should still be in group-1")
	}
}
