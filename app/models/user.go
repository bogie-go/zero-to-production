package models

import (
	"fmt"
	"strings"
	"time"
)

// User is a user. PasswordDigest is bcrypt's digest of the password, never
// the password itself — set and checked by app/services/password, kept out
// of here since this package imports nothing third-party.
type User struct {
	ID             int64
	Email          string
	PasswordDigest string
	Role           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RoleCustomer is every signup's role, the only one the public signup
// endpoint can ever set. RoleStaff can manage products and the rest of the
// admin namespace; nothing here promotes a user to it — see docs/bogie-gaps.md
// and db/seeds/users.go for how the one staff account in this app exists.
const (
	RoleCustomer = "customer"
	RoleStaff    = "staff"
)

// UserRoles are the values Role may hold.
var UserRoles = []string{RoleCustomer, RoleStaff}

// Validate is the model's validations. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule.
func (u User) Validate() error {
	if strings.TrimSpace(u.Email) == "" || !strings.Contains(u.Email, "@") {
		return fmt.Errorf("%w: email must look like an email address", ErrInvalid)
	}
	if u.PasswordDigest == "" {
		return fmt.Errorf("%w: password is required", ErrInvalid)
	}
	known := false
	for _, r := range UserRoles {
		if u.Role == r {
			known = true
			break
		}
	}
	if !known {
		return fmt.Errorf("%w: role %q is not recognized", ErrInvalid, u.Role)
	}
	return nil
}
