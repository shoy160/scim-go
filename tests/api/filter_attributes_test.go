package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"scim-go/model"

	"github.com/gin-gonic/gin"
)

func TestUserFilterAttributes(t *testing.T) {
	router, _ := setupTestRouter()

	// 创建测试用户
	createTestUsers(router, t)

	// 测试场景1: 基础查询
	t.Run("Basic User Query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Basic User Query status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.ListResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.TotalResults < 3 {
			t.Errorf("Basic User Query total results = %v, want at least 3", response.TotalResults)
		}
	})

	// 测试场景2: 带filter条件的查询
	t.Run("User Filter Query", func(t *testing.T) {
		testCases := []struct {
			name     string
			filter   string
			expected int
		}{
			{"Filter by userName", "userName eq \"user1\"", 1},
			{"Filter by active true", "active eq true", 2},
			{"Filter by active false", "active eq false", 1},
			{"Filter by familyName", "name.familyName eq \"Smith\"", 2},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?filter="+url.QueryEscape(tc.filter), nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Filter Query status = %v, want %v", w.Code, http.StatusOK)
				}

				var response model.ListResponse
				json.Unmarshal(w.Body.Bytes(), &response)
				if response.TotalResults != tc.expected {
					t.Errorf("Filter Query total results = %v, want %v", response.TotalResults, tc.expected)
				}
			})
		}
	})

	// 测试场景3: 指定返回字段的查询
	t.Run("User Attributes Query", func(t *testing.T) {
		testCases := []struct {
			name       string
			attributes string
			expected   []string
		}{
			{"Only id and userName", "id,userName", []string{"id", "userName"}},
			{"Only name", "name", []string{"name"}},
			{"Only active", "active", []string{"active"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?attributes="+tc.attributes, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Attributes Query status = %v, want %v", w.Code, http.StatusOK)
				}

				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)

				resources, ok := response["Resources"].([]interface{})
				if !ok {
					t.Error("Resources should be an array")
					return
				}

				for _, resource := range resources {
					userMap, ok := resource.(map[string]interface{})
					if !ok {
						t.Error("Resource should be a map")
						continue
					}

					// 检查预期字段是否存在
					for _, expectedField := range tc.expected {
						if _, ok := userMap[expectedField]; !ok {
							t.Errorf("User should have %s field when attributes=%s", expectedField, tc.attributes)
						}
					}
				}
			})
		}
	})

	// 测试场景4: 排除特定字段的查询
	t.Run("User ExcludedAttributes Query", func(t *testing.T) {
		testCases := []struct {
			name               string
			excludedAttributes string
			notExpected        []string
		}{
			{"Exclude name", "name", []string{"name"}},
			{"Exclude active", "active", []string{"active"}},
			{"Exclude name and active", "name,active", []string{"name", "active"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?excludedAttributes="+tc.excludedAttributes, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("ExcludedAttributes Query status = %v, want %v", w.Code, http.StatusOK)
				}

				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)

				resources, ok := response["Resources"].([]interface{})
				if !ok {
					t.Error("Resources should be an array")
					return
				}

				for _, resource := range resources {
					userMap, ok := resource.(map[string]interface{})
					if !ok {
						t.Error("Resource should be a map")
						continue
					}

					// 检查排除字段是否不存在
					for _, notExpectedField := range tc.notExpected {
						if _, ok := userMap[notExpectedField]; ok {
							t.Errorf("User should not have %s field when excludedAttributes=%s", notExpectedField, tc.excludedAttributes)
						}
					}
				}
			})
		}
	})

	// 测试场景5: 组合查询（filter + attributes）
	t.Run("User Combined Query", func(t *testing.T) {
		query := url.Values{}
		query.Add("filter", "active eq true")
		query.Add("attributes", "id,userName")
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?"+query.Encode(), nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Combined Query status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok {
			t.Error("Resources should be an array")
			return
		}

		// 应该只返回active=true的用户，且只包含id和userName字段
		for _, resource := range resources {
			userMap, ok := resource.(map[string]interface{})
			if !ok {
				t.Error("Resource should be a map")
				continue
			}

			if _, ok := userMap["id"]; !ok {
				t.Error("User should have id field")
			}
			if _, ok := userMap["userName"]; !ok {
				t.Error("User should have userName field")
			}
			if _, ok := userMap["active"]; ok {
				t.Error("User should not have active field when attributes=id,userName")
			}
		}
	})

	// 测试场景6: 边界条件查询
	t.Run("User Boundary Conditions", func(t *testing.T) {
		testCases := []struct {
			name  string
			query string
			code  int
		}{
			{"Empty filter", "/scim/v2/Users?filter=", http.StatusOK},
			{"Empty attributes", "/scim/v2/Users?attributes=", http.StatusOK},
			{"Empty excludedAttributes", "/scim/v2/Users?excludedAttributes=", http.StatusOK},
			{"Invalid filter syntax", "/scim/v2/Users?filter=invalid", http.StatusBadRequest},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, tc.query, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != tc.code {
					t.Errorf("%s status = %v, want %v", tc.name, w.Code, tc.code)
				}
			})
		}
	})
}

