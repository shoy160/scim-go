package util

import (
	"scim-go/util"
	"testing"
)

func TestParsePathWithFilter(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		expectedAttr    string
		expectedFilter  *util.PathFilter
		expectedSubPath string
		expectError     bool
	}{
		{
			name:         "simple path without filter",
			path:         "userName",
			expectedAttr: "userName",
			expectError:  false,
		},
		{
			name:            "path with dot notation",
			path:            "name.givenName",
			expectedAttr:    "name",
			expectedSubPath: "givenName",
			expectError:     false,
		},
		{
			name:         "path with filter - eq operator",
			path:         `phoneNumbers[type eq "mobile"]`,
			expectedAttr: "phoneNumbers",
			expectedFilter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "mobile",
				IsString:      true,
			},
			expectError: false,
		},
		{
			name:         "path with filter - ne operator",
			path:         `addresses[type ne "work"]`,
			expectedAttr: "addresses",
			expectedFilter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpNe,
				Value:         "work",
				IsString:      true,
			},
			expectError: false,
		},
		{
			name:         "path with filter and subpath",
			path:         `phoneNumbers[type eq "work"].value`,
			expectedAttr: "phoneNumbers",
			expectedFilter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "work",
				IsString:      true,
			},
			expectedSubPath: "value",
			expectError:     false,
		},
		{
			name:         "path with filter - co operator",
			path:         `emails[value co "@example"]`,
			expectedAttr: "emails",
			expectedFilter: &util.PathFilter{
				AttributeName: "value",
				Operator:      util.OpCo,
				Value:         "@example",
				IsString:      true,
			},
			expectError: false,
		},
		{
			name:         "path with filter - sw operator",
			path:         `emails[value sw "test"]`,
			expectedAttr: "emails",
			expectedFilter: &util.PathFilter{
				AttributeName: "value",
				Operator:      util.OpSw,
				Value:         "test",
				IsString:      true,
			},
			expectError: false,
		},
		{
			name:         "path with filter - ew operator",
			path:         `emails[value ew ".com"]`,
			expectedAttr: "emails",
			expectedFilter: &util.PathFilter{
				AttributeName: "value",
				Operator:      util.OpEw,
				Value:         ".com",
				IsString:      true,
			},
			expectError: false,
		},
		{
			name:         "path with filter - pr operator",
			path:         `addresses[type pr]`,
			expectedAttr: "addresses",
			expectedFilter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpPr,
				Value:         "",
				IsString:      false,
			},
			expectError: false,
		},
		{
			name:         "path with filter - gt operator",
			path:         `addresses[postalCode gt "10000"]`,
			expectedAttr: "addresses",
			expectedFilter: &util.PathFilter{
				AttributeName: "postalCode",
				Operator:      util.OpGt,
				Value:         "10000",
				IsString:      true,
			},
			expectError: false,
		},
		{
			name:         "path with filter - single quotes",
			path:         `phoneNumbers[type eq 'home']`,
			expectedAttr: "phoneNumbers",
			expectedFilter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "home",
				IsString:      true,
			},
			expectError: false,
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := util.ParsePathWithFilter(tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.AttributeName != tt.expectedAttr {
				t.Errorf("expected attribute name %s, got %s", tt.expectedAttr, result.AttributeName)
			}

			if tt.expectedFilter != nil {
				if result.Filter == nil {
					t.Errorf("expected filter but got nil")
					return
				}

				if result.Filter.AttributeName != tt.expectedFilter.AttributeName {
					t.Errorf("expected filter attribute name %s, got %s", tt.expectedFilter.AttributeName, result.Filter.AttributeName)
				}

				if result.Filter.Operator != tt.expectedFilter.Operator {
					t.Errorf("expected filter operator %s, got %s", tt.expectedFilter.Operator, result.Filter.Operator)
				}

				if result.Filter.Value != tt.expectedFilter.Value {
					t.Errorf("expected filter value %s, got %s", tt.expectedFilter.Value, result.Filter.Value)
				}
			}

			if result.SubPath != tt.expectedSubPath {
				t.Errorf("expected sub path %s, got %s", tt.expectedSubPath, result.SubPath)
			}
		})
	}
}

