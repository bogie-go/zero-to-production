package db_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/app/services/password"
	"ecommerce/test/factories"
)

var _ = Describe("Users", func() {
	Describe("GetUser", func() {
		DescribeTable("is not found for",
			func(id string) {
				_, err := store.GetUser(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved user", func() {
			var user models.User

			BeforeEach(func() {
				user = factories.User(GinkgoT(), store)
			})

			It("finds the user by id", func() {
				Expect(store.GetUser(GinkgoT().Context(), strconv.FormatInt(user.ID, 10))).To(Equal(user))
			})
		})
	})

	Describe("ListUsers", func() {
		Context("with a saved user", func() {
			var user models.User

			BeforeEach(func() {
				user = factories.User(GinkgoT(), store)
			})

			It("includes the saved user", func() {
				all, err := store.ListUsers(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ID).To(Equal(user.ID))
			})
		})
	})

	Describe("CreateUser", func() {
		Context("with a hashed password", func() {
			var digest string

			BeforeEach(func() {
				var err error
				digest, err = password.Hash("correct-horse-battery-staple")
				Expect(err).NotTo(HaveOccurred())
			})

			It("saves the user with an id", func() {
				created, err := store.CreateUser(GinkgoT().Context(), models.User{Email: "a@example.com", PasswordDigest: digest, Role: models.RoleCustomer})
				Expect(err).NotTo(HaveOccurred())
				Expect(created.ID).NotTo(BeZero())
			})
		})
	})

	Describe("UpdateUser", func() {
		Context("with a saved user", func() {
			var user models.User

			BeforeEach(func() {
				user = factories.User(GinkgoT(), store)
			})

			It("stores the change", func() {
				user.Email = "b@example.com"
				updated, err := store.UpdateUser(GinkgoT().Context(), user)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Email).To(Equal("b@example.com"))
			})
		})

		Context("when the user is not there", func() {
			It("is not found", func() {
				// A valid-looking user, so Validate passes and the not-found
				// check is what gets exercised.
				_, err := store.UpdateUser(GinkgoT().Context(), models.User{ID: 999999999, Email: "nope@example.com", PasswordDigest: "digest", Role: models.RoleCustomer})
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})

	Describe("DeleteUser", func() {
		DescribeTable("is not found for",
			func(id string) {
				Expect(store.DeleteUser(GinkgoT().Context(), id)).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved user", func() {
			var user models.User

			BeforeEach(func() {
				user = factories.User(GinkgoT(), store)
			})

			It("removes the user", func() {
				id := strconv.FormatInt(user.ID, 10)
				Expect(store.DeleteUser(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetUser(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})
})
