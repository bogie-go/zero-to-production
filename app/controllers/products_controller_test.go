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

	gotID string
}

func (f *fakeProductsStore) GetActiveProduct(_ context.Context, id string) (domain.Product, error) {
	f.gotID = id
	return f.one, f.err
}

func (f *fakeProductsStore) ListActiveProducts(context.Context, int32) ([]domain.Product, error) {
	return f.many, f.err
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