func TestGroupFilterAttributes(t *testing.T) {
	router, _ := setupTestRouter()

	// 创建测试组
	createTestGroups(router, t)

	// 测试场景1: 基础查询
	t.Run("Basic Group Query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Basic Group Query status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.ListResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.TotalResults < 3 {
			t.Errorf("Basic Group Query total results = %v, want at least 3", response.TotalResults)
		}
	})

	// 测试场景2: 带filter条件的查询
	t.Run("Group Filter Query", func(t *testing.T) {
		testCases := []struct {
			name     string
			filter   string
			expected int
		}{
			{"Filter by displayName", "displayName eq \"Group 1\"", 1},
			{"Filter by displayName contains", "displayName co \"Group\"", 3},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups?filter="+url.QueryEscape(tc.filter), nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Filter Query status = %v, want %v", w.Code, http.StatusOK)
				}

				var response model.ListResponse
				json.Unmarshal(w.Body.Bytes(), &response)
				if response.TotalResults != tc.expected {
					t.Errorf("Filter Query total results = %v, want %v", response.TotalResults, tc.expected)
				}
			})
		}
	})

	// 测试场景3: 指定返回字段的查询
	t.Run("Group Attributes Query", func(t *testing.T) {
		testCases := []struct {
			name       string
			attributes string
			expected   []string
		}{
			{"Only id and displayName", "id,displayName", []string{"id", "displayName"}},
			{"Only members (should return all fields)", "members", []string{"id", "displayName", "members", "schemas", "meta"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups?attributes="+tc.attributes, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Attributes Query status = %v, want %v", w.Code, http.StatusOK)
				}

				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)

				resources, ok := response["Resources"].([]interface{})
				if !ok {
					t.Error("Resources should be an array")
					return
				}

				for _, resource := range resources {
					groupMap, ok := resource.(map[string]interface{})
					if !ok {
						t.Error("Resource should be a map")
						continue
					}

					// 检查预期字段是否存在
					for _, expectedField := range tc.expected {
						if _, ok := groupMap[expectedField]; !ok {
							t.Errorf("Group should have %s field when attributes=%s, got: %v", expectedField, tc.attributes, groupMap)
						}
					}
				}
			})
		}
	})

	// 测试场景4: 排除特定字段的查询
	t.Run("Group ExcludedAttributes Query", func(t *testing.T) {
		testCases := []struct {
			name               string
			excludedAttributes string
			notExpected        []string
		}{
			{"Exclude members", "members", []string{"members"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups?excludedAttributes="+tc.excludedAttributes, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("ExcludedAttributes Query status = %v, want %v", w.Code, http.StatusOK)
				}

				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)

				resources, ok := response["Resources"].([]interface{})
				if !ok {
					t.Error("Resources should be an array")
					return
				}

				for _, resource := range resources {
					groupMap, ok := resource.(map[string]interface{})
					if !ok {
						t.Error("Resource should be a map")
						continue
					}

					// 检查排除字段是否不存在
					for _, notExpectedField := range tc.notExpected {
						if _, ok := groupMap[notExpectedField]; ok {
							t.Errorf("Group should not have %s field when excludedAttributes=%s", notExpectedField, tc.excludedAttributes)
						}
					}
				}
			})
		}
	})

	// 测试场景5: 组合查询（filter + attributes）
	t.Run("Group Combined Query", func(t *testing.T) {
		query := url.Values{}
		query.Add("filter", "displayName co \"Group\"")
		query.Add("attributes", "id,displayName")
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups?"+query.Encode(), nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Combined Query status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok {
			t.Error("Resources should be an array")
			return
		}

		// 应该只返回包含"Group"的组，且只包含id和displayName字段
		for _, resource := range resources {
			groupMap, ok := resource.(map[string]interface{})
			if !ok {
				t.Error("Resource should be a map")
				continue
			}

			if _, ok := groupMap["id"]; !ok {
				t.Error("Group should have id field")
			}
			if _, ok := groupMap["displayName"]; !ok {
				t.Error("Group should have displayName field")
			}
			if _, ok := groupMap["members"]; ok {
				t.Error("Group should not have members field when attributes=id,displayName")
			}
		}
	})

	// 测试场景6: 边界条件查询
	t.Run("Group Boundary Conditions", func(t *testing.T) {
		testCases := []struct {
			name  string
			query string
			code  int
		}{
			{"Empty filter", "/scim/v2/Groups?filter=", http.StatusOK},
			{"Empty attributes", "/scim/v2/Groups?attributes=", http.StatusOK},
			{"Empty excludedAttributes", "/scim/v2/Groups?excludedAttributes=", http.StatusOK},
			{"Invalid filter syntax", "/scim/v2/Groups?filter=invalid", http.StatusBadRequest},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, tc.query, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != tc.code {
					t.Errorf("%s status = %v, want %v", tc.name, w.Code, tc.code)
				}
			})
		}
	})
}

// 辅助函数：创建测试用户
func createTestUsers(router *gin.Engine, t *testing.T) {
	users := []model.User{
		{
			UserName: "user1",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
				FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
				MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
			}{
				GivenName:  "John",
				FamilyName: "Smith",
			},
			Active: true,
		},
		{
			UserName: "user2",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
				FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
				MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
			}{
				GivenName:  "Jane",
				FamilyName: "Smith",
			},
			Active: true,
		},
		{
			UserName: "user3",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
				FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
				MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
			}{
				GivenName:  "Bob",
				FamilyName: "Jones",
			},
			Active: false,
		},
	}

	for _, user := range users {
		body, _ := json.Marshal(user)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// 辅助函数：创建测试组
func createTestGroups(router *gin.Engine, t *testing.T) {
	groups := []model.Group{
		{DisplayName: "Group 1"},
		{DisplayName: "Group 2"},
		{DisplayName: "Group 3"},
	}

	for _, group := range groups {
		body, _ := json.Marshal(group)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
