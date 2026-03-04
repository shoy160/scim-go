package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"scim-go/model"
	"testing"
)

func TestGroupListAttributesMembersMetaCheck(t *testing.T) {
	router, _ := setupTestRouter()

	group := model.Group{
		DisplayName: "MetaTestGroup",
	}
	body, _ := json.Marshal(group)
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create group: %d", w.Code)
	}

	var createdGroup model.Group
	json.Unmarshal(w.Body.Bytes(), &createdGroup)

	t.Run("List Groups with attributes=members should have meta child properties", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups?attributes=members", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("List Groups status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok {
			t.Error("Resources should be an array")
			return
		}

		if len(resources) == 0 {
			t.Error("Resources should not be empty")
			return
		}

		groupMap, ok := resources[0].(map[string]interface{})
		if !ok {
			t.Error("Resource should be a map")
			return
		}

		meta, ok := groupMap["meta"].(map[string]interface{})
		if !ok {
			t.Error("Group should have meta field when attributes=members")
			return
		}

		expectedMetaProps := []string{"resourceType", "created", "lastModified", "location", "version"}
		for _, prop := range expectedMetaProps {
			if _, exists := meta[prop]; !exists {
				t.Errorf("meta.%s should exist when attributes=members, got meta: %+v", prop, meta)
			}
			if meta[prop] == "" {
				t.Errorf("meta.%s should not be empty when attributes=members, got: %v", prop, meta[prop])
			}
		}
	})

	t.Run("List Groups without attributes should have meta child properties", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("List Groups status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok {
			t.Error("Resources should be an array")
			return
		}

		groupMap, ok := resources[0].(map[string]interface{})
		if !ok {
			t.Error("Resource should be a map")
			return
		}

		meta, ok := groupMap["meta"].(map[string]interface{})
		if !ok {
			t.Error("Group should have meta field")
			return
		}

		expectedMetaProps := []string{"resourceType", "created", "lastModified", "location", "version"}
		for _, prop := range expectedMetaProps {
			if _, exists := meta[prop]; !exists {
				t.Errorf("meta.%s should exist, got meta: %+v", prop, meta)
			}
		}
	})

	t.Run("List Groups with attributes=id,members should have meta child properties", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups?attributes=id,members", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("List Groups status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok {
			t.Error("Resources should be an array")
			return
		}

		groupMap, ok := resources[0].(map[string]interface{})
		if !ok {
			t.Error("Resource should be a map")
			return
		}

		meta, ok := groupMap["meta"].(map[string]interface{})
		if !ok {
			t.Error("Group should have meta field when attributes includes id")
			return
		}

		expectedMetaProps := []string{"resourceType", "created", "lastModified", "location", "version"}
		for _, prop := range expectedMetaProps {
			if _, exists := meta[prop]; !exists {
				t.Errorf("meta.%s should exist when attributes=id,members, got meta: %+v", prop, meta)
			}
		}
	})
}
