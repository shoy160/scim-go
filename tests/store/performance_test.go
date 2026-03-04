package store

import (
	"fmt"
	"scim-go/model"
	"scim-go/store"
	"testing"
)

// BenchmarkUpdateUserWithEmailsAndRoles 测试用户更新性能
// 比较增量更新与暴力删除再重建的性能差异
func BenchmarkUpdateUserWithEmailsAndRoles(b *testing.B) {
	// 创建内存存储实例用于测试
	memoryStore := store.NewMemory()

	// 创建测试用户
	user := &model.User{
		ID:       "test-user",
		UserName: "testuser",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
			FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
		Emails: []model.Email{
			{
				UserID:  "test-user",
				Value:   "user1@example.com",
				Type:    "work",
				Primary: true,
			},
			{
				UserID:  "test-user",
				Value:   "user2@example.com",
				Type:    "home",
				Primary: false,
			},
		},
		Roles: []model.Role{
			{
				UserID:  "test-user",
				Value:   "role1",
				Type:    "system",
				Primary: true,
			},
			{
				UserID:  "test-user",
				Value:   "role2",
				Type:    "custom",
				Primary: false,
			},
		},
	}

	// 创建用户
	if err := memoryStore.CreateUser(user); err != nil {
		b.Fatalf("Failed to create user: %v", err)
	}

	// 测试场景 1: 只有少量变更（添加一个邮箱和一个角色）
	b.Run("SmallChanges", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			updatedUser := &model.User{
				ID:       "test-user",
				UserName: "testuser",
				Name: struct {
					Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
					GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
					FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
					MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
				}{
					GivenName:  "Test",
					FamilyName: "User",
				},
				Emails: []model.Email{
					{
						UserID:  "test-user",
						Value:   "user1@example.com",
						Type:    "work",
						Primary: true,
					},
					{
						UserID:  "test-user",
						Value:   "user2@example.com",
						Type:    "home",
						Primary: false,
					},
					{
						UserID:  "test-user",
						Value:   "user3@example.com",
						Type:    "other",
						Primary: false,
					},
				},
				Roles: []model.Role{
					{
						UserID:  "test-user",
						Value:   "role1",
						Type:    "system",
						Primary: true,
					},
					{
						UserID:  "test-user",
						Value:   "role2",
						Type:    "custom",
						Primary: false,
					},
					{
						UserID:  "test-user",
						Value:   "role3",
						Type:    "custom",
						Primary: false,
					},
				},
			}
			if err := memoryStore.UpdateUser(updatedUser); err != nil {
				b.Fatalf("Failed to update user: %v", err)
			}
		}
	})

	// 测试场景 2: 大量变更（完全替换邮箱和角色）
	b.Run("LargeChanges", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			updatedUser := &model.User{
				ID:       "test-user",
				UserName: "testuser",
				Name: struct {
					Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
					GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
					FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
					MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
				}{
					GivenName:  "Test",
					FamilyName: "User",
				},
				Emails: []model.Email{
					{
						UserID:  "test-user",
						Value:   "newuser1@example.com",
						Type:    "work",
						Primary: true,
					},
					{
						UserID:  "test-user",
						Value:   "newuser2@example.com",
						Type:    "home",
						Primary: false,
					},
					{
						UserID:  "test-user",
						Value:   "newuser3@example.com",
						Type:    "other",
						Primary: false,
					},
					{
						UserID:  "test-user",
						Value:   "newuser4@example.com",
						Type:    "other",
						Primary: false,
					},
				},
				Roles: []model.Role{
					{
						UserID:  "test-user",
						Value:   "newrole1",
						Type:    "system",
						Primary: true,
					},
					{
						UserID:  "test-user",
						Value:   "newrole2",
						Type:    "custom",
						Primary: false,
					},
					{
						UserID:  "test-user",
						Value:   "newrole3",
						Type:    "custom",
						Primary: false,
					},
					{
						UserID:  "test-user",
						Value:   "newrole4",
						Type:    "custom",
						Primary: false,
					},
				},
			}
			if err := memoryStore.UpdateUser(updatedUser); err != nil {
				b.Fatalf("Failed to update user: %v", err)
			}
		}
	})
}

