package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAttributeValidationBeforeQuery 验证属性格式验证在数据查询之前执行
// 这个测试确保当提供无效的属性格式时，系统不会查询数据库，而是直接返回400错误
func TestAttributeValidationBeforeQuery(t *testing.T) {
	router, store := setupTestRouter()

	// 创建测试用户
	user := map[string]interface{}{
		"userName": "test.user",
		"name": map[string]string{
			"givenName":  "Test",
			"familyName": "User",
		},
		"active": true,
	}
	body, _ := json.Marshal(user)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scim/v2/Users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test user: %d", w.Code)
	}

	// 创建测试组
	group := map[string]interface{}{
		"displayName": "Test Group",
	}
	body, _ = json.Marshal(group)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/scim/v2/Groups", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test group: %d", w.Code)
	}

	t.Run("Invalid attributes format in List Users", func(t *testing.T) {
		// 使用无效的属性格式（包含非法字符@）
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Users?attributes=invalid@attr", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		// 应该返回400错误，而不是查询数据库
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid attributes format, got %d", w.Code)
		}
	})

	t.Run("Invalid excludedAttributes format in List Users", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Users?excludedAttributes=invalid@attr", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid excludedAttributes format, got %d", w.Code)
		}
	})

	t.Run("Invalid attributes format in Get User", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Users/test-user?attributes=invalid@attr", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid attributes format, got %d", w.Code)
		}
	})

	t.Run("Invalid attributes format in List Groups", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Groups?attributes=invalid@attr", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid attributes format, got %d", w.Code)
		}
	})

	t.Run("Invalid attributes format in Get Group", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Groups/test-group?attributes=invalid@attr", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid attributes format, got %d", w.Code)
		}
	})

	t.Run("Valid attributes format should work", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Users?attributes=id,userName", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for valid attributes format, got %d", w.Code)
		}
	})

	// 验证数据库查询次数（可选，用于确认验证在查询之前）
	_ = store // 如果需要验证存储层的调用次数，可以使用mock store
}

// TestAttributeValidationWithWildcard 测试通配符属性的验证
func TestAttributeValidationWithWildcard(t *testing.T) {
	router, _ := setupTestRouter()

	// 创建测试用户
	user := map[string]interface{}{
		"userName": "wildcard.test",
		"name": map[string]string{
			"givenName":  "Wildcard",
			"familyName": "Test",
		},
		"active": true,
	}
	body, _ := json.Marshal(user)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scim/v2/Users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test user: %d", w.Code)
	}

	t.Run("Valid wildcard attributes", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Users?attributes=groups.*", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for valid wildcard attributes, got %d", w.Code)
		}
	})

	t.Run("Valid nested attributes", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Users?attributes=name.givenName,name.familyName", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for valid nested attributes, got %d", w.Code)
		}
	})

	t.Run("Invalid wildcard format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Users?attributes=groups.*.invalid", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid wildcard format, got %d", w.Code)
		}
	})
}

// TestAttributeValidationErrorResponse 测试属性验证错误的响应格式
func TestAttributeValidationErrorResponse(t *testing.T) {
	router, _ := setupTestRouter()

	t.Run("Error response format for invalid attributes", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Users?attributes=invalid@attr", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		// 验证错误响应包含SCIM标准的schemas字段
		body := w.Body.String()
		if !contains(body, "urn:ietf:params:scim:api:messages:2.0:Error") {
			t.Errorf("Error response should contain SCIM error schema, got: %s", body)
		}

		// 验证错误响应包含scimType字段
		if !contains(body, "invalidSyntax") {
			t.Errorf("Error response should contain scimType 'invalidSyntax', got: %s", body)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
