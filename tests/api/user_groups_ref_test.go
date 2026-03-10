package api

import (
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

func setupTestRouterWithGroups() (*gin.Engine, store.Store) {
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

func TestUserGroupsRefField(t *testing.T) {
	router, s := setupTestRouterWithGroups()

	user := &model.User{
		ID:       "test-user-1",
		UserName: "testuser",
	}
	err := s.CreateUser(user)
	assert.NoError(t, err)

	group1 := &model.Group{
		ID:          "group-1",
		DisplayName: "Test Group 1",
	}
	err = s.CreateGroup(group1)
	assert.NoError(t, err)

	group2 := &model.Group{
		ID:          "group-2",
		DisplayName: "Test Group 2",
	}
	err = s.CreateGroup(group2)
	assert.NoError(t, err)

	err = s.AddMemberToGroup("group-1", "test-user-1", "User")
	assert.NoError(t, err)

	err = s.AddMemberToGroup("group-2", "test-user-1", "User")
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users/test-user-1?attributes=groups.*", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.User
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Len(t, response.Groups, 2)

	for _, group := range response.Groups {
		assert.NotEmpty(t, group.Ref, "Group %s should have $ref field", group.Value)
		assert.Contains(t, group.Ref, "/Groups/"+group.Value, "Group $ref should contain /Groups/{id}")
	}
}

func TestUserListGroupsRefField(t *testing.T) {
	router, s := setupTestRouterWithGroups()

	user := &model.User{
		ID:       "test-user-2",
		UserName: "testuser2",
	}
	err := s.CreateUser(user)
	assert.NoError(t, err)

	group := &model.Group{
		ID:          "group-list-1",
		DisplayName: "Test Group List 1",
	}
	err = s.CreateGroup(group)
	assert.NoError(t, err)

	err = s.AddMemberToGroup("group-list-1", "test-user-2", "User")
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?startIndex=1&count=10&attributes=userName,groups.*", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Resources []model.User `json:"Resources"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Len(t, response.Resources, 1)

	for _, user := range response.Resources {
		for _, group := range user.Groups {
			assert.NotEmpty(t, group.Ref, "Group %s should have $ref field", group.Value)
			assert.Contains(t, group.Ref, "/Groups/"+group.Value, "Group $ref should contain /Groups/{id}")
		}
	}
}
