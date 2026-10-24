package admin_controller_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers/admin_controller"
	"ecommerce/app/domain"
	"ecommerce/app/services/token"
)

// testTokens mints and verifies bearer tokens for every test in this
// package, the same Maker a real RequireUser checks against in production.
var testTokens = token.Maker{Secret: []byte("a-test-secret-of-at-least-thirty-two-bytes"), Issuer: "ecommerce-test"}

func bearerFor(subject string) string {
	signed, err := testTokens.Mint(subject, time.Hour)
	Expect(err).NotTo(HaveOccurred())
	return "Bearer " + signed
}

// fakeStaffStore answers the namespace's own WithStaff check.
type fakeStaffStore struct {
	user domain.User
	err  error
}

func (f fakeStaffStore) GetUser(_ context.Context, id string) (domain.User, error) {
	return f.user, f.err
}

// fakeProductsStore is a fake of the store interface declared beside the
// controller: it records what the controller asked for and answers with
// what the test chose.
type fakeProductsStore struct {
	product domain.Product
	list    []domain.Product
	err     error

	total int64

	gotArchived bool
	gotLimit    int32
	gotOffset   int32

	gotID      string
	gotProduct domain.Product
	deleted    bool
	archived   bool
	restored   bool
}

func (f *fakeProductsStore) GetProduct(_ context.Context, id string) (domain.Product, error) {
	f.gotID = id
	return f.product, f.err
}

func (f *fakeProductsStore) ListProductsPage(_ context.Context, includeArchived bool, limit, offset int32) ([]domain.Product, error) {
	f.gotArchived, f.gotLimit, f.gotOffset = includeArchived, limit, offset
	return f.list, f.err
}

func (f *fakeProductsStore) CountProducts(_ context.Context, _ bool) (int64, error) {
	return f.total, f.err
}

func (f *fakeProductsStore) CreateProduct(_ context.Context, p domain.Product) (domain.Product, error) {
	f.gotProduct = p
	return f.product, f.err
}

func (f *fakeProductsStore) UpdateProduct(_ context.Context, p domain.Product) (domain.Product, error) {
	f.gotProduct = p
	return f.product, f.err
}

func (f *fakeProductsStore) ArchiveProduct(_ context.Context, id string) (domain.Product, error) {
	f.gotID = id
	f.archived = true
	return f.product, f.err
}

func (f *fakeProductsStore) RestoreProduct(_ context.Context, id string) (domain.Product, error) {
	f.gotID = id
	f.restored = true
	return f.product, f.err
}

func (f *fakeProductsStore) DeleteProduct(_ context.Context, id string) error {
	f.gotID = id
	f.deleted = true
	return f.err
}

