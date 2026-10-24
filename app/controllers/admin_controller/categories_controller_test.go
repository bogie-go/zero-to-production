package admin_controller_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers/admin_controller"
	"ecommerce/app/domain"
)

type fakeCategoriesStore struct {
	category domain.Category
	list     []domain.Category
	err      error

	gotCategory domain.Category
	gotID       string
	deleted     bool
}

func (f *fakeCategoriesStore) ListCategoriesWithCounts(context.Context) ([]domain.Category, error) {
	return f.list, f.err
}
func (f *fakeCategoriesStore) GetCategory(_ context.Context, id string) (domain.Category, error) {
	f.gotID = id
	return f.category, f.err
}
func (f *fakeCategoriesStore) CreateCategory(_ context.Context, c domain.Category) (domain.Category, error) {
	f.gotCategory = c
	return f.category, f.err
}
func (f *fakeCategoriesStore) UpdateCategory(_ context.Context, c domain.Category) (domain.Category, error) {
	f.gotCategory = c
	return f.category, f.err
}
func (f *fakeCategoriesStore) DeleteCategory(_ context.Context, id string) error {
	f.gotID, f.deleted = id, true
	return f.err
}

// adminCategoriesRequest mounts the whole namespace, staff gate included.
func adminCategoriesRequest(categories *fakeCategoriesStore, staff fakeStaffStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ns := admin_controller.NewServer(staff, testTokens, log)
	ns.Products = admin_controller.NewProductsController(&fakeProductsStore{}, log)
	ns.Categories = admin_controller.NewCategoriesController(categories, log)
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

var _ = Describe("admin categories", func() {
	var store *fakeCategoriesStore
	var staff string

	BeforeEach(func() {
		store = &fakeCategoriesStore{}
		staff = bearerFor("1")
	})

	Describe("staff gate", func() {
		It("answers 401 with no token", func() {
			rec := adminCategoriesRequest(store, aStaffUser, http.MethodGet, "/admin/categories", "", "")
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})

		It("answers 403 for a customer and creates nothing", func() {
			customer := fakeStaffStore{user: domain.User{ID: 9, Role: domain.RoleCustomer}}
			rec := adminCategoriesRequest(store, customer, http.MethodPost, "/admin/categories", `{"name":"x"}`, bearerFor("9"))
			Expect(rec.Code).To(Equal(http.StatusForbidden))
			Expect(store.gotCategory.Name).To(BeEmpty())
		})
	})

	Context("CRUD", func() {
		BeforeEach(func() {
			store.category = domain.Category{ID: 4, Name: "Audio"}
			store.list = []domain.Category{{ID: 4, Name: "Audio", ProductCount: 2}}
		})

		It("GET /admin/categories lists with product counts", func() {
			rec := adminCategoriesRequest(store, aStaffUser, http.MethodGet, "/admin/categories", "", staff)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(ContainSubstring(`"product_count":2`))
		})

		It("POST /admin/categories creates", func() {
			rec := adminCategoriesRequest(store, aStaffUser, http.MethodPost, "/admin/categories", `{"name":"Audio"}`, staff)
			Expect(rec.Code).To(Equal(http.StatusCreated))
			Expect(store.gotCategory.Name).To(Equal("Audio"))
		})

		It("PATCH /admin/categories/:id carries the id over with the new name", func() {
			rec := adminCategoriesRequest(store, aStaffUser, http.MethodPatch, "/admin/categories/4", `{"name":"Sound"}`, staff)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(store.gotCategory.ID).To(BeEquivalentTo(4))
			Expect(store.gotCategory.Name).To(Equal("Sound"))
		})

		It("DELETE /admin/categories/:id destroys", func() {
			rec := adminCategoriesRequest(store, aStaffUser, http.MethodDelete, "/admin/categories/4", "", staff)
			Expect(rec.Code).To(Equal(http.StatusNoContent))
			Expect(store.deleted).To(BeTrue())
			Expect(store.gotID).To(Equal("4"))
		})
	})

	DescribeTable("maps store errors without leaking the cause",
		func(err error, want int) {
			rec := adminCategoriesRequest(&fakeCategoriesStore{err: err}, aStaffUser, http.MethodPost, "/admin/categories", `{"name":"x"}`, staff)
			Expect(rec.Code).To(Equal(want))
			Expect(rec.Body.String()).NotTo(ContainSubstring("db down"))
		},
		Entry("invalid (blank or duplicate name)", domain.ErrInvalid, http.StatusUnprocessableEntity),
		Entry("not found", domain.ErrNotFound, http.StatusNotFound),
		Entry("anything else", errors.New("db down"), http.StatusInternalServerError),
	)
})
