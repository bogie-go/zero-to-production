package models_test

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/test/testdb"
)

var _ = Describe("Users by email", func() {

	newUser := func(email string) domain.User {
		return domain.User{Email: email, PasswordDigest: "digest", Role: domain.RoleCustomer}
	}

	Describe("CreateUser", func() {
		Context("with a user a@example.com", func() {
			BeforeEach(func() {
				_, err := store.CreateUser(GinkgoT().Context(), newUser("a@example.com"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("refuses a duplicate email", func() {
				_, err := store.CreateUser(GinkgoT().Context(), newUser("a@example.com"))
				Expect(err).To(MatchError(domain.ErrInvalid))
			})
		})
	})

	Describe("GetUserByEmail", func() {
		Context("with a user a@example.com", func() {
			var created domain.User

			BeforeEach(func() {
				var err error
				created, err = store.CreateUser(GinkgoT().Context(), newUser("a@example.com"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("finds the user", func() {
				found, err := store.GetUserByEmail(GinkgoT().Context(), "a@example.com")
				Expect(err).NotTo(HaveOccurred())
				Expect(found.ID).To(Equal(created.ID))
			})

			It("says not found for an unknown email", func() {
				_, err := store.GetUserByEmail(GinkgoT().Context(), "nobody@example.com")
				Expect(err).To(MatchError(domain.ErrNotFound))
			})
		})
	})

	Describe("RegisterUser", func() {
		// The hook runs inside the signup's transaction, with the new user,
		// and what it writes commits with the user.
		Context("with a hook that looks at the new user from outside the transaction", func() {
			var seen domain.User
			var visibleOutside int

			BeforeEach(func() {
				pool := testdb.Pool(GinkgoT())
				seen, visibleOutside = domain.User{}, 0
				store.OnUserRegistered(func(ctx context.Context, tx pgx.Tx, user domain.User) error {
					seen = user
					return pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE id = $1", user.ID).Scan(&visibleOutside)
				})
			})

			It("runs the OnUserRegistered hook inside the transaction", func() {
				user, err := store.RegisterUser(GinkgoT().Context(), newUser("new@example.com"))

				Expect(err).NotTo(HaveOccurred())
				Expect(seen.ID).To(Equal(user.ID))
				Expect(seen.Email).To(Equal("new@example.com"))
				Expect(visibleOutside).To(BeZero(), "not visible outside before commit")
				got, err := store.GetUserByEmail(GinkgoT().Context(), "new@example.com")
				Expect(err).NotTo(HaveOccurred())
				Expect(got.ID).To(Equal(user.ID))
			})
		})

		// A hook that fails (the queue is down) means no account.
		Context("with a hook that fails", func() {
			BeforeEach(func() {
				store.OnUserRegistered(func(context.Context, pgx.Tx, domain.User) error { return errors.New("queue is down") })
			})

			It("refuses the signup", func() {
				_, err := store.RegisterUser(GinkgoT().Context(), newUser("new@example.com"))

				Expect(err).To(HaveOccurred())
				_, err = store.GetUserByEmail(GinkgoT().Context(), "new@example.com")
				Expect(err).To(MatchError(domain.ErrNotFound))
			})
		})

		// Same answers as CreateUser: a duplicate is ErrInvalid and runs no
		// hook; an invalid user is refused before anything is written.
		Context("with a counting hook and a user already registered", func() {
			var calls int

			BeforeEach(func() {
				calls = 0
				store.OnUserRegistered(func(context.Context, pgx.Tx, domain.User) error { calls++; return nil })
				_, err := store.RegisterUser(GinkgoT().Context(), newUser("a@example.com"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("refuses a duplicate without running the hook", func() {
				_, err := store.RegisterUser(GinkgoT().Context(), newUser("a@example.com"))
				Expect(err).To(MatchError(domain.ErrInvalid))
				Expect(err).To(MatchError(ContainSubstring("already taken")))
				Expect(calls).To(Equal(1), "only the first signup ran the hook")
			})

			It("refuses an invalid user without running the hook", func() {
				_, err := store.RegisterUser(GinkgoT().Context(), newUser("nope"))
				Expect(err).To(MatchError(domain.ErrInvalid))
				Expect(calls).To(Equal(1), "only the first signup ran the hook")
			})
		})
	})
})
