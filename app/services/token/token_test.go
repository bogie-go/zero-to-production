package token_test

import (
	"ecommerce/app/services/token"

	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var maker = token.Maker{Secret: []byte("a-test-secret-of-at-least-thirty-two-bytes"), Issuer: "test"}

var _ = Describe("Maker", func() {
	Describe("Mint then Verify", func() {
		It("round-trips the claims", func() {
			raw, err := maker.Mint("user-42", time.Minute)
			Expect(err).NotTo(HaveOccurred())

			claims, err := maker.Verify(raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(claims.Subject).To(Equal("user-42"))
			Expect(claims.ID).NotTo(BeEmpty())
			Expect(time.Until(claims.ExpiresAt)).To(BeNumerically(">", 0))
		})

		It("refuses to mint a token with no subject", func() {
			_, err := maker.Mint("", time.Minute)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Verify", func() {
		var expired, otherSecret, otherIssuer string

		BeforeEach(func() {
			expired, _ = maker.Mint("user-42", -time.Minute)
			otherSecret, _ = token.Maker{Secret: []byte("another-secret-of-at-least-thirty-two-bytes"), Issuer: "test"}.Mint("user-42", time.Minute)
			otherIssuer, _ = token.Maker{Secret: maker.Secret, Issuer: "someone-else"}.Mint("user-42", time.Minute)
		})

		It("refuses an expired token", func() {
			_, err := maker.Verify(expired)
			Expect(err).To(MatchError(token.ErrExpired))
		})

		It("refuses a token signed with another secret", func() {
			_, err := maker.Verify(otherSecret)
			Expect(err).To(MatchError(token.ErrInvalid))
		})

		It("refuses a token from another issuer", func() {
			_, err := maker.Verify(otherIssuer)
			Expect(err).To(MatchError(token.ErrInvalid))
		})

		DescribeTable("refuses malformed input",
			func(raw string) {
				_, err := maker.Verify(raw)
				Expect(err).To(MatchError(token.ErrInvalid))
			},
			Entry("garbage", "not.a.token"),
			Entry("empty", ""),
		)
	})
})
