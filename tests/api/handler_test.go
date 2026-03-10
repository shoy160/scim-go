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
)

func setupTestRouter() (*gin.Engine, store.Store) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
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

	api.RegisterRoutes(r, s, cfg, "test-token", true, "/swagger")
	return r, s
}

func TestUserHandlers(t *testing.T) {
	router, _ := setupTestRouter()

	t.Run("Create User", func(t *testing.T) {
		user := model.User{
			UserName: "john.doe",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
				FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
				MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
			}{
				GivenName:  "John",
				FamilyName: "Doe",
			},
			Active: true,
		}

		body, _ := json.Marshal(user)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("CreateUser() status = %v, want %v", w.Code, http.StatusCreated)
		}

		var response model.User
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.UserName != "john.doe" {
			t.Errorf("CreateUser() userName = %v, want %v", response.UserName, "john.doe")
		}
		if response.ID == "" {
			t.Error("CreateUser() ID should not be empty")
		}
	})

	t.Run("Create User Without Required Fields", func(t *testing.T) {
		user := model.User{
			UserName: "",
		}

		body, _ := json.Marshal(user)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("CreateUser() without required fields status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("Get User", func(t *testing.T) {
		// 先创建用户
		user := model.User{
			UserName: "jane.doe",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
				FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
				MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
			}{
				GivenName:  "Jane",
				FamilyName: "Doe",
			},
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var created model.User
		json.Unmarshal(w.Body.Bytes(), &created)

		// 获取用户
		req = httptest.NewRequest(http.MethodGet, "/scim/v2/Users/"+created.ID, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetUser() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.User
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.UserName != "jane.doe" {
			t.Errorf("GetUser() userName = %v, want %v", response.UserName, "jane.doe")
		}
	})

	t.Run("Get Non-existent User", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users/nonexistent", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("GetUser() non-existent status = %v, want %v", w.Code, http.StatusNotFound)
		}
	})

	t.Run("List Users", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListUsers() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.ListResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.Schemas[0] != model.ListSchema.String() {
			t.Errorf("ListUsers() schema = %v, want %v", response.Schemas[0], model.ListSchema.String())
		}
	})

	t.Run("Delete User", func(t *testing.T) {
		// 先创建用户
		user := model.User{
			UserName: "delete.me",
			Name: struct {
				Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
				GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
				FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
				MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
			}{
				GivenName:  "Delete",
				FamilyName: "Me",
			},
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var created model.User
		json.Unmarshal(w.Body.Bytes(), &created)

		// 删除用户
		req = httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/"+created.ID, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("DeleteUser() status = %v, want %v", w.Code, http.StatusNoContent)
		}

		// 验证用户已删除
		req = httptest.NewRequest(http.MethodGet, "/scim/v2/Users/"+created.ID, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("GetUser() after delete status = %v, want %v", w.Code, http.StatusNotFound)
		}
	})
}

func TestGroupHandlers(t *testing.T) {
	router, _ := setupTestRouter()

	t.Run("Create Group", func(t *testing.T) {
		group := model.Group{
			DisplayName: "Test Group",
		}

		body, _ := json.Marshal(group)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("CreateGroup() status = %v, want %v", w.Code, http.StatusCreated)
		}

		var response model.Group
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.DisplayName != "Test Group" {
			t.Errorf("CreateGroup() displayName = %v, want %v", response.DisplayName, "Test Group")
		}
	})

	t.Run("Create Group Without Required Fields", func(t *testing.T) {
		group := model.Group{
			DisplayName: "",
		}

		body, _ := json.Marshal(group)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("CreateGroup() without required fields status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("Get Group", func(t *testing.T) {
		// 先创建组
		group := model.Group{
			DisplayName: "My Group",
		}
		body, _ := json.Marshal(group)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var created model.Group
		json.Unmarshal(w.Body.Bytes(), &created)

		// 获取组
		req = httptest.NewRequest(http.MethodGet, "/scim/v2/Groups/"+created.ID, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetGroup() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.Group
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.DisplayName != "My Group" {
			t.Errorf("GetGroup() displayName = %v, want %v", response.DisplayName, "My Group")
		}
	})

	t.Run("List Groups", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListGroups() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.ListResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.Schemas[0] != model.ListSchema.String() {
			t.Errorf("ListGroups() schema = %v, want %v", response.Schemas[0], model.ListSchema.String())
		}
	})
}

