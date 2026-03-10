package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"scim-go/model"
	"testing"
)

func TestUserMetaDataCompleteness(t *testing.T) {
	router, _ := setupTestRouter()

	user := model.User{
		UserName: "metatest",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Meta",
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
		t.Fatalf("Failed to create user: %d, body: %s", w.Code, w.Body.String())
	}

	var createdUser model.User
	json.Unmarshal(w.Body.Bytes(), &createdUser)

	t.Logf("Created user meta: %+v", createdUser.Meta)

	t.Run("Get User - meta should be complete", func(t *testing.T) {
		getReq := httptest.NewRequest(http.MethodGet, "/scim/v2/Users/"+createdUser.ID, nil)
		getReq.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, getReq)

		if w.Code != http.StatusOK {
			t.Errorf("Get User status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.User
		json.Unmarshal(w.Body.Bytes(), &response)

		t.Logf("Retrieved user meta: %+v", response.Meta)

		meta := response.Meta
		if meta.ResourceType == "" {
			t.Error("meta.resourceType should not be empty")
		}
		if meta.Location == "" {
			t.Error("meta.location should not be empty")
		}
		if meta.Created == "" {
			t.Error("meta.created should not be empty")
		}
		if meta.LastModified == "" {
			t.Error("meta.lastModified should not be empty")
		}
		if meta.Version == "" {
			t.Error("meta.version should not be empty")
		}
	})

	t.Run("List Users - meta should be complete", func(t *testing.T) {
		listReq := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?startIndex=1&count=10", nil)
		listReq.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, listReq)

		if w.Code != http.StatusOK {
			t.Errorf("List Users status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok || len(resources) == 0 {
			t.Fatal("Resources should not be empty")
		}

		userMap := resources[0].(map[string]interface{})
		
		metaJSON, _ := json.Marshal(userMap["meta"])
		t.Logf("List user meta JSON: %s", metaJSON)

		meta, ok := userMap["meta"].(map[string]interface{})
		if !ok {
			t.Fatal("User should have meta field")
		}

		if meta["resourceType"] == "" {
			t.Error("meta.resourceType should not be empty")
		}
		if meta["location"] == "" {
			t.Error("meta.location should not be empty")
		}
		if meta["created"] == "" {
			t.Error("meta.created should not be empty")
		}
		if meta["lastModified"] == "" {
			t.Error("meta.lastModified should not be empty")
		}
		if meta["version"] == "" {
			t.Error("meta.version should not be empty")
		}
	})
}