func TestMatchPathFilter(t *testing.T) {
	tests := []struct {
		name          string
		item          map[string]interface{}
		filter        *util.PathFilter
		expectedMatch bool
		expectError   bool
	}{
		{
			name: "eq operator - match",
			item: map[string]interface{}{
				"type": "mobile",
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "mobile",
			},
			expectedMatch: true,
		},
		{
			name: "eq operator - no match",
			item: map[string]interface{}{
				"type": "work",
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "mobile",
			},
			expectedMatch: false,
		},
		{
			name: "eq operator - case insensitive",
			item: map[string]interface{}{
				"type": "MOBILE",
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "mobile",
			},
			expectedMatch: true,
		},
		{
			name: "ne operator - match",
			item: map[string]interface{}{
				"type": "work",
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpNe,
				Value:         "mobile",
			},
			expectedMatch: true,
		},
		{
			name: "co operator - match",
			item: map[string]interface{}{
				"value": "test@example.com",
			},
			filter: &util.PathFilter{
				AttributeName: "value",
				Operator:      util.OpCo,
				Value:         "@example",
			},
			expectedMatch: true,
		},
		{
			name: "sw operator - match",
			item: map[string]interface{}{
				"value": "+86-13800138000",
			},
			filter: &util.PathFilter{
				AttributeName: "value",
				Operator:      util.OpSw,
				Value:         "+86",
			},
			expectedMatch: true,
		},
		{
			name: "ew operator - match",
			item: map[string]interface{}{
				"value": "test@example.com",
			},
			filter: &util.PathFilter{
				AttributeName: "value",
				Operator:      util.OpEw,
				Value:         ".com",
			},
			expectedMatch: true,
		},
		{
			name: "pr operator - present",
			item: map[string]interface{}{
				"type": "work",
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpPr,
			},
			expectedMatch: true,
		},
		{
			name: "pr operator - not present",
			item: map[string]interface{}{
				"value": "test@example.com",
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpPr,
			},
			expectedMatch: false,
		},
		{
			name: "missing attribute",
			item: map[string]interface{}{
				"value": "test@example.com",
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "work",
			},
			expectedMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := util.MatchPathFilter(tt.item, tt.filter)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if match != tt.expectedMatch {
				t.Errorf("expected match %v, got %v", tt.expectedMatch, match)
			}
		})
	}
}

func TestFindMatchingIndices(t *testing.T) {
	tests := []struct {
		name            string
		items           []map[string]interface{}
		filter          *util.PathFilter
		expectedIndices []int
		expectError     bool
	}{
		{
			name: "find all mobile phone numbers",
			items: []map[string]interface{}{
				{"type": "work", "value": "+86-13800138000"},
				{"type": "mobile", "value": "+86-13900139000"},
				{"type": "home", "value": "+86-13700137000"},
				{"type": "mobile", "value": "+86-13600136000"},
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "mobile",
			},
			expectedIndices: []int{1, 3},
		},
		{
			name: "find work addresses",
			items: []map[string]interface{}{
				{"type": "work", "streetAddress": "123 Main St"},
				{"type": "home", "streetAddress": "456 Oak Ave"},
				{"type": "work", "streetAddress": "789 Pine Rd"},
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "work",
			},
			expectedIndices: []int{0, 2},
		},
		{
			name: "no matching items",
			items: []map[string]interface{}{
				{"type": "work", "value": "test1@example.com"},
				{"type": "home", "value": "test2@example.com"},
			},
			filter: &util.PathFilter{
				AttributeName: "type",
				Operator:      util.OpEq,
				Value:         "mobile",
			},
			expectedIndices: []int{},
		},
		{
			name: "nil filter - return all",
			items: []map[string]interface{}{
				{"type": "work"},
				{"type": "home"},
			},
			filter:          nil,
			expectedIndices: []int{0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indices, err := util.FindMatchingIndices(tt.items, tt.filter)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(indices) != len(tt.expectedIndices) {
				t.Errorf("expected %d indices, got %d", len(tt.expectedIndices), len(indices))
				return
			}

			for i, idx := range indices {
				if idx != tt.expectedIndices[i] {
					t.Errorf("expected index %d at position %d, got %d", tt.expectedIndices[i], i, idx)
				}
			}
		})
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name          string
		actual        string
		expected      string
		op            util.Operator
		expectedMatch bool
		expectError   bool
	}{
		{
			name:          "eq - match",
			actual:        "test",
			expected:      "test",
			op:            util.OpEq,
			expectedMatch: true,
		},
		{
			name:          "eq - case insensitive match",
			actual:        "TEST",
			expected:      "test",
			op:            util.OpEq,
			expectedMatch: true,
		},
		{
			name:          "ne - match",
			actual:        "test",
			expected:      "other",
			op:            util.OpNe,
			expectedMatch: true,
		},
		{
			name:          "co - match",
			actual:        "test@example.com",
			expected:      "@example",
			op:            util.OpCo,
			expectedMatch: true,
		},
		{
			name:          "sw - match",
			actual:        "test@example.com",
			expected:      "test",
			op:            util.OpSw,
			expectedMatch: true,
		},
		{
			name:          "ew - match",
			actual:        "test@example.com",
			expected:      ".com",
			op:            util.OpEw,
			expectedMatch: true,
		},
		{
			name:          "pr - present",
			actual:        "test",
			expected:      "",
			op:            util.OpPr,
			expectedMatch: true,
		},
		{
			name:          "pr - not present",
			actual:        "",
			expected:      "",
			op:            util.OpPr,
			expectedMatch: false,
		},
		{
			name:          "gt - numeric",
			actual:        "100",
			expected:      "50",
			op:            util.OpGt,
			expectedMatch: true,
		},
		{
			name:          "lt - numeric",
			actual:        "50",
			expected:      "100",
			op:            util.OpLt,
			expectedMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := util.CompareValues(tt.actual, tt.expected, tt.op, nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if match != tt.expectedMatch {
				t.Errorf("expected match %v, got %v", tt.expectedMatch, match)
			}
		})
	}
}
