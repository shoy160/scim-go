package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetGroupMembers 测试获取组成员接口
func TestGetGroupMembers(t *testing.T) {
	router, _ := setupTestRouter()

	// 创建测试用户1
	user1 := map[string]interface{}{
		"userName": "user1",
		"name": map[string]string{
			"givenName":  "User",
			"familyName": "One",
		},
		"active": true,
	}
	body1, _ := json.Marshal(user1)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scim/v2/Users", bytes.NewBuffer(body1))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create user1: %d", w.Code)
	}
	var user1Resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &user1Resp)
	user1ID := user1Resp["id"].(string)

	// 创建测试用户2
	user2 := map[string]interface{}{
		"userName": "user2",
		"name": map[string]string{
			"givenName":  "User",
			"familyName": "Two",
		},
		"active": true,
	}
	body2, _ := json.Marshal(user2)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/scim/v2/Users", bytes.NewBuffer(body2))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create user2: %d", w.Code)
	}

	// 创建测试组
	group := map[string]interface{}{
		"displayName": "Test Group",
	}
	groupBody, _ := json.Marshal(group)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/scim/v2/Groups", bytes.NewBuffer(groupBody))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create group: %d", w.Code)
	}

	// 获取组ID
	var groupResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &groupResp)
	groupID := groupResp["id"].(string)

	t.Run("Get empty members", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Groups/"+groupID+"/members", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["totalResults"] != float64(0) {
			t.Errorf("Expected 0 members, got %v", resp["totalResults"])
		}
	})

	// 添加用户1到组
	addReq := map[string]interface{}{
		"value": user1ID,
		"type":  "User",
	}
	addBody, _ := json.Marshal(addReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/scim/v2/Groups/"+groupID+"/members", bytes.NewBuffer(addBody))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to add user to group: %d, body: %s", w.Code, w.Body.String())
	}

	t.Run("Get members after adding user", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Groups/"+groupID+"/members", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["totalResults"] != float64(1) {
			t.Errorf("Expected 1 member, got %v", resp["totalResults"])
		}
	})

	t.Run("Filter by User type", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Groups/"+groupID+"?memberType=User", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)
	})

	t.Run("Pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Groups/"+groupID+"/members?startIndex=1&count=10", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Get members from non-existent group", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/scim/v2/Groups/non-existent/members", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}
