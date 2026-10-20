package password

import (
	"errors"
	"testing"

	"ecommerce/app/domain"
)

func TestHashAndCheck(t *testing.T) {
	digest, err := Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatal(err)
	}
	if !Check(digest, "correct-horse-battery-staple") {
		t.Error("Check(digest, the password that was hashed) = false, want true")
	}
	if Check(digest, "wrong-password") {
		t.Error("Check(digest, a different password) = true, want false")
	}
}

func TestHashRejectsAShortPassword(t *testing.T) {
	if _, err := Hash("short"); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("Hash(short password): err = %v, want ErrInvalid", err)
	}
}