// TestIncrementalUpdateConsistency 测试增量更新的一致性
func TestIncrementalUpdateConsistency(t *testing.T) {
	// 创建内存存储实例用于测试
	memoryStore := store.NewMemory()

	// 创建测试用户
	user := &model.User{
		ID:       "test-user",
		UserName: "testuser",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
			FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
		Emails: []model.Email{
			{
				UserID:  "test-user",
				Value:   "user1@example.com",
				Type:    "work",
				Primary: true,
			},
		},
		Roles: []model.Role{
			{
				UserID:  "test-user",
				Value:   "role1",
				Type:    "system",
				Primary: true,
			},
		},
	}

	// 创建用户
	if err := memoryStore.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 测试场景 1: 添加新邮箱和角色
	updatedUser1 := &model.User{
		ID:       "test-user",
		UserName: "testuser",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
			FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
		Emails: []model.Email{
			{
				UserID:  "test-user",
				Value:   "user1@example.com",
				Type:    "work",
				Primary: true,
			},
			{
				UserID:  "test-user",
				Value:   "user2@example.com",
				Type:    "home",
				Primary: false,
			},
		},
		Roles: []model.Role{
			{
				UserID:  "test-user",
				Value:   "role1",
				Type:    "system",
				Primary: true,
			},
			{
				UserID:  "test-user",
				Value:   "role2",
				Type:    "custom",
				Primary: false,
			},
		},
	}

	if err := memoryStore.UpdateUser(updatedUser1); err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// 验证更新结果
	resultUser, err := memoryStore.GetUser("test-user")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if len(resultUser.Emails) != 2 {
		t.Errorf("Expected 2 emails, got %d", len(resultUser.Emails))
	}
	if len(resultUser.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(resultUser.Roles))
	}

	// 测试场景 2: 删除邮箱和角色
	updatedUser2 := &model.User{
		ID:       "test-user",
		UserName: "testuser",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
			FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
		Emails: []model.Email{
			{
				UserID:  "test-user",
				Value:   "user1@example.com",
				Type:    "work",
				Primary: true,
			},
		},
		Roles: []model.Role{
			{
				UserID:  "test-user",
				Value:   "role1",
				Type:    "system",
				Primary: true,
			},
		},
	}

	if err := memoryStore.UpdateUser(updatedUser2); err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// 验证更新结果
	resultUser2, err := memoryStore.GetUser("test-user")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if len(resultUser2.Emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(resultUser2.Emails))
	}
	if len(resultUser2.Roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(resultUser2.Roles))
	}

	// 测试场景 3: 完全替换邮箱和角色
	updatedUser3 := &model.User{
		ID:       "test-user",
		UserName: "testuser",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
			FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
		Emails: []model.Email{
			{
				UserID:  "test-user",
				Value:   "newuser@example.com",
				Type:    "work",
				Primary: true,
			},
		},
		Roles: []model.Role{
			{
				UserID:  "test-user",
				Value:   "newrole",
				Type:    "system",
				Primary: true,
			},
		},
	}

	if err := memoryStore.UpdateUser(updatedUser3); err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// 验证更新结果
	resultUser3, err := memoryStore.GetUser("test-user")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if len(resultUser3.Emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(resultUser3.Emails))
	}
	if len(resultUser3.Roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(resultUser3.Roles))
	}
	if resultUser3.Emails[0].Value != "newuser@example.com" {
		t.Errorf("Expected email 'newuser@example.com', got '%s'", resultUser3.Emails[0].Value)
	}
	if resultUser3.Roles[0].Value != "newrole" {
		t.Errorf("Expected role 'newrole', got '%s'", resultUser3.Roles[0].Value)
	}

	fmt.Println("Incremental update consistency test passed!")
}
