package password_test

import (
	"ecommerce/app/services/password"

	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
)

var _ = Describe("Hash and Check", func() {
	var digest string

	BeforeEach(func() {
		var err error
		digest, err = password.Hash("correct-horse-battery-staple")
		Expect(err).NotTo(HaveOccurred())
	})

	It("accepts the password that was hashed", func() {
		Expect(password.Check(digest, "correct-horse-battery-staple")).To(BeTrue())
	})

	It("refuses a different password", func() {
		Expect(password.Check(digest, "wrong-password")).To(BeFalse())
	})

	It("rejects a short password", func() {
		_, err := password.Hash("short")
		Expect(err).To(MatchError(domain.ErrInvalid))
	})
})

// DummyCheck always says no, and takes bcrypt's time doing it, so a login for
// an unknown email is not faster than one for a wrong password.
var _ = Describe("DummyCheck", func() {
	It("always refuses", func() {
		Expect(password.DummyCheck("whatever-you-like")).To(BeFalse())
	})

	It("costs about as much as a real check", func() {
		digest, err := password.Hash("correct-horse-battery")
		Expect(err).NotTo(HaveOccurred())

		measure := func(f func()) time.Duration {
			start := time.Now()
			f()
			return time.Since(start)
		}
		real := measure(func() { password.Check(digest, "wrong-password-here") })
		dummy := measure(func() { password.DummyCheck("wrong-password-here") })

		// Same work, so the same order of magnitude: not a hash-and-compare that
		// returns at once. (Generous bounds: this is a smoke test, not a benchmark.)
		Expect(dummy).To(BeNumerically(">=", real/5))
	})
})
