package password

import (
	"errors"
	"testing"
	"time"

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

// DummyCheck always says no, and takes bcrypt's time doing it, so a login for
// an unknown email is not faster than one for a wrong password.
func TestDummyCheckAlwaysRefusesAndCostsAsMuchAsARealCheck(t *testing.T) {
	if DummyCheck("whatever-you-like") {
		t.Fatal("DummyCheck accepted a password")
	}
	digest, err := Hash("correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}
	measure := func(f func()) time.Duration { start := now(); f(); return since(start) }
	real := measure(func() { Check(digest, "wrong-password-here") })
	dummy := measure(func() { DummyCheck("wrong-password-here") })
	// Same work, so the same order of magnitude: not a hash-and-compare that
	// returns at once. (Generous bounds: this is a smoke test, not a benchmark.)
	if dummy < real/5 {
		t.Errorf("DummyCheck took %v against a real check's %v; it must cost about the same", dummy, real)
	}
}

var (
	now   = time.Now
	since = time.Since
)
