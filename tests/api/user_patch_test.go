package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"scim-go/model"
)

func TestUserPatchEmails(t *testing.T) {
	router, s := setupTestRouter()

	user := &model.User{
		ID:       "test-user-1",
		UserName: "testuser1",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
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

	if err := s.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("add email", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
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
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-1", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result model.User
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(result.Emails) != 2 {
			t.Errorf("Expected 2 emails, got %d", len(result.Emails))
		}
	})

	t.Run("replace emails", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
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
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-1", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result model.User
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(result.Emails) != 2 {
			t.Errorf("Expected 2 emails, got %d", len(result.Emails))
		}
	})

	t.Run("remove email", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:   "remove",
					Path: "emails",
					Value: []any{
						map[string]any{
							"value": "replaced1@example.com",
						},
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-1", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result model.User
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(result.Emails) != 1 {
			t.Errorf("Expected 1 email, got %d", len(result.Emails))
		}
	})
}

func TestUserPatchRoles(t *testing.T) {
	router, s := setupTestRouter()

	user := &model.User{
		ID:       "test-user-2",
		UserName: "testuser2",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
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

	if err := s.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("add role", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
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
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-2", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result model.User
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(result.Roles) != 2 {
			t.Errorf("Expected 2 roles, got %d", len(result.Roles))
		}
	})

	t.Run("replace roles", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
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
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-2", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result model.User
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(result.Roles) != 2 {
			t.Errorf("Expected 2 roles, got %d", len(result.Roles))
		}
	})

	t.Run("remove role", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:   "remove",
					Path: "roles",
					Value: []any{
						map[string]any{
							"value": "replaced-role-1",
						},
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-2", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result model.User
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(result.Roles) != 1 {
			t.Errorf("Expected 1 role, got %d", len(result.Roles))
		}
	})
}

func TestUserPutEmails(t *testing.T) {
	router, s := setupTestRouter()

	user := &model.User{
		ID:       "test-user-3",
		UserName: "testuser3",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
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

	if err := s.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("replace emails", func(t *testing.T) {
		updateUser := &model.User{
			ID:       "test-user-3",
			UserName: "testuser3",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
				FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
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

		body, _ := json.Marshal(updateUser)
		req := httptest.NewRequest("PUT", "/scim/v2/Users/test-user-3", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result model.User
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(result.Emails) != 2 {
			t.Errorf("Expected 2 emails, got %d", len(result.Emails))
		}
	})
}

func TestUserPutRoles(t *testing.T) {
	router, s := setupTestRouter()

	user := &model.User{
		ID:       "test-user-4",
		UserName: "testuser4",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
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

	if err := s.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("replace roles", func(t *testing.T) {
		updateUser := &model.User{
			ID:       "test-user-4",
			UserName: "testuser4",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
				FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
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

		body, _ := json.Marshal(updateUser)
		req := httptest.NewRequest("PUT", "/scim/v2/Users/test-user-4", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result model.User
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(result.Roles) != 2 {
			t.Errorf("Expected 2 roles, got %d", len(result.Roles))
		}
	})
}
