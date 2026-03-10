package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"scim-go/model"
)

func TestUserPatchEnterpriseExtension(t *testing.T) {
	router, s := setupTestRouter()

	// 创建带有企业扩展属性的用户
	user := &model.User{
		ID:       "test-user-enterprise-1",
		UserName: "testuser-enterprise",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
		EnterpriseUserExtension: &model.EnterpriseUserExtension{
			EmployeeNumber: "12345",
			CostCenter:     "CC001",
			Organization:   "Org1",
			Division:       "Div1",
			Department:     "Dept1",
			Manager: &model.Manager{
				Value: "manager123",
				Ref:   "http://example.com/Users/manager123",
			},
		},
	}

	if err := s.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("patch employeeNumber", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Path:  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.employeeNumber",
					Value: "67890",
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-enterprise-1", bytes.NewReader(body))
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

		if result.EnterpriseUserExtension == nil {
			t.Errorf("EnterpriseUserExtension should not be nil")
			return
		}

		if result.EnterpriseUserExtension.EmployeeNumber != "67890" {
			t.Errorf("Expected employeeNumber 67890, got %s", result.EnterpriseUserExtension.EmployeeNumber)
		}

		// 验证其他属性保持不变
		if result.EnterpriseUserExtension.CostCenter != "CC001" {
			t.Errorf("Expected costCenter CC001, got %s", result.EnterpriseUserExtension.CostCenter)
		}
	})

	t.Run("patch costCenter", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Path:  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.costCenter",
					Value: "CC002",
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-enterprise-1", bytes.NewReader(body))
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

		if result.EnterpriseUserExtension == nil {
			t.Errorf("EnterpriseUserExtension should not be nil")
			return
		}

		if result.EnterpriseUserExtension.CostCenter != "CC002" {
			t.Errorf("Expected costCenter CC002, got %s", result.EnterpriseUserExtension.CostCenter)
		}
	})

	t.Run("patch organization", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Path:  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.organization",
					Value: "Org2",
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-enterprise-1", bytes.NewReader(body))
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

		if result.EnterpriseUserExtension == nil {
			t.Errorf("EnterpriseUserExtension should not be nil")
			return
		}

		if result.EnterpriseUserExtension.Organization != "Org2" {
			t.Errorf("Expected organization Org2, got %s", result.EnterpriseUserExtension.Organization)
		}
	})

	t.Run("patch division", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Path:  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.division",
					Value: "Div2",
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-enterprise-1", bytes.NewReader(body))
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

		if result.EnterpriseUserExtension == nil {
			t.Errorf("EnterpriseUserExtension should not be nil")
			return
		}

		if result.EnterpriseUserExtension.Division != "Div2" {
			t.Errorf("Expected division Div2, got %s", result.EnterpriseUserExtension.Division)
		}
	})

	t.Run("patch department", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Path:  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.department",
					Value: "Dept2",
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-enterprise-1", bytes.NewReader(body))
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

		if result.EnterpriseUserExtension == nil {
			t.Errorf("EnterpriseUserExtension should not be nil")
			return
		}

		if result.EnterpriseUserExtension.Department != "Dept2" {
			t.Errorf("Expected department Dept2, got %s", result.EnterpriseUserExtension.Department)
		}
	})

	t.Run("patch manager", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:   "replace",
					Path: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.manager",
					Value: map[string]any{
						"value": "manager456",
						"$ref":  "http://example.com/Users/manager456",
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-enterprise-1", bytes.NewReader(body))
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

		if result.EnterpriseUserExtension == nil {
			t.Errorf("EnterpriseUserExtension should not be nil")
			return
		}

		if result.EnterpriseUserExtension.Manager == nil {
			t.Errorf("Manager should not be nil")
			return
		}

		if result.EnterpriseUserExtension.Manager.Value != "manager456" {
			t.Errorf("Expected manager value manager456, got %s", result.EnterpriseUserExtension.Manager.Value)
		}
	})

	t.Run("patch multiple enterprise attributes", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:    "replace",
					Path:  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.employeeNumber",
					Value: "99999",
				},
				{
					Op:    "replace",
					Path:  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.costCenter",
					Value: "CC999",
				},
				{
					Op:   "replace",
					Path: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.manager",
					Value: map[string]any{
						"value": "manager999",
						"$ref":  "http://example.com/Users/manager999",
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-enterprise-1", bytes.NewReader(body))
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

		if result.EnterpriseUserExtension == nil {
			t.Errorf("EnterpriseUserExtension should not be nil")
			return
		}

		if result.EnterpriseUserExtension.EmployeeNumber != "99999" {
			t.Errorf("Expected employeeNumber 99999, got %s", result.EnterpriseUserExtension.EmployeeNumber)
		}

		if result.EnterpriseUserExtension.CostCenter != "CC999" {
			t.Errorf("Expected costCenter CC999, got %s", result.EnterpriseUserExtension.CostCenter)
		}

		if result.EnterpriseUserExtension.Manager == nil {
			t.Errorf("Manager should not be nil")
			return
		}

		if result.EnterpriseUserExtension.Manager.Value != "manager999" {
			t.Errorf("Expected manager value manager999, got %s", result.EnterpriseUserExtension.Manager.Value)
		}
	})

	t.Run("remove enterprise attribute", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op:   "remove",
					Path: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.costCenter",
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-enterprise-1", bytes.NewReader(body))
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

		if result.EnterpriseUserExtension == nil {
			t.Errorf("EnterpriseUserExtension should not be nil")
			return
		}

		if result.EnterpriseUserExtension.CostCenter != "" {
			t.Errorf("Expected costCenter to be empty, got %s", result.EnterpriseUserExtension.CostCenter)
		}
	})
}

func TestUserPatchEnterpriseExtensionWithEmptyPath(t *testing.T) {
	router, s := setupTestRouter()

	// 创建带有企业扩展属性的用户
	user := &model.User{
		ID:       "test-user-enterprise-2",
		UserName: "testuser-enterprise-2",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Test",
			FamilyName: "User",
		},
		EnterpriseUserExtension: &model.EnterpriseUserExtension{
			EmployeeNumber: "11111",
			CostCenter:     "CC111",
		},
	}

	if err := s.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("patch with empty path - replace entire enterprise extension", func(t *testing.T) {
		patchReq := model.PatchRequest{
			Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			Operations: []model.PatchOperation{
				{
					Op: "replace",
					Value: map[string]any{
						"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": map[string]any{
							"employeeNumber": "22222",
							"costCenter":     "CC222",
							"organization":   "Org222",
						},
					},
				},
			},
		}

		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest("PATCH", "/scim/v2/Users/test-user-enterprise-2", bytes.NewReader(body))
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

		if result.EnterpriseUserExtension == nil {
			t.Errorf("EnterpriseUserExtension should not be nil")
			return
		}

		if result.EnterpriseUserExtension.EmployeeNumber != "22222" {
			t.Errorf("Expected employeeNumber 22222, got %s", result.EnterpriseUserExtension.EmployeeNumber)
		}

		if result.EnterpriseUserExtension.CostCenter != "CC222" {
			t.Errorf("Expected costCenter CC222, got %s", result.EnterpriseUserExtension.CostCenter)
		}

		if result.EnterpriseUserExtension.Organization != "Org222" {
			t.Errorf("Expected organization Org222, got %s", result.EnterpriseUserExtension.Organization)
		}
	})
}
