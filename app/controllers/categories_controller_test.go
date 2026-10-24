package controllers_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers"
	"ecommerce/app/domain"
)

type fakeCategoriesStore struct {
	list []domain.Category
	err  error
}

func (f fakeCategoriesStore) ListCategoriesWithCounts(context.Context) ([]domain.Category, error) {
	return f.list, f.err
}

func categoriesRequest(store fakeCategoriesStore) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controllers.NewCategoriesController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(&router.RouterGroup)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/categories", nil))
	return rec
}

var _ = Describe("GET /categories", func() {
	It("lists each category with its product count", func() {
		rec := categoriesRequest(fakeCategoriesStore{list: []domain.Category{
			{ID: 1, Name: "Audio", ProductCount: 3}, {ID: 2, Name: "Empty"},
		}})

		Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
		Expect(rec.Body.String()).To(ContainSubstring(`"name":"Audio","product_count":3`))
		Expect(rec.Body.String()).To(ContainSubstring(`"name":"Empty","product_count":0`))
	})

	// No categories is an empty list, not null: the UI iterates it.
	It("is an empty list, not null, with no categories", func() {
		Expect(categoriesRequest(fakeCategoriesStore{}).Body.String()).To(ContainSubstring(`"categories":[]`))
	})

	It("answers 500 without leaking the cause", func() {
		rec := categoriesRequest(fakeCategoriesStore{err: errors.New("connection refused to db-1")})

		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		Expect(rec.Body.String()).NotTo(ContainSubstring("db-1"))
	})
})
