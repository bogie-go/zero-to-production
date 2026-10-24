package middlewares_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/middlewares"
)

type fakeStaffStore struct {
	user domain.User
	err  error
}

func (f fakeStaffStore) GetUser(_ context.Context, id string) (domain.User, error) {
	return f.user, f.err
}

func staffRequest(store fakeStaffStore, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/me", middlewares.RequireUser(testTokens), middlewares.RequireStaff(store), func(c *gin.Context) {
		c.String(http.StatusOK, middlewares.Subject(c))
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/me", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("RequireStaff", func() {
	mint := func(subject string) string {
		good, err := testTokens.Mint(subject, time.Minute)
		Expect(err).NotTo(HaveOccurred())
		return "Bearer " + good
	}

	It("allows a staff user", func() {
		store := fakeStaffStore{user: domain.User{ID: 7, Role: domain.RoleStaff}}
		rec := staffRequest(store, mint("7"))
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Body.String()).To(Equal("7"))
	})

	It("rejects a customer with 403", func() {
		store := fakeStaffStore{user: domain.User{ID: 7, Role: domain.RoleCustomer}}
		Expect(staffRequest(store, mint("7")).Code).To(Equal(http.StatusForbidden))
	})

	// A guest's subject is a bare uuid, so the store answers not-found exactly
	// as an unknown or malformed subject would.
	It("rejects a guest token with 403", func() {
		store := fakeStaffStore{err: domain.ErrNotFound}
		Expect(staffRequest(store, mint("a-guest-uuid")).Code).To(Equal(http.StatusForbidden))
	})

	// No token at all never reaches RequireStaff; RequireUser answers first.
	It("needs RequireUser first: no token is 401", func() {
		store := fakeStaffStore{user: domain.User{ID: 7, Role: domain.RoleStaff}}
		Expect(staffRequest(store, "").Code).To(Equal(http.StatusUnauthorized))
	})
})
