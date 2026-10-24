package mailers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/mailers"
)

var _ = Describe("Welcome", func() {
	It("links to the shop", func() {
		m := mailers.Welcome(domain.User{Email: "ann@example.com"}, "https://shop.example.com/")
		Expect(m.To).To(Equal("ann@example.com"))
		Expect(m.Subject).To(Equal("Welcome to Ecommerce"))
		// A trailing slash on the configured address does not double up.
		Expect(m.Text).To(ContainSubstring("Start shopping: https://shop.example.com\n"))
		Expect(m.HTML).To(ContainSubstring(`href="https://shop.example.com"`))
	})

	It("escapes the email in HTML", func() {
		m := mailers.Welcome(domain.User{Email: `a"<b>@example.com`}, "http://x")
		Expect(m.HTML).NotTo(ContainSubstring("<b>"))
	})
})
