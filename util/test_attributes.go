package util

import (
	"reflect"
	"testing"
)

type TestUser struct {
	ID          string   `json:"id"`
	UserName    string   `json:"userName"`
	DisplayName string   `json:"displayName"`
	Active      bool     `json:"active"`
	Emails      []Email  `json:"emails"`
	Name        NameInfo `json:"name"`
}

type Email struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type NameInfo struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

func TestApplyAttributeSelection(t *testing.T) {
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
		wantErr            bool
	}{
		{
			name:               "no selection",
			attributes:         "",
			excludedAttributes: "",
			wantFields:         []string{"id", "userName", "displayName", "active", "emails", "name"},
			wantErr:            false,
		},
		{
			name:               "select specific attributes",
			attributes:         "userName,displayName",
			excludedAttributes: "",
			wantFields:         []string{"id", "userName", "displayName"},
			wantErr:            false,
		},
		{
			name:               "exclude attributes",
			attributes:         "",
			excludedAttributes: "emails,name",
			wantFields:         []string{"id", "userName", "displayName", "active"},
			wantErr:            false,
		},
		{
			name:               "select nested attributes",
			attributes:         "name.givenName,userName",
			excludedAttributes: "",
			wantFields:         []string{"id", "userName", "name"},
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyAttributeSelection(user, tt.attributes, tt.excludedAttributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyAttributeSelection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 检查结果中是否包含期望的字段
			for _, field := range tt.wantFields {
				if _, ok := result[field]; !ok {
					t.Errorf("ApplyAttributeSelection() missing field %s", field)
				}
			}
		})
	}
}

func TestParseAttributeList(t *testing.T) {
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
			attrs:      "userName,displayName,emails",
			wantLength: 3,
		},
		{
			name:       "attributes with spaces",
			attrs:      "userName, displayName, emails",
			wantLength: 3,
		},
		{
			name:       "empty string",
			attrs:      "",
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAttributeList(tt.attrs)
			if len(result) != tt.wantLength {
				t.Errorf("parseAttributeList() length = %v, want %v", len(result), tt.wantLength)
			}
		})
	}
}

func TestGetNestedValue(t *testing.T) {
	obj := map[string]interface{}{
		"userName": "john.doe",
		"name": map[string]interface{}{
			"givenName":  "John",
			"familyName": "Doe",
		},
	}

	tests := []struct {
		name     string
		path     string
		expected interface{}
	}{
		{
			name:     "top level",
			path:     "userName",
			expected: "john.doe",
		},
		{
			name:     "nested",
			path:     "name.givenName",
			expected: "John",
		},
		{
			name:     "non-existent",
			path:     "nonexistent",
			expected: nil,
		},
		{
			name:     "non-existent nested",
			path:     "name.middleName",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNestedValue(obj, tt.path)
			if result != tt.expected {
				t.Errorf("GetNestedValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		value    interface{}
		expected interface{}
	}{
		{
			name:     "top level",
			path:     "userName",
			value:    "john.doe",
			expected: "john.doe",
		},
		{
			name:     "nested",
			path:     "name.givenName",
			value:    "John",
			expected: "John",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := make(map[string]interface{})
			SetNestedValue(obj, tt.path, tt.value)

			result := GetNestedValue(obj, tt.path)
			if result != tt.expected {
				t.Errorf("SetNestedValue() result = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStructToMap(t *testing.T) {
	user := TestUser{
		ID:       "123",
		UserName: "john.doe",
	}

	result, err := StructToMap(user)
	if err != nil {
		t.Fatalf("StructToMap() error = %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("StructToMap() id = %v, want %v", result["id"], "123")
	}

	if result["userName"] != "john.doe" {
		t.Errorf("StructToMap() userName = %v, want %v", result["userName"], "john.doe")
	}
}

func TestMapToStruct(t *testing.T) {
	m := map[string]interface{}{
		"id":       "123",
		"userName": "john.doe",
		"active":   true,
	}

	var user TestUser
	err := MapToStruct(m, &user)
	if err != nil {
		t.Fatalf("MapToStruct() error = %v", err)
	}

	if user.ID != "123" {
		t.Errorf("MapToStruct() ID = %v, want %v", user.ID, "123")
	}

	if user.UserName != "john.doe" {
		t.Errorf("MapToStruct() UserName = %v, want %v", user.UserName, "john.doe")
	}

	if !user.Active {
		t.Errorf("MapToStruct() Active = %v, want %v", user.Active, true)
	}
}

func TestValidateAttributes(t *testing.T) {
	validAttrs := []string{"userName", "displayName", "emails", "name.givenName"}

	tests := []struct {
		name             string
		attrs            string
		wantValidCount   int
		wantInvalidCount int
	}{
		{
			name:             "all valid",
			attrs:            "userName,displayName",
			wantValidCount:   2,
			wantInvalidCount: 0,
		},
		{
			name:             "some invalid",
			attrs:            "userName,invalidAttr",
			wantValidCount:   1,
			wantInvalidCount: 1,
		},
		{
			name:             "nested valid",
			attrs:            "name.givenName",
			wantValidCount:   1,
			wantInvalidCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, invalid := ValidateAttributes(validAttrs, tt.attrs)
			if len(valid) != tt.wantValidCount {
				t.Errorf("ValidateAttributes() valid count = %v, want %v", len(valid), tt.wantValidCount)
			}
			if len(invalid) != tt.wantInvalidCount {
				t.Errorf("ValidateAttributes() invalid count = %v, want %v", len(invalid), tt.wantInvalidCount)
			}
		})
	}
}

func TestGetAttributeNames(t *testing.T) {
	names := GetAttributeNames(reflect.TypeOf(TestUser{}), "")

	// 应该包含所有字段
	expectedFields := map[string]bool{
		"id":          false,
		"userName":    false,
		"displayName": false,
		"active":      false,
		"emails":      false,
		"name":        false,
	}

	for _, name := range names {
		if _, ok := expectedFields[name]; ok {
			expectedFields[name] = true
		}
	}

	for field, found := range expectedFields {
		if !found {
			t.Errorf("GetAttributeNames() missing field %s", field)
		}
	}
}
