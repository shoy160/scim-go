package api

import (
	"scim-go/model"
	"scim-go/util"
	"testing"
)

// TestUserHandler_ParseAttributeList 测试 ParseAttributeList 函数
func TestUserHandler_ParseAttributeList(t *testing.T) {
	tests := []struct {
		name       string
		attrs      string
		wantLength int
	}{
		{
			name:       "single attribute",
			attrs:      "userName",
			wantLength: 1,
		},
		{
			name:       "multiple attributes",
			attrs:      "userName,displayName,groups.value",
			wantLength: 3,
		},
		{
			name:       "attributes with spaces",
			attrs:      "userName, displayName, groups.value",
			wantLength: 3,
		},
		{
			name:       "empty string",
			attrs:      "",
			wantLength: 0,
		},
		{
			name:       "wildcard pattern",
			attrs:      "groups.*",
			wantLength: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.ParseAttributeList(tt.attrs)
			if len(result) != tt.wantLength {
				t.Errorf("parseAttributeList() length = %v, want %v", len(result), tt.wantLength)
			}
		})
	}
}

// TestUserHandler_GroupsValueAttribute 测试 groups.value 属性过滤
func TestUserHandler_GroupsValueAttribute(t *testing.T) {
	user := &model.User{
		ID:       "user-123",
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
		Groups: []model.UserGroup{
			{Value: "group-1", Display: "Group One"},
			{Value: "group-2", Display: "Group Two"},
		},
	}

	tests := []struct {
		name       string
		attributes string
		checkFunc  func(t *testing.T, result map[string]interface{})
	}{
		{
			name:       "groups.value only",
			attributes: "groups.value",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				groups, ok := result["groups"].([]interface{})
				if !ok {
					t.Errorf("groups should be an array")
					return
				}
				if len(groups) != 2 {
					t.Errorf("groups length = %v, want 2", len(groups))
					return
				}
				// 检查第一个组
				firstGroup, ok := groups[0].(map[string]interface{})
				if !ok {
					t.Errorf("first group should be a map")
					return
				}
				if _, ok := firstGroup["value"]; !ok {
					t.Errorf("first group should have 'value' field")
				}
				if _, ok := firstGroup["display"]; ok {
					t.Errorf("first group should NOT have 'display' field")
				}
			},
		},
		{
			name:       "groups.display only",
			attributes: "groups.display",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				groups, ok := result["groups"].([]interface{})
				if !ok {
					t.Errorf("groups should be an array")
					return
				}
				firstGroup, ok := groups[0].(map[string]interface{})
				if !ok {
					t.Errorf("first group should be a map")
					return
				}
				if _, ok := firstGroup["display"]; !ok {
					t.Errorf("first group should have 'display' field")
				}
				if _, ok := firstGroup["value"]; ok {
					t.Errorf("first group should NOT have 'value' field")
				}
			},
		},
		{
			name:       "groups.value and groups.display",
			attributes: "groups.value,groups.display",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				groups, ok := result["groups"].([]interface{})
				if !ok {
					t.Errorf("groups should be an array")
					return
				}
				firstGroup, ok := groups[0].(map[string]interface{})
				if !ok {
					t.Errorf("first group should be a map")
					return
				}
				if _, ok := firstGroup["value"]; !ok {
					t.Errorf("first group should have 'value' field")
				}
				if _, ok := firstGroup["display"]; !ok {
					t.Errorf("first group should have 'display' field")
				}
			},
		},
		{
			name:       "groups.* wildcard",
			attributes: "groups.*",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				groups, ok := result["groups"].([]interface{})
				if !ok {
					t.Errorf("groups should be an array")
					return
				}
				firstGroup, ok := groups[0].(map[string]interface{})
				if !ok {
					t.Errorf("first group should be a map")
					return
				}
				if _, ok := firstGroup["value"]; !ok {
					t.Errorf("first group should have 'value' field")
				}
				if _, ok := firstGroup["display"]; !ok {
					t.Errorf("first group should have 'display' field")
				}
			},
		},
		{
			name:       "groups with other attributes",
			attributes: "userName,groups.value,displayName",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				if _, ok := result["userName"]; !ok {
					t.Errorf("result should have 'userName' field")
				}
				groups, ok := result["groups"].([]interface{})
				if !ok {
					t.Errorf("groups should be an array")
					return
				}
				firstGroup, ok := groups[0].(map[string]interface{})
				if !ok {
					t.Errorf("first group should be a map")
					return
				}
				if _, ok := firstGroup["value"]; !ok {
					t.Errorf("first group should have 'value' field")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证属性格式
			if err := util.ValidateAttributeFormat(tt.attributes); err != nil {
				t.Errorf("ValidateAttributeFormat() error = %v", err)
				return
			}

			// 应用属性选择
			filtered, err := util.ApplyAttributeSelectionWithSpecialRules(user, tt.attributes, "", "groups")
			if err != nil {
				t.Errorf("ApplyAttributeSelectionWithSpecialRules() error = %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, filtered)
			}
		})
	}
}

// TestUserHandler_EmptyGroups 测试空 groups 数组的处理
func TestUserHandler_EmptyGroups(t *testing.T) {
	user := &model.User{
		ID:       "user-123",
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
		Groups: []model.UserGroup{},
	}

	tests := []struct {
		name       string
		attributes string
		checkFunc  func(t *testing.T, result map[string]interface{})
	}{
		{
			name:       "empty groups with groups.value",
			attributes: "groups.value",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				groups, ok := result["groups"].([]interface{})
				if !ok {
					t.Errorf("groups should be an array")
					return
				}
				if len(groups) != 0 {
					t.Errorf("groups length = %v, want 0", len(groups))
				}
			},
		},
		{
			name:       "empty groups with groups.*",
			attributes: "groups.*",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				groups, ok := result["groups"].([]interface{})
				if !ok {
					t.Errorf("groups should be an array")
					return
				}
				if len(groups) != 0 {
					t.Errorf("groups length = %v, want 0", len(groups))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, err := util.ApplyAttributeSelectionWithSpecialRules(user, tt.attributes, "", "groups")
			if err != nil {
				t.Errorf("ApplyAttributeSelectionWithSpecialRules() error = %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, filtered)
			}
		})
	}
}
