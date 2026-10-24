package mailers_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/mailers"
)

var signIn = domain.SignIn{UserID: 3, At: time.Date(2026, 10, 22, 14, 30, 0, 0, time.UTC), IP: "203.0.113.9", UserAgent: "Mozilla/5.0 (Mac)"}

var _ = Describe("SignInAlert", func() {
	It("says when, from where and with what", func() {
		m := mailers.SignInAlert(domain.User{Email: "ann@example.com"}, signIn)
		Expect(m.To).To(Equal("ann@example.com"))
		Expect(m.Subject).To(Equal("New sign-in to your account"))
		for _, want := range []string{"Thu, 22 Oct 2026 14:30 UTC", "203.0.113.9", "Mozilla/5.0 (Mac)"} {
			Expect(m.Text).To(ContainSubstring(want))
			Expect(m.HTML).To(ContainSubstring(want))
		}
	})

	// The address and browser string come from the request, so the HTML part
	// escapes them, and an absurd one cannot fill the email.
	It("distrusts the request", func() {
		s := signIn
		s.UserAgent = `<script>alert(1)</script>` + strings.Repeat("x", 500)
		m := mailers.SignInAlert(domain.User{Email: "a@example.com"}, s)
		Expect(m.HTML).NotTo(ContainSubstring("<script>"))
		Expect(m.HTML).To(ContainSubstring("&lt;script&gt;"))
		Expect(strings.Count(m.Text, "x")).To(BeNumerically("<=", mailers.MaxUserAgent+20))
	})

	It("tolerates a missing address", func() {
		s := signIn
		s.IP, s.UserAgent = "", ""
		m := mailers.SignInAlert(domain.User{Email: "a@example.com"}, s)
		Expect(m.Text).To(ContainSubstring("unknown"))
	})
})
