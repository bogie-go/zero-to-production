package domain

import (
	"fmt"
	"strings"
	"time"
)

// User is a user. PasswordDigest is bcrypt's digest of the password, never
// the password itself — set and checked by app/services/password, kept out
// of here since this package imports nothing third-party (rule 3).
type User struct {
	ID             int64
	Email          string
	PasswordDigest string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Validate is the model's validations, as Rails would put them. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule.
func (u User) Validate() error {
	if strings.TrimSpace(u.Email) == "" || !strings.Contains(u.Email, "@") {
		return fmt.Errorf("%w: email must look like an email address", ErrInvalid)
	}
	if u.PasswordDigest == "" {
		return fmt.Errorf("%w: password is required", ErrInvalid)
	}
	return nil
}
