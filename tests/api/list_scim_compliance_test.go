package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"scim-go/model"

	"github.com/gin-gonic/gin"
)

// TestListUsersSCIMCompliance 测试用户列表接口的 SCIM 2.0 合规性
func TestListUsersSCIMCompliance(t *testing.T) {
	router, _ := setupTestRouter()

	// 准备测试数据
	createComplianceTestUsers(router, t)

	// 测试场景1: 基本列表请求 - 验证响应结构
	t.Run("Basic List Request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 验证状态码
		if w.Code != http.StatusOK {
			t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
		}

		// 验证响应结构
		var response model.ListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}

		// 验证 Schema
		if len(response.Schemas) == 0 || response.Schemas[0] != "urn:ietf:params:scim:api:messages:2.0:ListResponse" {
			t.Error("Invalid schemas in response")
		}

		// 验证必要字段
		if response.TotalResults < 0 {
			t.Error("TotalResults should be >= 0")
		}
		if response.StartIndex < 1 {
			t.Error("StartIndex should be >= 1")
		}
		if response.ItemsPerPage < 0 {
			t.Error("ItemsPerPage should be >= 0")
		}
		if response.Resources == nil {
			t.Error("Resources should not be nil")
		}
	})

	// 测试场景2: 分页参数 - 验证 startIndex 和 count
	t.Run("Pagination Parameters", func(t *testing.T) {
		testCases := []struct {
			name       string
			startIndex int
			count      int
			expectedOK bool
		}{
			{"Valid pagination", 1, 5, true},
			{"StartIndex 0 (should default to 1)", 0, 5, true},
			{"Negative count (should default)", 1, -1, true},
			{"Large count (should be limited)", 1, 1000, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				url := "/scim/v2/Users?startIndex=" + strconv.Itoa(tc.startIndex) + "&count=" + strconv.Itoa(tc.count)
				req := httptest.NewRequest(http.MethodGet, url, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if tc.expectedOK && w.Code != http.StatusOK {
					t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
				}
			})
		}
	})

	// 测试场景3: 过滤功能 - 验证 filter 参数
	t.Run("Filter Functionality", func(t *testing.T) {
		testCases := []struct {
			name     string
			filter   string
			expected bool // 是否期望成功
		}{
			{"Valid equality filter", "userName eq \"user1\"", true},
			{"Valid presence filter", "userName pr", true},
			{"Valid contains filter", "userName co \"user\"", true},
			{"Invalid filter syntax", "userName eq", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// 正确构建包含filter参数的URL
				var url string
				if tc.filter == "userName eq \"user1\"" {
					url = "/scim/v2/Users?filter=userName+eq+%22user1%22"
				} else if tc.filter == "userName pr" {
					url = "/scim/v2/Users?filter=userName+pr"
				} else if tc.filter == "userName co \"user\"" {
					url = "/scim/v2/Users?filter=userName+co+%22user%22"
				} else {
					url = "/scim/v2/Users?filter=invalid"
				}
				req := httptest.NewRequest(http.MethodGet, url, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if tc.expected {
					if w.Code != http.StatusOK {
						t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
					}
				} else {
					if w.Code != http.StatusBadRequest {
						t.Errorf("Status code = %v, want %v", w.Code, http.StatusBadRequest)
					}
				}
			})
		}
	})

	// 测试场景4: 排序功能 - 验证 sortBy 和 sortOrder
	t.Run("Sort Functionality", func(t *testing.T) {
		testCases := []struct {
			name      string
			sortBy    string
			sortOrder string
			expected  bool
		}{
			{"Valid sort by userName", "userName", "ascending", true},
			{"Valid sort by userName descending", "userName", "descending", true},
			{"Invalid sortOrder", "userName", "invalid", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				url := "/scim/v2/Users?sortBy=" + tc.sortBy + "&sortOrder=" + tc.sortOrder
				req := httptest.NewRequest(http.MethodGet, url, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if tc.expected {
					if w.Code != http.StatusOK {
						t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
					}
				} else {
					if w.Code != http.StatusBadRequest {
						t.Errorf("Status code = %v, want %v", w.Code, http.StatusBadRequest)
					}
				}
			})
		}
	})

	// 测试场景5: 属性选择 - 验证 attributes 和 excludedAttributes
	t.Run("Attribute Selection", func(t *testing.T) {
		testCases := []struct {
			name               string
			attributes         string
			excludedAttributes string
			expectedOK         bool
		}{
			{"Only id and userName", "id,userName", "", true},
			{"Exclude name", "", "name", true},
			{"Both attributes and excludedAttributes", "id,userName", "userName", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				url := "/scim/v2/Users"
				params := []string{}
				if tc.attributes != "" {
					params = append(params, "attributes="+tc.attributes)
				}
				if tc.excludedAttributes != "" {
					params = append(params, "excludedAttributes="+tc.excludedAttributes)
				}
				if len(params) > 0 {
					url += "?" + strings.Join(params, "&")
				}

				req := httptest.NewRequest(http.MethodGet, url, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if tc.expectedOK && w.Code != http.StatusOK {
					t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
				}
			})
		}
	})

	// 测试场景6: 综合参数 - 验证多个参数组合
	t.Run("Combined Parameters", func(t *testing.T) {
		url := "/scim/v2/Users?startIndex=1&count=5&filter=active%20eq%20true&sortBy=userName&sortOrder=ascending&attributes=id,userName,active"
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
		}
	})

	// 测试场景7: 错误处理 - 验证无效请求
	t.Run("Error Handling", func(t *testing.T) {
		testCases := []struct {
			name         string
			url          string
			expectedCode int
		}{
			{"Invalid filter syntax", "/scim/v2/Users?filter=invalid", http.StatusBadRequest},
			{"Invalid sortOrder", "/scim/v2/Users?sortOrder=invalid", http.StatusBadRequest},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, tc.url, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != tc.expectedCode {
					t.Errorf("Status code = %v, want %v", w.Code, tc.expectedCode)
				}

				// 验证错误响应结构
				var errorResp model.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
					return
				}

				if errorResp.Schemas != "urn:ietf:params:scim:api:messages:2.0:Error" {
					t.Error("Invalid error schema")
				}
				if errorResp.Status != tc.expectedCode {
					t.Errorf("Error status = %v, want %v", errorResp.Status, tc.expectedCode)
				}
			})
		}
	})
}

