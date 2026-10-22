package domain

import (
	"errors"
	"testing"
)

func TestUserValidate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{"valid", User{Email: "a@example.com", PasswordDigest: "digest", Role: RoleCustomer}, false},
		{"valid staff", User{Email: "a@example.com", PasswordDigest: "digest", Role: RoleStaff}, false},
		{"empty email", User{Email: "", PasswordDigest: "digest", Role: RoleCustomer}, true},
		{"blank email", User{Email: "   ", PasswordDigest: "digest", Role: RoleCustomer}, true},
		{"email with no @", User{Email: "not-an-email", PasswordDigest: "digest", Role: RoleCustomer}, true},
		{"empty password digest", User{Email: "a@example.com", PasswordDigest: "", Role: RoleCustomer}, true},
		{"unknown role", User{Email: "a@example.com", PasswordDigest: "digest", Role: "owner"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.wantErr && !errors.Is(err, ErrInvalid) {
				t.Errorf("Validate() = %v, want ErrInvalid", err)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}
