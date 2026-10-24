package controllers_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers"
	"ecommerce/app/domain"
	"ecommerce/app/middlewares"
)

type fakeMeStore struct {
	user domain.User
	err  error
}

func (f fakeMeStore) GetUser(_ context.Context, id string) (domain.User, error) {
	return f.user, f.err
}

func meRequest(store fakeMeStore, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	guarded := router.Group("", middlewares.RequireUser(testTokens))
	controllers.NewMeController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(guarded)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("GET /me", func() {
	It("returns the account", func() {
		store := fakeMeStore{user: domain.User{ID: 3, Email: "a@example.com", Role: domain.RoleCustomer}}
		rec := meRequest(store, bearerFor("3"))

		Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
		Expect(rec.Body.String()).To(ContainSubstring(`"email":"a@example.com"`))
	})

	It("rejects a guest token with 403", func() {
		rec := meRequest(fakeMeStore{err: domain.ErrNotFound}, bearerFor("a-guest-uuid"))
		Expect(rec.Code).To(Equal(http.StatusForbidden))
	})

	It("needs a token", func() {
		Expect(meRequest(fakeMeStore{}, "").Code).To(Equal(http.StatusUnauthorized))
	})
})
