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

var _ = Describe("maintenance", func() {
	It("names the database and points at the same server's postgres database", func() {
		name, admin, err := database.Maintenance("postgres://u:secret@localhost:5440/ecommerce_test?sslmode=disable")
		Expect(err).NotTo(HaveOccurred())
		Expect(name).To(Equal("ecommerce_test"))
		Expect(admin).To(Equal("postgres://u:secret@localhost:5440/postgres?sslmode=disable"))
	})

	It("refuses a URL without a database", func() {
		_, _, err := database.Maintenance("postgres://localhost:5440/")
		Expect(err).To(HaveOccurred())
	})
})

// The three failures Hint exists to tell apart. Issue #11 reported the first
// one as a bare pgx dump, which reads the same whether Postgres is down, the
// password is wrong, or the database was never created.
var _ = Describe("Hint", func() {
	const bundled = "postgres://postgres:postgres@localhost:5440/ecommerce_development?sslmode=disable"

	refused := fmt.Errorf("failed to connect: dial tcp 127.0.0.1:5440: connect: %w", syscall.ECONNREFUSED)

	When("the connection is refused on the bundled port", func() {
		It("says where to start Postgres and keeps the original error", func() {
			got := database.Hint(refused, bundled).Error()
			Expect(got).To(ContainSubstring("localhost:5440"))
			Expect(got).To(ContainSubstring("make up"))
			Expect(got).To(ContainSubstring("bogie db:prepare"))
			Expect(got).To(ContainSubstring("connect: connection refused"))
		})
	})

	// Someone else's Postgres: `make up` would be the wrong advice.
	When("the connection is refused on another port", func() {
		It("explains how to repoint instead of running make up", func() {
			own := database.Hint(refused, "postgres://me@localhost:5432/ecommerce_development").Error()
			Expect(own).NotTo(ContainSubstring("make up"))
			Expect(own).To(ContainSubstring("DATABASE_URL"))
		})
	})

	When("the server rejects the credentials", func() {
		It("reads as an auth failure, not as unreachable", func() {
			auth := database.Hint(&pgconn.PgError{Code: database.InvalidPassword, Message: "password authentication failed"}, bundled).Error()
			Expect(auth).To(ContainSubstring("credentials"))
			Expect(auth).NotTo(ContainSubstring("Nothing is listening"))
		})
	})

	When("the database does not exist", func() {
		It("reads as a missing database, not as unreachable", func() {
			missing := database.Hint(&pgconn.PgError{Code: database.InvalidCatalogName, Message: "database does not exist"}, bundled).Error()
			Expect(missing).To(ContainSubstring("bogie db:prepare"))
			Expect(missing).NotTo(ContainSubstring("Nothing is listening"))
		})
	})

	Describe("what it does not know", func() {
		It("returns nil for nil", func() {
			Expect(database.Hint(nil, "postgres://localhost:5440/x")).To(BeNil())
		})

		// A syntax error in a query is not a connection problem; adding "start
		// Postgres" to it would send the reader the wrong way.
		It("leaves an unrelated error alone", func() {
			other := errors.New("models: create widget: syntax error at or near \"slect\"")
			Expect(database.Hint(other, "postgres://localhost:5440/x")).To(MatchError(other.Error()))
		})
	})
})
