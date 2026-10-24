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
	var (
		store fakeStaffStore
		auth  string
		rec   *httptest.ResponseRecorder
	)

	mint := func(subject string) string {
		good, err := testTokens.Mint(subject, time.Minute)
		Expect(err).NotTo(HaveOccurred())
		return "Bearer " + good
	}

	JustBeforeEach(func() {
		rec = staffRequest(store, auth)
	})

	Context("when the user is staff", func() {
		BeforeEach(func() {
			store = fakeStaffStore{user: domain.User{ID: 7, Role: domain.RoleStaff}}
			auth = mint("7")
		})

		It("allows the request", func() {
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(Equal("7"))
		})
	})

	Context("when the user is a customer", func() {
		BeforeEach(func() {
			store = fakeStaffStore{user: domain.User{ID: 7, Role: domain.RoleCustomer}}
			auth = mint("7")
		})

		It("rejects with 403", func() {
			Expect(rec.Code).To(Equal(http.StatusForbidden))
		})
	})

	// A guest's subject is a bare uuid, so the store answers not-found exactly
	// as an unknown or malformed subject would.
	Context("with a guest token", func() {
		BeforeEach(func() {
			store = fakeStaffStore{err: domain.ErrNotFound}
			auth = mint("a-guest-uuid")
		})

		It("rejects with 403", func() {
			Expect(rec.Code).To(Equal(http.StatusForbidden))
		})
	})

	// No token at all never reaches RequireStaff; RequireUser answers first.
	Context("without a token", func() {
		BeforeEach(func() {
			store = fakeStaffStore{user: domain.User{ID: 7, Role: domain.RoleStaff}}
			auth = ""
		})

		It("answers 401 from RequireUser first", func() {
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})
