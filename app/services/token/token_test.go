package token_test

import (
	"ecommerce/app/services/token"

	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var maker = token.Maker{Secret: []byte("a-test-secret-of-at-least-thirty-two-bytes"), Issuer: "test"}

var _ = Describe("Maker", func() {
	Describe("Mint and Verify", func() {
		var (
			raw    string
			mintEr error
		)

		Context("with a subject", func() {
			BeforeEach(func() {
				raw, mintEr = maker.Mint("user-42", time.Minute)
			})

			It("round-trips the claims", func() {
				Expect(mintEr).NotTo(HaveOccurred())

				claims, err := maker.Verify(raw)
				Expect(err).NotTo(HaveOccurred())
				Expect(claims.Subject).To(Equal("user-42"))
				Expect(claims.ID).NotTo(BeEmpty())
				Expect(time.Until(claims.ExpiresAt)).To(BeNumerically(">", 0))
			})
		})

		Context("without a subject", func() {
			BeforeEach(func() {
				_, mintEr = maker.Mint("", time.Minute)
			})

			It("refuses to mint a token", func() {
				Expect(mintEr).To(HaveOccurred())
			})
		})
	})

	Describe("Verify", func() {
		var (
			raw string
			err error
		)

		JustBeforeEach(func() {
			_, err = maker.Verify(raw)
		})

		Context("with an expired token", func() {
			BeforeEach(func() {
				raw, _ = maker.Mint("user-42", -time.Minute)
			})

			It("refuses it as expired", func() {
				Expect(err).To(MatchError(token.ErrExpired))
			})
		})

		Context("with a token signed with another secret", func() {
			BeforeEach(func() {
				raw, _ = token.Maker{Secret: []byte("another-secret-of-at-least-thirty-two-bytes"), Issuer: "test"}.Mint("user-42", time.Minute)
			})

			It("refuses it as invalid", func() {
				Expect(err).To(MatchError(token.ErrInvalid))
			})
		})

		Context("with a token from another issuer", func() {
			BeforeEach(func() {
				raw, _ = token.Maker{Secret: maker.Secret, Issuer: "someone-else"}.Mint("user-42", time.Minute)
			})

			It("refuses it as invalid", func() {
				Expect(err).To(MatchError(token.ErrInvalid))
			})
		})

		DescribeTable("refuses malformed input",
			func(input string) {
				_, err := maker.Verify(input)
				Expect(err).To(MatchError(token.ErrInvalid))
			},
			Entry("garbage", "not.a.token"),
			Entry("empty", ""),
		)
	})
})
