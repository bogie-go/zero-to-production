// Package password hashes and checks passwords with bcrypt — kept out of
// app/domain, which imports nothing third-party (AGENTS.md rule 3), the
// same way Bogie's own api_key shape keeps bcrypt in its own service
// package rather than on domain.ApiKey. A domain.User only ever holds the
// digest this package produces; it never sees a plain password.
package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"ecommerce/app/domain"
)

// minLength is Rails' has_secure_password default; good enough here too.
const minLength = 8

// Hash bcrypt-hashes plain, for storing as a User's PasswordDigest.
func Hash(plain string) (string, error) {
	if len(plain) < minLength {
		return "", fmt.Errorf("%w: password must be at least %d characters", domain.ErrInvalid, minLength)
	}
	digest, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("password: hash: %w", err)
	}
	return string(digest), nil
}

// Check reports whether plain is the password digest hashed.
func Check(digest, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(digest), []byte(plain)) == nil
}
