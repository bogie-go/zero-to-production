// Package password hashes and checks passwords with bcrypt — kept out of
// app/models, which imports nothing third-party (AGENTS.md rule 3), the
// same way Bogie's own api_key shape keeps bcrypt in its own service
// package rather than on models.ApiKey. A models.User only ever holds the
// digest this package produces; it never sees a plain password.
package password

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"ecommerce/app/models"
)

// minLength is Rails' has_secure_password default; good enough here too.
const minLength = 8

// Hash bcrypt-hashes plain, for storing as a User's PasswordDigest.
func Hash(plain string) (string, error) {
	if len(plain) < minLength {
		return "", fmt.Errorf("%w: password must be at least %d characters", models.ErrInvalid, minLength)
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

var (
	dummyOnce   sync.Once
	dummyDigest []byte
)

// DummyCheck spends the time a real Check would and always reports false. A
// login for an email with no account calls it, so answering "no such account"
// takes as long as answering "wrong password": otherwise the faster refusal
// tells a caller which emails have accounts, even though both say the same
// words. The digest it compares against is made once, at the same cost every
// real one is, from a value nothing can match.
func DummyCheck(plain string) bool {
	dummyOnce.Do(func() {
		dummyDigest, _ = bcrypt.GenerateFromPassword([]byte("no account has this password digest"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyDigest, []byte(plain))
	return false
}
