package admin_controller

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

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
func adminCategoriesRequest(t *testing.T, categories *fakeCategoriesStore, staff fakeStaffStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ns := NewServer(staff, testTokens, log)
	ns.Products = NewProductsController(&fakeProductsStore{}, log)
	ns.Categories = NewCategoriesController(categories, log)
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

func TestAdminCategoriesRequiresStaff(t *testing.T) {
	store := &fakeCategoriesStore{}
	if rec := adminCategoriesRequest(t, store, aStaffUser, http.MethodGet, "/admin/categories", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: %d, want 401", rec.Code)
	}
	customer := fakeStaffStore{user: domain.User{ID: 9, Role: domain.RoleCustomer}}
	if rec := adminCategoriesRequest(t, store, customer, http.MethodPost, "/admin/categories", `{"name":"x"}`, bearerFor(t, "9")); rec.Code != http.StatusForbidden || store.gotCategory.Name != "" {
		t.Errorf("customer: %d (created %q), want 403 and nothing created", rec.Code, store.gotCategory.Name)
	}
}

func TestAdminCategoriesCRUD(t *testing.T) {
	store := &fakeCategoriesStore{category: domain.Category{ID: 4, Name: "Audio"}, list: []domain.Category{{ID: 4, Name: "Audio", ProductCount: 2}}}
	staff := bearerFor(t, "1")

	if rec := adminCategoriesRequest(t, store, aStaffUser, http.MethodGet, "/admin/categories", "", staff); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"product_count":2`) {
		t.Errorf("index: %d %s", rec.Code, rec.Body)
	}
	if rec := adminCategoriesRequest(t, store, aStaffUser, http.MethodPost, "/admin/categories", `{"name":"Audio"}`, staff); rec.Code != http.StatusCreated || store.gotCategory.Name != "Audio" {
		t.Errorf("create: %d, got %+v", rec.Code, store.gotCategory)
	}
	if rec := adminCategoriesRequest(t, store, aStaffUser, http.MethodPatch, "/admin/categories/4", `{"name":"Sound"}`, staff); rec.Code != http.StatusOK || store.gotCategory.ID != 4 || store.gotCategory.Name != "Sound" {
		t.Errorf("update: %d, got %+v; want id 4 carried over with the new name", rec.Code, store.gotCategory)
	}
	if rec := adminCategoriesRequest(t, store, aStaffUser, http.MethodDelete, "/admin/categories/4", "", staff); rec.Code != http.StatusNoContent || !store.deleted || store.gotID != "4" {
		t.Errorf("destroy: %d deleted=%v id=%q", rec.Code, store.deleted, store.gotID)
	}
}

func TestAdminCategoriesMapsErrors(t *testing.T) {
	staff := bearerFor(t, "1")
	cases := map[string]struct {
		err  error
		want int
	}{
		"invalid (blank or duplicate name)": {domain.ErrInvalid, http.StatusUnprocessableEntity},
		"not found":                         {domain.ErrNotFound, http.StatusNotFound},
		"anything else":                     {errors.New("db down"), http.StatusInternalServerError},
	}
	for name, tc := range cases {
		rec := adminCategoriesRequest(t, &fakeCategoriesStore{err: tc.err}, aStaffUser, http.MethodPost, "/admin/categories", `{"name":"x"}`, staff)
		if rec.Code != tc.want || strings.Contains(rec.Body.String(), "db down") {
			t.Errorf("%s: %d %s, want %d without leaking", name, rec.Code, rec.Body, tc.want)
		}
	}
}
