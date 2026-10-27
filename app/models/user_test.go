package models_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
)

var _ = Describe("User.Validate", func() {
	var user models.User

	BeforeEach(func() {
		user = models.User{Email: "a@example.com", PasswordDigest: "digest", Role: models.RoleCustomer}
	})

	Context("with an email, a password digest and the customer role", func() {
		It("is valid", func() {
			Expect(user.Validate()).To(Succeed())
		})
	})

	// The only rule on the address is that it has an @: whether it can receive
	// mail is for the confirmation email to find out, not for Validate.
	Context("with an email on a host with no dot", func() {
		BeforeEach(func() {
			user.Email = "a@localhost"
		})

		It("is valid", func() {
			Expect(user.Validate()).To(Succeed())
		})
	})

	Context("with the staff role", func() {
		BeforeEach(func() {
			user.Role = models.RoleStaff
		})

		It("is valid", func() {
			Expect(user.Validate()).To(Succeed())
		})
	})

	DescribeTable("is refused for",
		func(change func(*models.User), reason string) {
			change(&user)

			err := user.Validate()

			Expect(err).To(MatchError(models.ErrInvalid))
			Expect(err).To(MatchError(ContainSubstring(reason)))
		},
		Entry("an empty email", func(u *models.User) { u.Email = "" }, "email must look like an email address"),
		Entry("a blank email", func(u *models.User) { u.Email = "   " }, "email must look like an email address"),
		Entry("an email with no @", func(u *models.User) { u.Email = "not-an-email" }, "email must look like an email address"),
		Entry("a dotted name with no @", func(u *models.User) { u.Email = "first.last" }, "email must look like an email address"),
		Entry("an empty password digest", func(u *models.User) { u.PasswordDigest = "" }, "password is required"),
		Entry("an unknown role", func(u *models.User) { u.Role = "owner" }, `role "owner" is not recognized`),
	)
})
