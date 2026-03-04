package util

import (
	"scim-go/util"
	"testing"
)

func TestValidateEmailFormat(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "email without @",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "email without domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "email without local part",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "email without dot in domain",
			email:   "user@example",
			wantErr: true,
		},
		{
			name:    "email with multiple @",
			email:   "user@@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := util.ValidateEmailFormat(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmailFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRoleDefinition(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{
			name:    "valid role",
			role:    "admin",
			wantErr: false,
		},
		{
			name:    "valid role with spaces",
			role:    "system admin",
			wantErr: false,
		},
		{
			name:    "empty role",
			role:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := util.ValidateRoleDefinition(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRoleDefinition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
