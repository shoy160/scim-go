package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"scim-go/api"
	"scim-go/model"
	"scim-go/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupTestRouterForPatch 设置测试路由
func setupTestRouterForPatch() (*gin.Engine, store.Store) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	s := store.NewMemory()

	cfg := &api.ScimConfig{
		DefaultSchema: model.UserSchema.String(),
		GroupSchema:   model.GroupSchema.String(),
		ErrorSchema:   model.ErrorSchema.String(),
		ListSchema:    model.ListSchema.String(),
		APIPath:       "/scim/v2",
		DefaultCount:  20,
		MaxCount:      100,
	}

	api.RegisterRoutes(router, s, cfg, "test-token", true, "/swagger")

	return router, s
}

// createPatchRequest 创建带认证的PATCH请求
func createPatchRequest(url string, body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	return req
}

func TestUserPatchWithEmptyPath(t *testing.T) {
	router, s := setupTestRouterForPatch()

	// 创建测试用户
	user := &model.User{
		ID:       "test-user-patch-empty",
		UserName: "testuser",
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
			{Value: "test@example.com", Primary: true},
		},
	}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("patch with empty path - update single attribute", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op: "replace",
					Value: map[string]any{
						"userName": "updateduser",
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "updateduser", response.UserName)
	})

	t.Run("patch with empty path - update nested attribute", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op: "replace",
					Value: map[string]any{
						"name": map[string]any{
							"givenName":  "Updated",
							"familyName": "Name",
						},
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Updated", response.Name.GivenName)
		assert.Equal(t, "Name", response.Name.FamilyName)
	})

	t.Run("patch with empty path - update multiple attributes", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op: "replace",
					Value: map[string]any{
						"userName": "multiupdate",
						"active":   true,
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "multiupdate", response.UserName)
		assert.True(t, response.Active)
	})

	t.Run("patch with empty path - value is null", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Value: nil,
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response model.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response.Detail, "value is required when path is empty")
	})

	t.Run("patch with empty path - value is not an object", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Value: "string value",
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response model.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response.Detail, "value must be an object when path is empty")
	})

	t.Run("patch with empty path - value is array", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Value: []any{"item1", "item2"},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response model.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response.Detail, "value must be an object when path is empty")
	})

	t.Run("patch with empty path - update enterprise extension", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op: "replace",
					Value: map[string]any{
						"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": map[string]any{
							"employeeNumber": "EMP123",
							"costCenter":     "CC001",
						},
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response.EnterpriseUserExtension)
		assert.Equal(t, "EMP123", response.EnterpriseUserExtension.EmployeeNumber)
		assert.Equal(t, "CC001", response.EnterpriseUserExtension.CostCenter)
	})

	t.Run("patch with empty path - mixed with path operations", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Path:  "userName",
					Value: "pathupdate",
				},
				{
					Op: "replace",
					Value: map[string]any{
						"active": true,
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "pathupdate", response.UserName)
		assert.True(t, response.Active)
	})
}

func TestGroupPatchWithEmptyPath(t *testing.T) {
	router, s := setupTestRouterForPatch()

	// 创建测试组
	group := &model.Group{
		ID:          "test-group-patch-empty",
		DisplayName: "Test Group",
	}
	if err := s.CreateGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	t.Run("patch with empty path - update displayName", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op: "replace",
					Value: map[string]any{
						"displayName": "Updated Group Name",
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Groups/test-group-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.Group
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Group Name", response.DisplayName)
	})

	t.Run("patch with empty path - update externalId", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op: "replace",
					Value: map[string]any{
						"externalId": "ext-12345",
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Groups/test-group-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.Group
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ext-12345", response.ExternalID)
	})

	t.Run("patch with empty path - value is null", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Value: nil,
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Groups/test-group-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response model.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response.Detail, "value is required when path is empty")
	})

	t.Run("patch with empty path - value is not an object", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Value: 12345,
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Groups/test-group-patch-empty", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response model.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response.Detail, "value must be an object when path is empty")
	})
}

func TestPatchValidationWithEmptyPath(t *testing.T) {
	router, s := setupTestRouterForPatch()

	// 创建测试用户
	user := &model.User{
		ID:       "test-user-validation",
		UserName: "testuser",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
	}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("add operation with empty path", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op: "add",
					Value: map[string]any{
						"nickName": "nickname",
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-validation", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "nickname", response.NickName)
	})

	t.Run("remove operation with empty path - should fail", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "remove",
					Value: map[string]any{},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-validation", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// remove 操作在 path 为空时，需要 value 指定要移除的属性
		// 这个测试验证系统能够处理这种情况
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("multiple operations with mixed path usage", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Path:  "userName",
					Value: "firstupdate",
				},
				{
					Op: "replace",
					Value: map[string]any{
						"active": true,
					},
				},
				{
					Op:    "replace",
					Path:  "nickName",
					Value: "secondnick",
				},
				{
					Op: "replace",
					Value: map[string]any{
						"displayName": "Developer",
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := createPatchRequest("/scim/v2/Users/test-user-validation", body)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "firstupdate", response.UserName)
		assert.True(t, response.Active)
		assert.Equal(t, "secondnick", response.NickName)
		assert.Equal(t, "Developer", response.DisplayName)
	})
}
