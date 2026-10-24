package domain_test

import (
	"ecommerce/app/domain"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("User.Validate", func() {
	DescribeTable("accepts",
		func(user domain.User) {
			Expect(user.Validate()).To(Succeed())
		},
		Entry("a customer", domain.User{Email: "a@example.com", PasswordDigest: "digest", Role: domain.RoleCustomer}),
		Entry("a staff member", domain.User{Email: "a@example.com", PasswordDigest: "digest", Role: domain.RoleStaff}),
	)

	DescribeTable("rejects",
		func(user domain.User) {
			Expect(user.Validate()).To(MatchError(domain.ErrInvalid))
		},
		Entry("an empty email", domain.User{Email: "", PasswordDigest: "digest", Role: domain.RoleCustomer}),
		Entry("a blank email", domain.User{Email: "   ", PasswordDigest: "digest", Role: domain.RoleCustomer}),
		Entry("an email with no @", domain.User{Email: "not-an-email", PasswordDigest: "digest", Role: domain.RoleCustomer}),
		Entry("an empty password digest", domain.User{Email: "a@example.com", PasswordDigest: "", Role: domain.RoleCustomer}),
		Entry("an unknown role", domain.User{Email: "a@example.com", PasswordDigest: "digest", Role: "owner"}),
	)
})
