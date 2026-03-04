package util

import (
	"scim-go/util"
	"testing"
)

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{
			name:    "simple equality",
			filter:  `userName eq "john"`,
			wantErr: false,
		},
		{
			name:    "equality with email",
			filter:  `emails.value eq "john@example.com"`,
			wantErr: false,
		},
		{
			name:    "contains filter",
			filter:  `userName co "john"`,
			wantErr: false,
		},
		{
			name:    "starts with filter",
			filter:  `userName sw "john"`,
			wantErr: false,
		},
		{
			name:    "ends with filter",
			filter:  `userName ew "john"`,
			wantErr: false,
		},
		{
			name:    "presence filter",
			filter:  `userName pr`,
			wantErr: false,
		},
		{
			name:    "greater than filter",
			filter:  `meta.created gt "2023-01-01"`,
			wantErr: false,
		},
		{
			name:    "and filter",
			filter:  `userName eq "john" and active eq true`,
			wantErr: false,
		},
		{
			name:    "or filter",
			filter:  `userName eq "john" or userName eq "jane"`,
			wantErr: false,
		},
		{
			name:    "not filter",
			filter:  `not (userName eq "john")`,
			wantErr: false,
		},
		{
			name:    "complex filter",
			filter:  `(userName eq "john" or userName eq "jane") and active eq true`,
			wantErr: false,
		},
		{
			name:    "empty filter",
			filter:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := util.ParseFilter(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.filter != "" && node == nil {
				t.Error("ParseFilter() returned nil node for non-empty filter")
			}
		})
	}
}

func TestMatchFilter(t *testing.T) {
	obj := map[string]interface{}{
		"userName": "john.doe",
		"active":   true,
		"name": map[string]interface{}{
			"givenName":  "John",
			"familyName": "Doe",
		},
		"emails": []interface{}{
			map[string]interface{}{
				"value":   "john@example.com",
				"primary": true,
			},
		},
	}

	tests := []struct {
		name    string
		filter  string
		want    bool
		wantErr bool
	}{
		{
			name:    "match username equality",
			filter:  `userName eq "john.doe"`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "no match username equality",
			filter:  `userName eq "jane.doe"`,
			want:    false,
			wantErr: false,
		},
		{
			name:    "match contains",
			filter:  `userName co "john"`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "match starts with",
			filter:  `userName sw "john"`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "match ends with",
			filter:  `userName ew "doe"`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "match presence",
			filter:  `userName pr`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "match nested property",
			filter:  `name.givenName eq "John"`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "match and",
			filter:  `userName eq "john.doe" and active eq true`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "no match and",
			filter:  `userName eq "john.doe" and active eq false`,
			want:    false,
			wantErr: false,
		},
		{
			name:    "match or",
			filter:  `userName eq "john.doe" or userName eq "jane.doe"`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "match not",
			filter:  `not (userName eq "jane.doe")`,
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := util.ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			got, err := util.MatchFilter(node, obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MatchFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterToSQL(t *testing.T) {
	columnMapping := map[string]string{
		"userName": "user_name",
		"active":   "active",
	}

	tests := []struct {
		name    string
		filter  string
		wantSQL string
		wantErr bool
	}{
		{
			name:    "equality",
			filter:  `userName eq "john"`,
			wantSQL: "user_name = ?",
			wantErr: false,
		},
		{
			name:    "contains",
			filter:  `userName co "john"`,
			wantSQL: "user_name LIKE ?",
			wantErr: false,
		},
		{
			name:    "starts with",
			filter:  `userName sw "john"`,
			wantSQL: "user_name LIKE ?",
			wantErr: false,
		},
		{
			name:    "ends with",
			filter:  `userName ew "john"`,
			wantSQL: "user_name LIKE ?",
			wantErr: false,
		},
		{
			name:    "presence",
			filter:  `userName pr`,
			wantSQL: "user_name IS NOT NULL AND user_name != ''",
			wantErr: false,
		},
		{
			name:    "and",
			filter:  `userName eq "john" and active eq true`,
			wantSQL: "(user_name = ? AND active = ?)",
			wantErr: false,
		},
		{
			name:    "or",
			filter:  `userName eq "john" or userName eq "jane"`,
			wantSQL: "(user_name = ? OR user_name = ?)",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := util.ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			sql, _, err := util.FilterToSQL(node, columnMapping)
			if (err != nil) != tt.wantErr {
				t.Errorf("FilterToSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if sql != tt.wantSQL {
				t.Errorf("FilterToSQL() sql = %v, want %v", sql, tt.wantSQL)
			}
		})
	}
}

func TestValidateFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{
			name:    "valid filter",
			filter:  `userName eq "john"`,
			wantErr: false,
		},
		{
			name:    "empty filter",
			filter:  "",
			wantErr: false,
		},
		{
			name:    "invalid operator",
			filter:  `userName xx "john"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := util.ValidateFilter(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