// adminProductsRequest mounts the whole namespace, staff gate included, so
// these tests see exactly what a real request would.
func adminProductsRequest(products *fakeProductsStore, staff fakeStaffStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ns := admin_controller.NewServer(staff, testTokens, log)
	ns.Products = admin_controller.NewProductsController(products, log)
	ns.SetupRoutes(&router.RouterGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

var aStaffUser = fakeStaffStore{user: domain.User{ID: 1, Role: domain.RoleStaff}}

var _ = Describe("admin products", func() {
	var store *fakeProductsStore

	BeforeEach(func() {
		store = &fakeProductsStore{}
	})

	// Every action is mounted: a route that answers 404 was never registered.
	DescribeTable("routes are mounted",
		func(method, path string) {
			rec := adminProductsRequest(store, aStaffUser, method, path, "", bearerFor("1"))
			Expect(rec.Code).NotTo(Equal(http.StatusNotFound), "%s %s is not mounted", method, path)
		},
		Entry(nil, http.MethodGet, "/admin/products"),
		Entry(nil, http.MethodGet, "/admin/products/1"),
		Entry(nil, http.MethodPost, "/admin/products"),
		Entry(nil, http.MethodPatch, "/admin/products/1"),
		Entry(nil, http.MethodDelete, "/admin/products/1"),
		Entry(nil, http.MethodPost, "/admin/products/1/archive"),
		Entry(nil, http.MethodPost, "/admin/products/1/restore"),
	)

	// The whole namespace sits behind the staff gate: no token is 401, a
	// customer's token is 403, before any controller action runs.
	Describe("staff gate", func() {
		BeforeEach(func() {
			store.list = []domain.Product{{ID: 1, Name: "Widget"}}
		})

		It("answers 401 with no token", func() {
			rec := adminProductsRequest(store, aStaffUser, http.MethodGet, "/admin/products", "", "")
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})

		It("answers 403 for a customer token", func() {
			customer := fakeStaffStore{user: domain.User{ID: 9, Role: domain.RoleCustomer}}
			rec := adminProductsRequest(store, customer, http.MethodGet, "/admin/products", "", bearerFor("9"))
			Expect(rec.Code).To(Equal(http.StatusForbidden))
		})

		// Archiving is staff-only like every other admin action.
		It("refuses a customer archiving a product", func() {
			customer := fakeStaffStore{user: domain.User{ID: 9, Role: domain.RoleCustomer}}
			rec := adminProductsRequest(store, customer, http.MethodPost, "/admin/products/5/archive", "", bearerFor("9"))
			Expect(rec.Code).To(Equal(http.StatusForbidden))
			Expect(store.archived).To(BeFalse())
		})
	})

	Describe("GET /admin/products", func() {
		It("returns the listing", func() {
			store.list = []domain.Product{{ID: 1, Name: "Widget"}}
			rec := adminProductsRequest(store, aStaffUser, http.MethodGet, "/admin/products", "", bearerFor("1"))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(rec.Body.String()).To(ContainSubstring(`"name":"Widget"`))
		})

		// The admin list includes archived products unless told not to, and pages.
		Context("paging and the archived filter", func() {
			BeforeEach(func() {
				store.total = 45
			})

			It("includes archived and pages by 20 by default", func() {
				rec := adminProductsRequest(store, aStaffUser, http.MethodGet, "/admin/products", "", bearerFor("1"))
				Expect(store.gotArchived).To(BeTrue())
				Expect(store.gotLimit).To(Equal(int32(20)))
				Expect(store.gotOffset).To(Equal(int32(0)))
				Expect(rec.Body.String()).To(ContainSubstring(`"total_pages":3`))
			})

			It("honours include_archived, page and per_page", func() {
				adminProductsRequest(store, aStaffUser, http.MethodGet, "/admin/products?include_archived=false&page=2&per_page=10", "", bearerFor("1"))
				Expect(store.gotArchived).To(BeFalse())
				Expect(store.gotLimit).To(Equal(int32(10)))
				Expect(store.gotOffset).To(Equal(int32(10)))
			})

			It("answers 422 for page=0", func() {
				rec := adminProductsRequest(store, aStaffUser, http.MethodGet, "/admin/products?page=0", "", bearerFor("1"))
				Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity))
			})
		})
	})

	Describe("POST /admin/products", func() {
		It("returns 201 and passes the fields to the store", func() {
			store.product = domain.Product{ID: 5, Name: "Widget", PriceCents: 500, Stock: 10}
			body := `{"name":"Widget","description":"a widget","price_cents":500,"stock":10,"image_url":"/w.webp"}`
			rec := adminProductsRequest(store, aStaffUser, http.MethodPost, "/admin/products", body, bearerFor("1"))

			Expect(rec.Code).To(Equal(http.StatusCreated), rec.Body.String())
			Expect(store.gotProduct.Name).To(Equal("Widget"))
			Expect(store.gotProduct.PriceCents).To(BeEquivalentTo(500))
		})

		// A validation failure (the store's Validate, run on create) is a 422,
		// not a 500: the store reports domain.ErrInvalid, as a blank name would.
		It("reports 422 on a validation failure", func() {
			store.err = domain.ErrInvalid
			rec := adminProductsRequest(store, aStaffUser, http.MethodPost, "/admin/products", `{"name":""}`, bearerFor("1"))

			Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
		})
	})

	Describe("PATCH /admin/products/:id", func() {
		It("merges onto the existing product, carrying the id over", func() {
			store.product = domain.Product{ID: 5, Name: "Widget", PriceCents: 500, Stock: 10, ImageUrl: "/w.webp"}
			body := `{"name":"Widget v2","description":"updated","price_cents":600,"stock":8,"image_url":"/w2.webp"}`
			rec := adminProductsRequest(store, aStaffUser, http.MethodPatch, "/admin/products/5", body, bearerFor("1"))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(store.gotProduct.ID).To(BeEquivalentTo(5))
			Expect(store.gotProduct.Name).To(Equal("Widget v2"))
			Expect(store.gotProduct.PriceCents).To(BeEquivalentTo(600))
		})
	})

	Describe("DELETE /admin/products/:id", func() {
		It("returns 204 and deletes", func() {
			rec := adminProductsRequest(store, aStaffUser, http.MethodDelete, "/admin/products/5", "", bearerFor("1"))

			Expect(rec.Code).To(Equal(http.StatusNoContent), rec.Body.String())
			Expect(store.deleted).To(BeTrue())
			Expect(store.gotID).To(Equal("5"))
		})
	})

	Describe("POST /admin/products/:id/archive", func() {
		It("archives and returns an archived_at time", func() {
			when := time.Now()
			store.product = domain.Product{ID: 5, Name: "Widget", ArchivedAt: &when}
			rec := adminProductsRequest(store, aStaffUser, http.MethodPost, "/admin/products/5/archive", "", bearerFor("1"))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(store.archived).To(BeTrue())
			Expect(store.gotID).To(Equal("5"))
			Expect(rec.Body.String()).NotTo(ContainSubstring(`"archived_at":null`))
		})
	})

	Describe("POST /admin/products/:id/restore", func() {
		It("restores and returns archived_at null", func() {
			store.product = domain.Product{ID: 5, Name: "Widget"}
			rec := adminProductsRequest(store, aStaffUser, http.MethodPost, "/admin/products/5/restore", "", bearerFor("1"))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(store.restored).To(BeTrue())
			Expect(rec.Body.String()).To(ContainSubstring(`"archived_at":null`))
		})
	})
})
