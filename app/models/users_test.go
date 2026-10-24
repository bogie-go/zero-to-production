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

	Describe("CreateUser", func() {
		It("creates a user", func() {
			digest, err := password.Hash("correct-horse-battery-staple")
			Expect(err).NotTo(HaveOccurred())

			created, err := store.CreateUser(GinkgoT().Context(), domain.User{Email: "a@example.com", PasswordDigest: digest, Role: domain.RoleCustomer})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.ID).NotTo(BeZero())
		})
	})

	Context("with a created user", func() {
		var created domain.User
		var id string

		BeforeEach(func() {
			digest, err := password.Hash("correct-horse-battery-staple")
			Expect(err).NotTo(HaveOccurred())
			created, err = store.CreateUser(GinkgoT().Context(), domain.User{Email: "a@example.com", PasswordDigest: digest, Role: domain.RoleCustomer})
			Expect(err).NotTo(HaveOccurred())
			id = strconv.FormatInt(created.ID, 10)
		})

		It("reads it", func() {
			Expect(store.GetUser(GinkgoT().Context(), id)).To(Equal(created))
		})

		It("lists it", func() {
			all, err := store.ListUsers(GinkgoT().Context(), 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(all).To(HaveLen(1))
			Expect(all[0].ID).To(Equal(created.ID))
		})

		It("updates it", func() {
			created.Email = "b@example.com"
			updated, err := store.UpdateUser(GinkgoT().Context(), created)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Email).To(Equal("b@example.com"))
		})

		It("deletes it", func() {
			Expect(store.DeleteUser(GinkgoT().Context(), id)).To(Succeed())
			_, err := store.GetUser(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})
	})
})
