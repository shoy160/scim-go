package util

import (
	"testing"

	"scim-go/util"
)

// TestGroup 用于测试通配符和多属性值的组结构
type TestGroup struct {
	ID          string       `json:"id"`
	DisplayName string       `json:"displayName"`
	Members     []TestMember `json:"members"`
	Meta        TestMeta     `json:"meta"`
}

type TestMember struct {
	Value   string `json:"value"`
	Display string `json:"display"`
	Type    string `json:"type"`
	Ref     string `json:"$ref"`
}

type TestMeta struct {
	Created      string `json:"created"`
	LastModified string `json:"lastModified"`
}

func TestWildcardPatterns(t *testing.T) {
	group := TestGroup{
		ID:          "group-123",
		DisplayName: "Test Group",
		Members: []TestMember{
			{Value: "user-1", Display: "User One", Type: "User", Ref: "/Users/user-1"},
			{Value: "user-2", Display: "User Two", Type: "User", Ref: "/Users/user-2"},
		},
		Meta: TestMeta{
			Created:      "2024-01-01T00:00:00Z",
			LastModified: "2024-01-02T00:00:00Z",
		},
	}

	tests := []struct {
		name       string
		attributes string
		wantFields []string
		checkFunc  func(t *testing.T, result map[string]interface{})
	}{
		{
			name:       "wildcard members.*",
			attributes: "members.*",
			wantFields: []string{"id", "members"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// 验证 members 包含所有字段
				members, ok := result["members"].([]interface{})
				if !ok {
					t.Errorf("members should be an array")
					return
				}
				if len(members) != 2 {
					t.Errorf("members length = %v, want 2", len(members))
					return
				}
				// 检查第一个成员是否包含所有字段
				firstMember, ok := members[0].(map[string]interface{})
				if !ok {
					t.Errorf("first member should be a map")
					return
				}
				if _, ok := firstMember["value"]; !ok {
					t.Errorf("first member should have 'value' field")
				}
				if _, ok := firstMember["display"]; !ok {
					t.Errorf("first member should have 'display' field")
				}
				if _, ok := firstMember["type"]; !ok {
					t.Errorf("first member should have 'type' field")
				}
			},
		},
		{
			name:       "multiple specific member attributes",
			attributes: "members.value,members.display",
			wantFields: []string{"id", "members"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				members, ok := result["members"].([]interface{})
				if !ok {
					t.Errorf("members should be an array")
					return
				}
				firstMember, ok := members[0].(map[string]interface{})
				if !ok {
					t.Errorf("first member should be a map")
					return
				}
				// 应该只有 value 和 display 字段
				if _, ok := firstMember["value"]; !ok {
					t.Errorf("first member should have 'value' field")
				}
				if _, ok := firstMember["display"]; !ok {
					t.Errorf("first member should have 'display' field")
				}
				if _, ok := firstMember["type"]; ok {
					t.Errorf("first member should NOT have 'type' field")
				}
			},
		},
		{
			name:       "mixed wildcard and specific attributes",
			attributes: "members.*,displayName",
			wantFields: []string{"id", "members", "displayName"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// 验证 displayName 存在
				if _, ok := result["displayName"]; !ok {
					t.Errorf("result should have 'displayName' field")
				}
				// 验证 members 包含所有字段
				members, ok := result["members"].([]interface{})
				if !ok {
					t.Errorf("members should be an array")
					return
				}
				if len(members) != 2 {
					t.Errorf("members length = %v, want 2", len(members))
				}
			},
		},
		{
			name:       "single attribute without wildcard",
			attributes: "displayName",
			wantFields: []string{"id", "displayName"},
			checkFunc:  nil,
		},
		{
			name:       "multiple top-level attributes",
			attributes: "id,displayName",
			wantFields: []string{"id", "displayName"},
			checkFunc:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := util.ApplyAttributeSelection(group, tt.attributes, "")
			if err != nil {
				t.Errorf("ApplyAttributeSelection() error = %v", err)
				return
			}

			// 检查期望的字段是否存在
			for _, field := range tt.wantFields {
				if _, ok := result[field]; !ok {
					t.Errorf("ApplyAttributeSelection() missing field %s", field)
				}
			}

			// 执行额外的检查函数
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestWildcardWithNestedObjects(t *testing.T) {
	// 测试嵌套对象的通配符
	obj := map[string]interface{}{
		"id":   "test-123",
		"name": "Test",
		"meta": map[string]interface{}{
			"created":      "2024-01-01",
			"lastModified": "2024-01-02",
			"version":      "1.0",
		},
	}

	tests := []struct {
		name       string
		attributes string
		checkFunc  func(t *testing.T, result map[string]interface{})
	}{
		{
			name:       "wildcard on nested object",
			attributes: "meta.*",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				meta, ok := result["meta"].(map[string]interface{})
				if !ok {
					t.Errorf("meta should be a map")
					return
				}
				// 应该包含所有 meta 字段
				if _, ok := meta["created"]; !ok {
					t.Errorf("meta should have 'created' field")
				}
				if _, ok := meta["lastModified"]; !ok {
					t.Errorf("meta should have 'lastModified' field")
				}
				if _, ok := meta["version"]; !ok {
					t.Errorf("meta should have 'version' field")
				}
			},
		},
		{
			name:       "specific nested attribute",
			attributes: "meta.created",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				meta, ok := result["meta"].(map[string]interface{})
				if !ok {
					t.Errorf("meta should be a map")
					return
				}
				if _, ok := meta["created"]; !ok {
					t.Errorf("meta should have 'created' field")
				}
				if _, ok := meta["lastModified"]; ok {
					t.Errorf("meta should NOT have 'lastModified' field")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := util.ApplyAttributeSelection(obj, tt.attributes, "")
			if err != nil {
				t.Errorf("ApplyAttributeSelection() error = %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestValidateAttributeFormatWithWildcards(t *testing.T) {
	tests := []struct {
		name    string
		attrs   string
		wantErr bool
		errMsg  string
	}{
		// 有效的通配符模式
		{
			name:    "valid wildcard pattern",
			attrs:   "members.*",
			wantErr: false,
		},
		{
			name:    "valid multiple with wildcard",
			attrs:   "members.*,displayName",
			wantErr: false,
		},
		{
			name:    "valid multiple specific attributes",
			attrs:   "members.value,members.display",
			wantErr: false,
		},
		{
			name:    "valid mixed patterns",
			attrs:   "members.*,groups.value,name.givenName",
			wantErr: false,
		},
		// 无效的通配符模式
		{
			name:    "invalid wildcard in middle",
			attrs:   "members.*.value",
			wantErr: true,
			errMsg:  "wildcard can only be used as '.*' at the end",
		},
		{
			name:    "invalid standalone wildcard",
			attrs:   "*",
			wantErr: true,
			errMsg:  "wildcard",
		},
		{
			name:    "invalid wildcard without dot",
			attrs:   "members*",
			wantErr: true,
			errMsg:  "wildcard can only be used as '.*' at the end",
		},
		// 有效的普通属性
		{
			name:    "valid single attribute",
			attrs:   "userName",
			wantErr: false,
		},
		{
			name:    "valid nested attribute",
			attrs:   "name.givenName",
			wantErr: false,
		},
		{
			name:    "valid multiple attributes",
			attrs:   "userName,displayName,emails",
			wantErr: false,
		},
		// 无效的属性格式
		{
			name:    "invalid empty attribute",
			attrs:   "",
			wantErr: false, // 空字符串是允许的
		},
		{
			name:    "invalid starts with dot",
			attrs:   ".userName",
			wantErr: true,
			errMsg:  "cannot start with dot",
		},
		{
			name:    "invalid consecutive dots",
			attrs:   "name..givenName",
			wantErr: true,
			errMsg:  "cannot contain consecutive dots",
		},
		{
			name:    "invalid character",
			attrs:   "user-name",
			wantErr: true,
			errMsg:  "contains invalid character",
		},
		{
			name:    "invalid starts with digit",
			attrs:   "123user",
			wantErr: true,
			errMsg:  "cannot start with digit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := util.ValidateAttributeFormat(tt.attrs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAttributeFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateAttributeFormat() error message = %v, should contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(substr) <= len(s) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBackwardCompatibility(t *testing.T) {
	// 确保向后兼容 - 测试原有的功能仍然正常工作
	user := TestUser{
		ID:          "123",
		UserName:    "john.doe",
		DisplayName: "John Doe",
		Active:      true,
		Emails: []Email{
			{Value: "john@example.com", Primary: true},
		},
		Name: NameInfo{
			GivenName:  "John",
			FamilyName: "Doe",
		},
	}

	tests := []struct {
		name               string
		attributes         string
		excludedAttributes string
		wantFields         []string
		notWantFields      []string
	}{
		{
			name:               "no selection - all fields",
			attributes:         "",
			excludedAttributes: "",
			wantFields:         []string{"id", "userName", "displayName", "active", "emails", "name"},
			notWantFields:      []string{},
		},
		{
			name:               "select specific fields",
			attributes:         "userName,displayName",
			excludedAttributes: "",
			wantFields:         []string{"id", "userName", "displayName"},
			notWantFields:      []string{"active", "emails", "name"},
		},
		{
			name:               "exclude fields",
			attributes:         "",
			excludedAttributes: "emails,name",
			wantFields:         []string{"id", "userName", "displayName", "active"},
			notWantFields:      []string{"emails", "name"},
		},
		{
			name:               "nested attribute selection",
			attributes:         "name.givenName",
			excludedAttributes: "",
			wantFields:         []string{"id", "name"},
			notWantFields:      []string{"userName", "displayName", "active", "emails"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := util.ApplyAttributeSelection(user, tt.attributes, tt.excludedAttributes)
			if err != nil {
				t.Errorf("ApplyAttributeSelection() error = %v", err)
				return
			}

			// 检查期望的字段是否存在
			for _, field := range tt.wantFields {
				if _, ok := result[field]; !ok {
					t.Errorf("ApplyAttributeSelection() missing field %s", field)
				}
			}

			// 检查不应该存在的字段
			for _, field := range tt.notWantFields {
				if _, ok := result[field]; ok {
					t.Errorf("ApplyAttributeSelection() should not have field %s", field)
				}
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		obj        interface{}
		attributes string
		checkFunc  func(t *testing.T, result map[string]interface{})
	}{
		{
			name:       "empty array with wildcard",
			obj:        map[string]interface{}{"id": "123", "members": []interface{}{}},
			attributes: "members.*",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				members, ok := result["members"].([]interface{})
				if !ok {
					t.Errorf("members should be an array")
					return
				}
				if len(members) != 0 {
					t.Errorf("members length = %v, want 0", len(members))
				}
			},
		},
		{
			name:       "nil value with wildcard",
			obj:        map[string]interface{}{"id": "123", "members": nil},
			attributes: "members.*",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// nil 值应该被保留
				if _, ok := result["members"]; !ok {
					t.Errorf("members field should exist")
				}
			},
		},
		{
			name:       "wildcard on non-existent attribute",
			obj:        map[string]interface{}{"id": "123"},
			attributes: "nonexistent.*",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// 不存在的属性应该被忽略
				if _, ok := result["nonexistent"]; ok {
					t.Errorf("nonexistent field should not exist")
				}
				// 但 id 应该存在
				if _, ok := result["id"]; !ok {
					t.Errorf("id field should exist")
				}
			},
		},
		{
			name:       "whitespace in attributes",
			obj:        map[string]interface{}{"id": "123", "displayName": "Test"},
			attributes: "  id  ,  displayName  ",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				if _, ok := result["id"]; !ok {
					t.Errorf("id field should exist")
				}
				if _, ok := result["displayName"]; !ok {
					t.Errorf("displayName field should exist")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := util.ApplyAttributeSelection(tt.obj, tt.attributes, "")
			if err != nil {
				t.Errorf("ApplyAttributeSelection() error = %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}