func TestAuthentication(t *testing.T) {
	router, _ := setupTestRouter()

	t.Run("Request Without Auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
		// 不设置 Authorization 头

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Request without auth status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("Request With Invalid Auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Request with invalid auth status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})
}

func TestServiceProviderConfig(t *testing.T) {
	router, _ := setupTestRouter()

	t.Run("Get ServiceProviderConfig", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ServiceProviderConfig() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.ServiceProviderConfig
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.Schemas[0] != model.ServiceProviderConfigSchema {
			t.Errorf("ServiceProviderConfig() schema = %v, want %v", response.Schemas[0], model.ServiceProviderConfigSchema)
		}
	})
}

func TestResourceTypes(t *testing.T) {
	router, _ := setupTestRouter()

	t.Run("List ResourceTypes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/ResourceTypes", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("List ResourceTypes status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.ListResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.TotalResults != 2 {
			t.Errorf("List ResourceTypes total = %v, want %v", response.TotalResults, 2)
		}
	})

	t.Run("Get User ResourceType", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/ResourceTypes/User", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Get User ResourceType status = %v, want %v", w.Code, http.StatusOK)
		}

		var response model.ResourceType
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.ID != "User" {
			t.Errorf("ResourceType ID = %v, want %v", response.ID, "User")
		}
	})
}

