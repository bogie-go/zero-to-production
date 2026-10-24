package mailers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/mailers"
	"ecommerce/app/services/mailer"
)

var _ = Describe("Welcome", func() {
	var (
		user    domain.User
		shopURL string
		m       mailer.Message
	)

	JustBeforeEach(func() {
		m = mailers.Welcome(user, shopURL)
	})

	// A trailing slash on the configured address does not double up.
	Context("with a shop URL ending in a slash", func() {
		BeforeEach(func() {
			user = domain.User{Email: "ann@example.com"}
			shopURL = "https://shop.example.com/"
		})

		It("links to the shop", func() {
			Expect(m.To).To(Equal("ann@example.com"))
			Expect(m.Subject).To(Equal("Welcome to Ecommerce"))
			Expect(m.Text).To(ContainSubstring("Start shopping: https://shop.example.com\n"))
			Expect(m.HTML).To(ContainSubstring(`href="https://shop.example.com"`))
		})
	})

	Context("with an email containing HTML", func() {
		BeforeEach(func() {
			user = domain.User{Email: `a"<b>@example.com`}
			shopURL = "http://x"
		})

		It("escapes the email in HTML", func() {
			Expect(m.HTML).NotTo(ContainSubstring("<b>"))
		})
	})
})
