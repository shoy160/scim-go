package store

import (
	"testing"

	"scim-go/model"
	"scim-go/store"
)

func TestPatchUserEmails(t *testing.T) {
	store := store.NewMemory()

	user := &model.User{
		ID:       "test-user-1",
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
				UserID:  "test-user-1",
				Value:   "existing@example.com",
				Type:    "work",
				Primary: true,
			},
		},
	}

	if err := store.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("add email", func(t *testing.T) {
		ops := []model.PatchOperation{
			{
				Op:   "add",
				Path: "emails",
				Value: []any{
					map[string]any{
						"value":   "new@example.com",
						"type":    "home",
						"primary": false,
					},
				},
			},
		}

		if err := store.PatchUser("test-user-1", ops); err != nil {
			t.Errorf("Failed to add email: %v", err)
		}

		updated, err := store.GetUser("test-user-1")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if len(updated.Emails) != 2 {
			t.Errorf("Expected 2 emails, got %d", len(updated.Emails))
		}
	})

	t.Run("replace emails", func(t *testing.T) {
		ops := []model.PatchOperation{
			{
				Op:   "replace",
				Path: "emails",
				Value: []any{
					map[string]any{
						"value":   "replaced1@example.com",
						"type":    "work",
						"primary": true,
					},
					map[string]any{
						"value":   "replaced2@example.com",
						"type":    "home",
						"primary": false,
					},
				},
			},
		}

		if err := store.PatchUser("test-user-1", ops); err != nil {
			t.Errorf("Failed to replace emails: %v", err)
		}

		updated, err := store.GetUser("test-user-1")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if len(updated.Emails) != 2 {
			t.Errorf("Expected 2 emails, got %d", len(updated.Emails))
		}
	})

	t.Run("remove email", func(t *testing.T) {
		ops := []model.PatchOperation{
			{
				Op:   "remove",
				Path: "emails",
				Value: []any{
					map[string]any{
						"value": "replaced1@example.com",
					},
				},
			},
		}

		if err := store.PatchUser("test-user-1", ops); err != nil {
			t.Errorf("Failed to remove email: %v", err)
		}

		updated, err := store.GetUser("test-user-1")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if len(updated.Emails) != 1 {
			t.Errorf("Expected 1 email, got %d", len(updated.Emails))
		}
	})
}

func TestPatchUserRoles(t *testing.T) {
	store := store.NewMemory()

	user := &model.User{
		ID:       "test-user-2",
		UserName: "testuser2",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
			FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
		Roles: []model.Role{
			{
				UserID:  "test-user-2",
				Value:   "existing-role",
				Type:    "system",
				Primary: true,
			},
		},
	}

	if err := store.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("add role", func(t *testing.T) {
		ops := []model.PatchOperation{
			{
				Op:   "add",
				Path: "roles",
				Value: []any{
					map[string]any{
						"value":   "new-role",
						"type":    "custom",
						"primary": false,
					},
				},
			},
		}

		if err := store.PatchUser("test-user-2", ops); err != nil {
			t.Errorf("Failed to add role: %v", err)
		}

		updated, err := store.GetUser("test-user-2")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if len(updated.Roles) != 2 {
			t.Errorf("Expected 2 roles, got %d", len(updated.Roles))
		}
	})

	t.Run("replace roles", func(t *testing.T) {
		ops := []model.PatchOperation{
			{
				Op:   "replace",
				Path: "roles",
				Value: []any{
					map[string]any{
						"value":   "replaced-role-1",
						"type":    "system",
						"primary": true,
					},
					map[string]any{
						"value":   "replaced-role-2",
						"type":    "custom",
						"primary": false,
					},
				},
			},
		}

		if err := store.PatchUser("test-user-2", ops); err != nil {
			t.Errorf("Failed to replace roles: %v", err)
		}

		updated, err := store.GetUser("test-user-2")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if len(updated.Roles) != 2 {
			t.Errorf("Expected 2 roles, got %d", len(updated.Roles))
		}
	})

	t.Run("remove role", func(t *testing.T) {
		ops := []model.PatchOperation{
			{
				Op:   "remove",
				Path: "roles",
				Value: []any{
					map[string]any{
						"value": "replaced-role-1",
					},
				},
			},
		}

		if err := store.PatchUser("test-user-2", ops); err != nil {
			t.Errorf("Failed to remove role: %v", err)
		}

		updated, err := store.GetUser("test-user-2")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if len(updated.Roles) != 1 {
			t.Errorf("Expected 1 role, got %d", len(updated.Roles))
		}
	})
}

func TestUpdateUserEmails(t *testing.T) {
	store := store.NewMemory()

	user := &model.User{
		ID:       "test-user-3",
		UserName: "testuser3",
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
				UserID:  "test-user-3",
				Value:   "old@example.com",
				Type:    "work",
				Primary: true,
			},
		},
	}

	if err := store.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("replace emails", func(t *testing.T) {
		updatedUser := &model.User{
			ID:       "test-user-3",
			UserName: "testuser3",
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
					UserID:  "test-user-3",
					Value:   "new1@example.com",
					Type:    "work",
					Primary: true,
				},
				{
					UserID:  "test-user-3",
					Value:   "new2@example.com",
					Type:    "home",
					Primary: false,
				},
			},
		}

		if err := store.UpdateUser(updatedUser); err != nil {
			t.Errorf("Failed to update user: %v", err)
		}

		retrieved, err := store.GetUser("test-user-3")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if len(retrieved.Emails) != 2 {
			t.Errorf("Expected 2 emails, got %d", len(retrieved.Emails))
		}
	})
}

func TestUpdateUserRoles(t *testing.T) {
	store := store.NewMemory()

	user := &model.User{
		ID:       "test-user-4",
		UserName: "testuser4",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
			FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
		Roles: []model.Role{
			{
				UserID:  "test-user-4",
				Value:   "old-role",
				Type:    "system",
				Primary: true,
			},
		},
	}

	if err := store.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("replace roles", func(t *testing.T) {
		updatedUser := &model.User{
			ID:       "test-user-4",
			UserName: "testuser4",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
				FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
				MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
			}{
				GivenName:  "Test",
				FamilyName: "User",
			},
			Roles: []model.Role{
				{
					UserID:  "test-user-4",
					Value:   "new-role-1",
					Type:    "system",
					Primary: true,
				},
				{
					UserID:  "test-user-4",
					Value:   "new-role-2",
					Type:    "custom",
					Primary: false,
				},
			},
		}

		if err := store.UpdateUser(updatedUser); err != nil {
			t.Errorf("Failed to update user: %v", err)
		}

		retrieved, err := store.GetUser("test-user-4")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if len(retrieved.Roles) != 2 {
			t.Errorf("Expected 2 roles, got %d", len(retrieved.Roles))
		}
	})
}