func TestSchemas(t *testing.T) {
	router, _ := setupTestRouter()

	t.Run("List Schemas", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Schemas", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("List Schemas status = %v, want %v", w.Code, http.StatusOK)
		}
	})

	t.Run("Get User Schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Schemas/urn:ietf:params:scim:schemas:core:2.0:User", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Get User Schema status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestGroupMemberManagement(t *testing.T) {
	router, store := setupTestRouter()

	// 创建测试用户
	user1 := model.User{
		UserName: "alice",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Alice",
			FamilyName: "Smith",
		},
		Active: true,
	}

	user2 := model.User{
		UserName: "bob",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Bob",
			FamilyName: "Jones",
		},
		Active: true,
	}

	// 创建用户
	body, _ := json.Marshal(user1)
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createdUser1 model.User
	json.Unmarshal(w.Body.Bytes(), &createdUser1)

	body, _ = json.Marshal(user2)
	req = httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createdUser2 model.User
	json.Unmarshal(w.Body.Bytes(), &createdUser2)

	// 创建测试组
	group := model.Group{
		DisplayName: "Test Group",
	}

	body, _ = json.Marshal(group)
	req = httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createdGroup model.Group
	json.Unmarshal(w.Body.Bytes(), &createdGroup)

	t.Run("Create Group with Members", func(t *testing.T) {
		groupWithMembers := model.Group{
			DisplayName: "Group with Members",
			Members: []model.Member{
				{Value: createdUser1.ID},
				{Value: createdUser2.ID},
			},
		}

		body, _ := json.Marshal(groupWithMembers)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Create Group with Members status = %v, want %v", w.Code, http.StatusCreated)
		}

		var response model.Group
		json.Unmarshal(w.Body.Bytes(), &response)
		if len(response.Members) != 2 {
			t.Errorf("Group should have 2 members, got %d", len(response.Members))
		}
	})

	t.Run("Create Group with Invalid Member", func(t *testing.T) {
		groupWithInvalidMember := model.Group{
			DisplayName: "Group with Invalid Member",
			Members: []model.Member{
				{Value: "non-existent-user"},
			},
		}

		body, _ := json.Marshal(groupWithInvalidMember)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Create Group with Invalid Member status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("Add User to Group", func(t *testing.T) {
		addReq := model.Member{
			Value: createdUser1.ID,
		}

		body, _ := json.Marshal(addReq)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups/"+createdGroup.ID+"/members", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Add User to Group status = %v, want %v", w.Code, http.StatusCreated)
		}

		var response model.Group
		json.Unmarshal(w.Body.Bytes(), &response)
		if len(response.Members) != 1 {
			t.Errorf("Group should have 1 member, got %d", len(response.Members))
		}
	})

	t.Run("Add Duplicate User to Group", func(t *testing.T) {
		addReq := model.Member{
			Value: createdUser1.ID,
		}

		body, _ := json.Marshal(addReq)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups/"+createdGroup.ID+"/members", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Add Duplicate User status = %v, want %v", w.Code, http.StatusConflict)
		}
	})

	t.Run("Add User to Non-existent Group", func(t *testing.T) {
		addReq := model.Member{
			Value: createdUser1.ID,
		}

		body, _ := json.Marshal(addReq)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups/non-existent/members", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Add User to Non-existent Group status = %v, want %v", w.Code, http.StatusNotFound)
		}
	})

	t.Run("Add Non-existent User to Group", func(t *testing.T) {
		addReq := model.Member{
			Value: "non-existent-user",
		}

		body, _ := json.Marshal(addReq)
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups/"+createdGroup.ID+"/members", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Add Non-existent User status = %v, want %v", w.Code, http.StatusNotFound)
		}
	})

	// 添加第二个用户
	store.AddMemberToGroup(createdGroup.ID, createdUser2.ID, "User")

	t.Run("Remove User from Group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Groups/"+createdGroup.ID+"/members/"+createdUser1.ID, nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Remove User from Group status = %v, want %v", w.Code, http.StatusNoContent)
		}

		// 验证用户已移除
		inGroup, _ := store.IsMemberInGroup(createdGroup.ID, createdUser1.ID)
		if inGroup {
			t.Error("User should not be in group after removal")
		}
	})

	t.Run("Remove Non-existent User from Group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Groups/"+createdGroup.ID+"/members/non-existent", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Remove Non-existent User status = %v, want %v", w.Code, http.StatusNotFound)
		}
	})

	t.Run("Remove User from Non-existent Group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Groups/non-existent/members/"+createdUser2.ID, nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Remove User from Non-existent Group status = %v, want %v", w.Code, http.StatusNotFound)
		}
	})

	t.Run("Verify Other Members Unaffected", func(t *testing.T) {
		group, _ := store.GetGroup(createdGroup.ID, false)
		if len(group.Members) != 1 {
			t.Errorf("Group should have 1 member remaining, got %d", len(group.Members))
		}
		if group.Members[0].Value != createdUser2.ID {
			t.Errorf("Remaining member should be user2, got %s", group.Members[0].Value)
		}
	})
}

func TestUserAttributesGroups(t *testing.T) {
	router, store := setupTestRouter()

	// 创建测试用户
	user1 := model.User{
		UserName: "alice",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Alice",
			FamilyName: "Smith",
		},
		Active: true,
	}

	user2 := model.User{
		UserName: "bob",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Bob",
			FamilyName: "Jones",
		},
		Active: true,
	}

	// 创建用户
	body, _ := json.Marshal(user1)
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var createdUser1 model.User
	json.Unmarshal(w.Body.Bytes(), &createdUser1)

	body, _ = json.Marshal(user2)
	req = httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var createdUser2 model.User
	json.Unmarshal(w.Body.Bytes(), &createdUser2)

	// 创建测试组
	group := model.Group{
		DisplayName: "Test Group",
	}
	body, _ = json.Marshal(group)
	req = httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var createdGroup model.Group
	json.Unmarshal(w.Body.Bytes(), &createdGroup)

	// 添加用户到组
	store.AddMemberToGroup(createdGroup.ID, createdUser1.ID, "User")
	store.AddMemberToGroup(createdGroup.ID, createdUser2.ID, "User")

	t.Run("List Users with attributes=groups", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?attributes=groups", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("List Users status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok {
			t.Error("Resources should be an array")
			return
		}

		// 验证每个用户都有 groups 字段
		for _, resource := range resources {
			userMap, ok := resource.(map[string]interface{})
			if !ok {
				t.Error("Resource should be a map")
				continue
			}

			if _, ok := userMap["groups"]; !ok {
				t.Error("User should have groups field when attributes=groups")
			}
		}
	})

	t.Run("Get User with attributes=groups", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users/"+createdUser1.ID+"?attributes=groups", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Get User status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if _, ok := response["groups"]; !ok {
			t.Error("User should have groups field when attributes=groups")
		}

		// 验证 groups 内容
		groups, ok := response["groups"].([]interface{})
		if !ok {
			t.Error("groups should be an array")
			return
		}

		if len(groups) != 1 {
			t.Errorf("User should be in 1 group, got %d", len(groups))
		}
	})

	t.Run("List Users with attributes=id,userName (no groups)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users?attributes=id,userName", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("List Users status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		resources, ok := response["Resources"].([]interface{})
		if !ok {
			t.Error("Resources should be an array")
			return
		}

		// 验证用户没有 groups 字段
		for _, resource := range resources {
			userMap, ok := resource.(map[string]interface{})
			if !ok {
				t.Error("Resource should be a map")
				continue
			}

			if _, ok := userMap["groups"]; ok {
				t.Error("User should not have groups field when attributes does not include groups")
			}
		}
	})
}

