package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"scim-go/model"
	"testing"
)

func TestUserListCacheMetaCheck(t *testing.T) {
	router, _ := setupTestRouter()

	user := model.User{
		UserName: "cachetest",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
			FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Cache",
			FamilyName: "Test",
		},
	}
	body, _ := json.Marshal(user)
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create user: %d", w.Code)
	}

	t.Run("First request - meta should be complete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?attributes=userName,meta", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("First request status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok || len(resources) == 0 {
			t.Fatal("Resources should not be empty")
		}

		userMap := resources[0].(map[string]interface{})
		meta, ok := userMap["meta"].(map[string]interface{})
		if !ok {
			t.Fatal("User should have meta field")
		}

		expectedMetaProps := []string{"resourceType", "created", "lastModified", "location", "version"}
		for _, prop := range expectedMetaProps {
			if _, exists := meta[prop]; !exists {
				t.Errorf("First request: meta.%s should exist, got meta: %+v", prop, meta)
			}
			if meta[prop] == "" {
				t.Errorf("First request: meta.%s should not be empty, got: %v", prop, meta[prop])
			}
		}
	})

	t.Run("Second request (cached) - meta should still be complete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?attributes=userName,meta", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Second request status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok || len(resources) == 0 {
			t.Fatal("Resources should not be empty")
		}

		userMap := resources[0].(map[string]interface{})
		meta, ok := userMap["meta"].(map[string]interface{})
		if !ok {
			t.Fatal("User should have meta field in second request")
		}

		expectedMetaProps := []string{"resourceType", "created", "lastModified", "location", "version"}
		for _, prop := range expectedMetaProps {
			if _, exists := meta[prop]; !exists {
				t.Errorf("Second request (cached): meta.%s should exist, got meta: %+v", prop, meta)
			}
			if meta[prop] == "" {
				t.Errorf("Second request (cached): meta.%s should not be empty, got: %v", prop, meta[prop])
			}
		}
	})
}
