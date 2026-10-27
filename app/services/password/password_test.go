package password_test

import (
	"ecommerce/app/services/password"

	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
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

	Context("with a short password", func() {
		var hashErr error

		BeforeEach(func() {
			_, hashErr = password.Hash("short")
		})

		It("rejects it", func() {
			Expect(hashErr).To(MatchError(models.ErrInvalid))
		})
	})
})

// DummyCheck always says no, and takes bcrypt's time doing it, so a login for
// an unknown email is not faster than one for a wrong password.
var _ = Describe("DummyCheck", func() {
	It("always refuses", func() {
		Expect(password.DummyCheck("whatever-you-like")).To(BeFalse())
	})

	Context("with a real digest to compare against", func() {
		var digest string

		BeforeEach(func() {
			var err error
			digest, err = password.Hash("correct-horse-battery")
			Expect(err).NotTo(HaveOccurred())
		})

		It("costs about as much as a real check", func() {
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
})