func TestGroupAttributesMembers(t *testing.T) {
	router, _ := setupTestRouter()

	// 创建测试用户
	user1 := model.User{
		UserName: "alice",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Alice",
			FamilyName: "Smith",
		},
		Active: true,
	}

	user2 := model.User{
		UserName: "bob",
		Name: struct {
			Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
			GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
			FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
			MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
		}{
			GivenName:  "Bob",
			FamilyName: "Jones",
		},
		Active: true,
	}

	// 创建用户
	body, _ := json.Marshal(user1)
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var createdUser1 model.User
	json.Unmarshal(w.Body.Bytes(), &createdUser1)

	body, _ = json.Marshal(user2)
	req = httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var createdUser2 model.User
	json.Unmarshal(w.Body.Bytes(), &createdUser2)

	// 创建测试组并添加成员
	group := model.Group{
		DisplayName: "Test Group",
		Members: []model.Member{
			{Value: createdUser1.ID},
			{Value: createdUser2.ID},
		},
	}
	body, _ = json.Marshal(group)
	req = httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var createdGroup model.Group
	json.Unmarshal(w.Body.Bytes(), &createdGroup)

	t.Run("List Groups with attributes=members", func(t *testing.T) {
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

		// 验证每个组都有 members 字段
		for _, resource := range resources {
			groupMap, ok := resource.(map[string]interface{})
			if !ok {
				t.Error("Resource should be a map")
				continue
			}

			if _, ok := groupMap["members"]; !ok {
				t.Error("Group should have members field when attributes=members")
			}
		}
	})

	t.Run("Get Group with attributes=members", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups/"+createdGroup.ID+"?attributes=members", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Get Group status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if _, ok := response["members"]; !ok {
			t.Error("Group should have members field when attributes=members")
		}

		// 验证 members 内容
		members, ok := response["members"].([]interface{})
		if !ok {
			t.Error("members should be an array")
			return
		}

		if len(members) != 2 {
			t.Errorf("Group should have 2 members, got %d", len(members))
		}
	})

	t.Run("List Groups with attributes=id,displayName (no members)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups?attributes=id,displayName", nil)
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

		// 验证组没有 members 字段
		for _, resource := range resources {
			groupMap, ok := resource.(map[string]interface{})
			if !ok {
				t.Error("Resource should be a map")
				continue
			}

			if _, ok := groupMap["members"]; ok {
				t.Error("Group should not have members field when attributes does not include members")
			}
		}
	})

	t.Run("List Groups with excludedAttributes=members", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups?excludedAttributes=members", nil)
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

		// 验证组没有 members 字段
		for _, resource := range resources {
			groupMap, ok := resource.(map[string]interface{})
			if !ok {
				t.Error("Resource should be a map")
				continue
			}

			if _, ok := groupMap["members"]; ok {
				t.Error("Group should not have members field when excludedAttributes=members")
			}
		}
	})
}
