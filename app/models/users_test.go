package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/services/password"
)

var _ = Describe("Users", func() {

	DescribeTable("an unknown user is not found",
		func(id string) {
			_, err := store.GetUser(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(store.DeleteUser(GinkgoT().Context(), id)).To(MatchError(domain.ErrNotFound))
		},
		Entry("a malformed id", "nope"),
		Entry("an empty id", ""),
		Entry("a well-formed id that names nothing", "999999999"),
	)

	It("cannot update one that is not there", func() {
		// A valid-looking user, so Validate passes and the not-found check is
		// what gets exercised.
		_, err := store.UpdateUser(GinkgoT().Context(), domain.User{ID: 999999999, Email: "nope@example.com", PasswordDigest: "digest", Role: domain.RoleCustomer})
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	It("is created, read, listed, updated and deleted", func() {
		ctx := GinkgoT().Context()
		digest, err := password.Hash("correct-horse-battery-staple")
		Expect(err).NotTo(HaveOccurred())

		created, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: digest, Role: domain.RoleCustomer})
		Expect(err).NotTo(HaveOccurred())
		Expect(created.ID).NotTo(BeZero())
		id := strconv.FormatInt(created.ID, 10)

		Expect(store.GetUser(ctx, id)).To(Equal(created))
		all, err := store.ListUsers(ctx, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(all).To(HaveLen(1))
		Expect(all[0].ID).To(Equal(created.ID))

		created.Email = "b@example.com"
		updated, err := store.UpdateUser(ctx, created)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Email).To(Equal("b@example.com"))

		Expect(store.DeleteUser(ctx, id)).To(Succeed())
		_, err = store.GetUser(ctx, id)
		Expect(err).To(MatchError(domain.ErrNotFound))
	})
})
