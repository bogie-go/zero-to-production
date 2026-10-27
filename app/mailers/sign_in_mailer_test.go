package mailers_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/mailers"
	"ecommerce/app/models"
	"ecommerce/app/services/mailer"
)

var signIn = models.SignIn{UserID: 3, At: time.Date(2026, 10, 22, 14, 30, 0, 0, time.UTC), IP: "203.0.113.9", UserAgent: "Mozilla/5.0 (Mac)"}

var _ = Describe("SignInAlert", func() {
	var (
		m       mailer.Message
		request models.SignIn
	)

	BeforeEach(func() {
		request = signIn
	})

	JustBeforeEach(func() {
		m = mailers.SignInAlert(models.User{Email: "ann@example.com"}, request)
	})

	Context("with a full sign-in", func() {
		It("says when, from where and with what", func() {
			Expect(m.To).To(Equal("ann@example.com"))
			Expect(m.Subject).To(Equal("New sign-in to your Bogie Depot account"))
			for _, want := range []string{"Thu, 22 Oct 2026 14:30 UTC", "203.0.113.9", "Mozilla/5.0 (Mac)"} {
				Expect(m.Text).To(ContainSubstring(want))
				Expect(m.HTML).To(ContainSubstring(want))
			}
		})
	})

	// The address and browser string come from the request, so the HTML part
	// escapes them, and an absurd one cannot fill the email.
	Context("with a hostile and oversized user agent", func() {
		BeforeEach(func() {
			request.UserAgent = `<script>alert(1)</script>` + strings.Repeat("x", 500)
		})

		It("escapes and truncates it", func() {
			Expect(m.HTML).NotTo(ContainSubstring("<script>"))
			Expect(m.HTML).To(ContainSubstring("&lt;script&gt;"))
			Expect(strings.Count(m.Text, "x")).To(BeNumerically("<=", mailers.MaxUserAgent+20))
		})
	})

	Context("without an address or user agent", func() {
		BeforeEach(func() {
			request.IP, request.UserAgent = "", ""
		})

		It("says unknown", func() {
			Expect(m.Text).To(ContainSubstring("unknown"))
		})
	})
})
