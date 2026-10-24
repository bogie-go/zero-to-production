package database_test

import (
	"ecommerce/app/services/database"

	"errors"
	"fmt"
	"syscall"

	"github.com/jackc/pgx/v5/pgconn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Maintenance", func() {
	var (
		name, admin string
		err         error
	)

	Context("with a URL naming a database", func() {
		BeforeEach(func() {
			name, admin, err = database.Maintenance("postgres://u:secret@localhost:5440/ecommerce_test?sslmode=disable")
		})

		It("names the database and points at the same server's postgres database", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("ecommerce_test"))
			Expect(admin).To(Equal("postgres://u:secret@localhost:5440/postgres?sslmode=disable"))
		})
	})

	Context("with a URL without a database", func() {
		BeforeEach(func() {
			_, _, err = database.Maintenance("postgres://localhost:5440/")
		})

		It("refuses it", func() {
			Expect(err).To(HaveOccurred())
		})
	})
})

// The three failures Hint exists to tell apart. Issue #11 reported the first
// one as a bare pgx dump, which reads the same whether Postgres is down, the
// password is wrong, or the database was never created.
var _ = Describe("Hint", func() {
	const bundled = "postgres://postgres:postgres@localhost:5440/ecommerce_development?sslmode=disable"

	var (
		refused error
		got     string
	)

	BeforeEach(func() {
		refused = fmt.Errorf("failed to connect: dial tcp 127.0.0.1:5440: connect: %w", syscall.ECONNREFUSED)
	})

	Context("when the connection is refused on the bundled port", func() {
		BeforeEach(func() {
			got = database.Hint(refused, bundled).Error()
		})

		It("says where to start Postgres and keeps the original error", func() {
			Expect(got).To(ContainSubstring("localhost:5440"))
			Expect(got).To(ContainSubstring("make up"))
			Expect(got).To(ContainSubstring("bogie db:prepare"))
			Expect(got).To(ContainSubstring("connect: connection refused"))
		})
	})

	// Someone else's Postgres: `make up` would be the wrong advice.
	Context("when the connection is refused on another port", func() {
		BeforeEach(func() {
			got = database.Hint(refused, "postgres://me@localhost:5432/ecommerce_development").Error()
		})

		It("explains how to repoint instead of running make up", func() {
			Expect(got).NotTo(ContainSubstring("make up"))
			Expect(got).To(ContainSubstring("DATABASE_URL"))
		})
	})

	Context("when the server rejects the credentials", func() {
		BeforeEach(func() {
			got = database.Hint(&pgconn.PgError{Code: database.InvalidPassword, Message: "password authentication failed"}, bundled).Error()
		})

		It("reads as an auth failure, not as unreachable", func() {
			Expect(got).To(ContainSubstring("credentials"))
			Expect(got).NotTo(ContainSubstring("Nothing is listening"))
		})
	})

	Context("when the database does not exist", func() {
		BeforeEach(func() {
			got = database.Hint(&pgconn.PgError{Code: database.InvalidCatalogName, Message: "database does not exist"}, bundled).Error()
		})

		It("reads as a missing database, not as unreachable", func() {
			Expect(got).To(ContainSubstring("bogie db:prepare"))
			Expect(got).NotTo(ContainSubstring("Nothing is listening"))
		})
	})

	Context("with no error", func() {
		It("returns nil", func() {
			Expect(database.Hint(nil, "postgres://localhost:5440/x")).To(BeNil())
		})
	})

	// A syntax error in a query is not a connection problem; adding "start
	// Postgres" to it would send the reader the wrong way.
	Context("with an unrelated error", func() {
		var other error

		BeforeEach(func() {
			other = errors.New("models: create widget: syntax error at or near \"slect\"")
		})

		It("leaves it alone", func() {
			Expect(database.Hint(other, "postgres://localhost:5440/x")).To(MatchError(other.Error()))
		})
	})
})
