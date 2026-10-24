package controllers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers"
	"ecommerce/app/domain"
)

// fakeProductsStore is a fake of the store interface declared beside the
// controller: it records what the controller asked for and answers with
// what the test chose.
type fakeProductsStore struct {
	one  domain.Product
	many []domain.Product
	err  error

	total int64

	gotID         string
	gotCategoryID *int64
	gotLimit      int32
	gotOffset     int32
}

func (f *fakeProductsStore) GetActiveProduct(_ context.Context, id string) (domain.Product, error) {
	f.gotID = id
	return f.one, f.err
}

func (f *fakeProductsStore) ListActiveProducts(_ context.Context, categoryID *int64, limit, offset int32) ([]domain.Product, error) {
	f.gotCategoryID = categoryID
	f.gotLimit, f.gotOffset = limit, offset
	return f.many, f.err
}

func (f *fakeProductsStore) CountActiveProducts(_ context.Context, _ *int64) (int64, error) {
	return f.total, f.err
}

// productsRequest mounts the controller alone on a bare router, so these
// tests see the controller and nothing else: no middleware, no other routes.
func productsRequest(store *fakeProductsStore, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controllers.NewProductsController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(&router.RouterGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("products", func() {
	var store *fakeProductsStore

	BeforeEach(func() {
		store = &fakeProductsStore{}
	})

	// Every action is mounted: a route that answers 404 was never registered.
	DescribeTable("routes",
		func(method, path string) {
			rec := productsRequest(store, method, path, "")
			Expect(rec.Code).NotTo(Equal(http.StatusNotFound))
		},
		Entry("GET /products", http.MethodGet, "/products"),
		Entry("GET /products/:id", http.MethodGet, "/products/1"),
	)

	Describe("GET /products", func() {
		It("lists", func() {
			store.many = []domain.Product{{ID: 1}, {ID: 2}}

			rec := productsRequest(store, http.MethodGet, "/products", "")

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			var body struct {
				Products []struct{ ID int64 } `json:"products"`
			}
			Expect(json.Unmarshal(rec.Body.Bytes(), &body)).To(Succeed())
			Expect(body.Products).To(HaveLen(2))
			Expect(body.Products[1].ID).To(Equal(int64(2)))
		})

		// An unexpected error is a 500 that says nothing about its cause.
		It("answers 500 without detail for an unexpected error", func() {
			store.err = errors.New("connection refused to db-1")

			rec := productsRequest(store, http.MethodGet, "/products", "")

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			Expect(rec.Body.String()).NotTo(ContainSubstring("db-1"))
		})

		// ?category_id=N is handed to the store as a number; without it the store is
		// asked for every category (nil).
		Describe("filtering by category", func() {
			It("hands the category id to the store", func() {
				store.many = []domain.Product{{ID: 1, Name: "Mouse", CategoryName: "Peripherals"}}

				rec := productsRequest(store, http.MethodGet, "/products?category_id=4", "")

				Expect(rec.Code).To(Equal(http.StatusOK))
				Expect(store.gotCategoryID).NotTo(BeNil())
				Expect(*store.gotCategoryID).To(Equal(int64(4)))
				Expect(rec.Body.String()).To(ContainSubstring(`"category_name":"Peripherals"`), "the category's name in the response")
			})

			It("asks for every category (nil) without a filter", func() {
				productsRequest(store, http.MethodGet, "/products", "")
				Expect(store.gotCategoryID).To(BeNil())
			})

			It("refuses a non-numeric category with 422", func() {
				rec := productsRequest(store, http.MethodGet, "/products?category_id=audio", "")
				Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
			})
		})

		Describe("pagination", func() {
			// With no query the first page of twelve is asked for, and the answer says
			// where the client is in the whole list.
			It("defaults to the first page", func() {
				store.many = []domain.Product{{ID: 1}}
				store.total = 30

				rec := productsRequest(store, http.MethodGet, "/products", "")

				Expect(store.gotLimit).To(Equal(int32(12)))
				Expect(store.gotOffset).To(Equal(int32(0)))
				var body struct {
					Pagination struct {
						Page       int   `json:"page"`
						PerPage    int   `json:"per_page"`
						Total      int64 `json:"total"`
						TotalPages int   `json:"total_pages"`
					} `json:"pagination"`
				}
				Expect(json.Unmarshal(rec.Body.Bytes(), &body)).To(Succeed())
				Expect(body.Pagination.Page).To(Equal(1))
				Expect(body.Pagination.PerPage).To(Equal(12))
				Expect(body.Pagination.Total).To(Equal(int64(30)))
				Expect(body.Pagination.TotalPages).To(Equal(3))
			})

			It("passes the page through", func() {
				productsRequest(store, http.MethodGet, "/products?page=3&per_page=5", "")
				Expect(store.gotLimit).To(Equal(int32(5)))
				Expect(store.gotOffset).To(Equal(int32(10)))
			})

			It("clamps per_page to 50", func() {
				productsRequest(store, http.MethodGet, "/products?per_page=5000", "")
				Expect(store.gotLimit).To(Equal(int32(50)))
			})

			DescribeTable("refuses a bad page with 422",
				func(query string) {
					rec := productsRequest(store, http.MethodGet, "/products?"+query, "")
					Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity))
				},
				Entry("page=0", "page=0"),
				Entry("page=-1", "page=-1"),
				Entry("page=abc", "page=abc"),
				Entry("per_page=0", "per_page=0"),
				Entry("per_page=x", "per_page=x"),
				Entry("page=99999999999", "page=99999999999"),
			)
		})
	})

	Describe("GET /products/:id", func() {
		It("answers 404 for an unknown product", func() {
			store.err = domain.ErrNotFound

			rec := productsRequest(store, http.MethodGet, "/products/nope", "")

			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})

		It("finds by id", func() {
			store.one = domain.Product{ID: 1, Name: "Widget"}

			rec := productsRequest(store, http.MethodGet, "/products/1", "")

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(store.gotID).To(Equal("1"))
			Expect(rec.Body.String()).To(ContainSubstring(`"name":"Widget"`))
		})
	})
})
