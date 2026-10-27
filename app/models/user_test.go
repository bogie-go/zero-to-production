package models_test

import (
	"ecommerce/app/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("User.Validate", func() {
	DescribeTable("accepts",
		func(user models.User) {
			Expect(user.Validate()).To(Succeed())
		},
		Entry("a customer", models.User{Email: "a@example.com", PasswordDigest: "digest", Role: models.RoleCustomer}),
		Entry("a staff member", models.User{Email: "a@example.com", PasswordDigest: "digest", Role: models.RoleStaff}),
	)

	DescribeTable("rejects",
		func(user models.User) {
			Expect(user.Validate()).To(MatchError(models.ErrInvalid))
		},
		Entry("an empty email", models.User{Email: "", PasswordDigest: "digest", Role: models.RoleCustomer}),
		Entry("a blank email", models.User{Email: "   ", PasswordDigest: "digest", Role: models.RoleCustomer}),
		Entry("an email with no @", models.User{Email: "not-an-email", PasswordDigest: "digest", Role: models.RoleCustomer}),
		Entry("an empty password digest", models.User{Email: "a@example.com", PasswordDigest: "", Role: models.RoleCustomer}),
		Entry("an unknown role", models.User{Email: "a@example.com", PasswordDigest: "digest", Role: "owner"}),
	)
})