// TestListGroupsSCIMCompliance 测试组列表接口的 SCIM 2.0 合规性
func TestListGroupsSCIMCompliance(t *testing.T) {
	router, _ := setupTestRouter()

	// 准备测试数据
	createComplianceTestGroups(router, t)

	// 测试场景1: 基本列表请求 - 验证响应结构
	t.Run("Basic List Request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 验证状态码
		if w.Code != http.StatusOK {
			t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
		}

		// 验证响应结构
		var response model.ListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}

		// 验证 Schema
		if len(response.Schemas) == 0 || response.Schemas[0] != "urn:ietf:params:scim:api:messages:2.0:ListResponse" {
			t.Error("Invalid schemas in response")
		}

		// 验证必要字段
		if response.TotalResults < 0 {
			t.Error("TotalResults should be >= 0")
		}
		if response.StartIndex < 1 {
			t.Error("StartIndex should be >= 1")
		}
		if response.ItemsPerPage < 0 {
			t.Error("ItemsPerPage should be >= 0")
		}
		if response.Resources == nil {
			t.Error("Resources should not be nil")
		}
	})

	// 测试场景2: 分页参数 - 验证 startIndex 和 count
	t.Run("Pagination Parameters", func(t *testing.T) {
		testCases := []struct {
			name       string
			startIndex int
			count      int
			expectedOK bool
		}{
			{"Valid pagination", 1, 5, true},
			{"StartIndex 0 (should default to 1)", 0, 5, true},
			{"Negative count (should default)", 1, -1, true},
			{"Large count (should be limited)", 1, 1000, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				url := "/scim/v2/Groups?startIndex=" + strconv.Itoa(tc.startIndex) + "&count=" + strconv.Itoa(tc.count)
				req := httptest.NewRequest(http.MethodGet, url, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if tc.expectedOK && w.Code != http.StatusOK {
					t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
				}
			})
		}
	})

	// 测试场景3: 过滤功能 - 验证 filter 参数
	t.Run("Filter Functionality", func(t *testing.T) {
		testCases := []struct {
			name     string
			filter   string
			expected bool // 是否期望成功
		}{
			{"Valid equality filter", "displayName eq \"Group1\"", true},
			{"Valid presence filter", "displayName pr", true},
			{"Invalid filter syntax", "displayName eq", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// 正确构建包含filter参数的URL
				var url string
				if tc.filter == "displayName eq \"Group1\"" {
					url = "/scim/v2/Groups?filter=displayName+eq+%22Group1%22"
				} else if tc.filter == "displayName pr" {
					url = "/scim/v2/Groups?filter=displayName+pr"
				} else {
					url = "/scim/v2/Groups?filter=invalid"
				}
				req := httptest.NewRequest(http.MethodGet, url, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if tc.expected {
					if w.Code != http.StatusOK {
						t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
					}
				} else {
					if w.Code != http.StatusBadRequest {
						t.Errorf("Status code = %v, want %v", w.Code, http.StatusBadRequest)
					}
				}
			})
		}
	})

	// 测试场景4: 属性选择 - 验证 attributes 和 excludedAttributes
	t.Run("Attribute Selection", func(t *testing.T) {
		testCases := []struct {
			name               string
			attributes         string
			excludedAttributes string
			expectedOK         bool
		}{
			{"Only id and displayName", "id,displayName", "", true},
			{"Exclude members", "", "members", true},
			{"Include members", "id,displayName,members", "", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				url := "/scim/v2/Groups"
				params := []string{}
				if tc.attributes != "" {
					params = append(params, "attributes="+tc.attributes)
				}
				if tc.excludedAttributes != "" {
					params = append(params, "excludedAttributes="+tc.excludedAttributes)
				}
				if len(params) > 0 {
					url += "?" + strings.Join(params, "&")
				}

				req := httptest.NewRequest(http.MethodGet, url, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if tc.expectedOK && w.Code != http.StatusOK {
					t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
				}
			})
		}
	})

	// 测试场景5: 错误处理 - 验证无效请求
	t.Run("Error Handling", func(t *testing.T) {
		testCases := []struct {
			name         string
			url          string
			expectedCode int
		}{
			{"Invalid filter syntax", "/scim/v2/Groups?filter=invalid", http.StatusBadRequest},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, tc.url, nil)
				req.Header.Set("Authorization", "Bearer test-token")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != tc.expectedCode {
					t.Errorf("Status code = %v, want %v", w.Code, tc.expectedCode)
				}

				// 验证错误响应结构
				var errorResp model.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
					return
				}

				if errorResp.Schemas != "urn:ietf:params:scim:api:messages:2.0:Error" {
					t.Error("Invalid error schema")
				}
				if errorResp.Status != tc.expectedCode {
					t.Errorf("Error status = %v, want %v", errorResp.Status, tc.expectedCode)
				}
			})
		}
	})
}

// createComplianceTestUsers 创建测试用户
func createComplianceTestUsers(router *gin.Engine, t *testing.T) {
	users := []model.User{
		{
			UserName: "user1",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
				FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
				MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
			}{
				GivenName:  "John",
				FamilyName: "Doe",
			},
			Active: true,
		},
		{
			UserName: "user2",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
				FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
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
				GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
				FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
				MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
			}{
				GivenName:  "Bob",
				FamilyName: "Johnson",
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

		if w.Code != http.StatusCreated {
			t.Errorf("Create user status = %v, want %v", w.Code, http.StatusCreated)
		}
	}
}

// createComplianceTestGroups 创建测试组
func createComplianceTestGroups(router *gin.Engine, t *testing.T) {
	groups := []model.Group{
		{
			DisplayName: "Group1",
		},
		{
			DisplayName: "Group2",
		},
		{
			DisplayName: "Group3",
		},
	}

	for _, group := range groups {
		body, _ := json.Marshal(group)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Create group status = %v, want %v", w.Code, http.StatusCreated)
		}
	}
}
