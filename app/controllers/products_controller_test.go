package controllers

import (
	"context"
	"encoding/json"
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
func productsRequest(t *testing.T, store *fakeProductsStore, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewProductsController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(&router.RouterGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

// Every action is mounted: a route that answers 404 was never registered.
func TestProductsRoutesAreMounted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	NewProductsController(&fakeProductsStore{}, log).SetupRoutes(&router.RouterGroup)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/products"},
		{http.MethodGet, "/products/1"},
	}
	for _, r := range routes {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(r.method, r.path, nil))
		if rec.Code == http.StatusNotFound {
			t.Errorf("%s %s is not mounted", r.method, r.path)
		}
	}
}

func TestProductsIndexLists(t *testing.T) {
	store := &fakeProductsStore{many: []domain.Product{{ID: 1}, {ID: 2}}}
	rec := productsRequest(t, store, http.MethodGet, "/products", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Products []struct{ ID int64 } `json:"products"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Products) != 2 || body.Products[1].ID != 2 {
		t.Errorf("body = %+v", body)
	}
}

func TestProductsShowUnknownIs404(t *testing.T) {
	rec := productsRequest(t, &fakeProductsStore{err: domain.ErrNotFound}, http.MethodGet, "/products/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestProductsShowFindsByID(t *testing.T) {
	store := &fakeProductsStore{one: domain.Product{ID: 1, Name: "Widget"}}
	rec := productsRequest(t, store, http.MethodGet, "/products/1", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.gotID != "1" {
		t.Errorf("store got id = %q, want %q", store.gotID, "1")
	}
	if !strings.Contains(rec.Body.String(), `"name":"Widget"`) {
		t.Errorf("body = %s", rec.Body)
	}
}

// An unexpected error is a 500 that says nothing about its cause.
func TestProductsUnexpectedErrorIs500WithoutDetail(t *testing.T) {
	rec := productsRequest(t, &fakeProductsStore{err: errors.New("connection refused to db-1")}, http.MethodGet, "/products", "")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db-1") {
		t.Errorf("body leaks the internal error: %s", rec.Body)
	}
}

// ?category_id=N is handed to the store as a number; without it the store is
// asked for every category (nil).
func TestProductsIndexFiltersByCategory(t *testing.T) {
	store := &fakeProductsStore{many: []domain.Product{{ID: 1, Name: "Mouse", CategoryName: "Peripherals"}}}
	rec := productsRequest(t, store, http.MethodGet, "/products?category_id=4", "")
	if rec.Code != http.StatusOK || store.gotCategoryID == nil || *store.gotCategoryID != 4 {
		t.Fatalf("status %d, category %v; want 200 and 4", rec.Code, store.gotCategoryID)
	}
	if !strings.Contains(rec.Body.String(), `"category_name":"Peripherals"`) {
		t.Errorf("body = %s, want the category's name in the response", rec.Body)
	}

	store = &fakeProductsStore{}
	productsRequest(t, store, http.MethodGet, "/products", "")
	if store.gotCategoryID != nil {
		t.Errorf("no filter asked for category %v, want nil (all)", *store.gotCategoryID)
	}
}

func TestProductsIndexRefusesANonNumericCategory(t *testing.T) {
	rec := productsRequest(t, &fakeProductsStore{}, http.MethodGet, "/products?category_id=audio", "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422: %s", rec.Code, rec.Body)
	}
}

// With no query the first page of twelve is asked for, and the answer says
// where the client is in the whole list.
func TestProductsIndexDefaultsToTheFirstPage(t *testing.T) {
	store := &fakeProductsStore{many: []domain.Product{{ID: 1}}, total: 30}
	rec := productsRequest(t, store, http.MethodGet, "/products", "")

	if store.gotLimit != 12 || store.gotOffset != 0 {
		t.Errorf("asked for limit %d offset %d, want 12 and 0", store.gotLimit, store.gotOffset)
	}
	var body struct {
		Pagination struct {
			Page       int   `json:"page"`
			PerPage    int   `json:"per_page"`
			Total      int64 `json:"total"`
			TotalPages int   `json:"total_pages"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if p := body.Pagination; p.Page != 1 || p.PerPage != 12 || p.Total != 30 || p.TotalPages != 3 {
		t.Errorf("pagination = %+v", p)
	}
}

func TestProductsIndexPassesThePageAndClampsPerPage(t *testing.T) {
	store := &fakeProductsStore{}
	productsRequest(t, store, http.MethodGet, "/products?page=3&per_page=5", "")
	if store.gotLimit != 5 || store.gotOffset != 10 {
		t.Errorf("page 3 of 5: limit %d offset %d, want 5 and 10", store.gotLimit, store.gotOffset)
	}
	productsRequest(t, store, http.MethodGet, "/products?per_page=5000", "")
	if store.gotLimit != 50 {
		t.Errorf("per_page=5000: limit %d, want it clamped to 50", store.gotLimit)
	}
}

func TestProductsIndexRefusesABadPage(t *testing.T) {
	for _, q := range []string{"page=0", "page=-1", "page=abc", "per_page=0", "per_page=x", "page=99999999999"} {
		rec := productsRequest(t, &fakeProductsStore{}, http.MethodGet, "/products?"+q, "")
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("?%s: status = %d, want 422", q, rec.Code)
		}
	}
}
