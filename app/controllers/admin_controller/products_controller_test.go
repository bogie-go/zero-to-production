package admin_controller

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/services/token"
)

// testTokens mints and verifies bearer tokens for every test in this
// package, the same Maker a real RequireUser checks against in production.
var testTokens = token.Maker{Secret: []byte("a-test-secret-of-at-least-thirty-two-bytes"), Issuer: "ecommerce-test"}

func bearerFor(t *testing.T, subject string) string {
	t.Helper()
	signed, err := testTokens.Mint(subject, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
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
func adminProductsRequest(t *testing.T, products *fakeProductsStore, staff fakeStaffStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ns := NewServer(staff, testTokens, log)
	ns.Products = NewProductsController(products, log)
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

// Every action is mounted: a route that answers 404 was never registered.
func TestAdminProductsRoutesAreMounted(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodGet, "/admin/products"},
		{http.MethodGet, "/admin/products/1"},
		{http.MethodPost, "/admin/products"},
		{http.MethodPatch, "/admin/products/1"},
		{http.MethodDelete, "/admin/products/1"},
		{http.MethodPost, "/admin/products/1/archive"},
		{http.MethodPost, "/admin/products/1/restore"},
	}
	for _, r := range routes {
		rec := adminProductsRequest(t, &fakeProductsStore{}, aStaffUser, r.method, r.path, "", bearerFor(t, "1"))
		if rec.Code == http.StatusNotFound {
			t.Errorf("%s %s is not mounted", r.method, r.path)
		}
	}
}

// The whole namespace sits behind the staff gate: no token is 401, a
// customer's token is 403, before any controller action runs.
func TestAdminProductsRequiresStaff(t *testing.T) {
	store := &fakeProductsStore{list: []domain.Product{{ID: 1, Name: "Widget"}}}

	if rec := adminProductsRequest(t, store, aStaffUser, http.MethodGet, "/admin/products", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: %d, want 401", rec.Code)
	}
	customer := fakeStaffStore{user: domain.User{ID: 9, Role: domain.RoleCustomer}}
	if rec := adminProductsRequest(t, store, customer, http.MethodGet, "/admin/products", "", bearerFor(t, "9")); rec.Code != http.StatusForbidden {
		t.Errorf("customer token: %d, want 403", rec.Code)
	}
}

func TestAdminProductsIndexReturnsTheListing(t *testing.T) {
	store := &fakeProductsStore{list: []domain.Product{{ID: 1, Name: "Widget"}}}
	rec := adminProductsRequest(t, store, aStaffUser, http.MethodGet, "/admin/products", "", bearerFor(t, "1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"name":"Widget"`) {
		t.Errorf("body = %s, want Widget listed", rec.Body)
	}
}

func TestAdminProductsCreateReturns201(t *testing.T) {
	store := &fakeProductsStore{product: domain.Product{ID: 5, Name: "Widget", PriceCents: 500, Stock: 10}}
	body := `{"name":"Widget","description":"a widget","price_cents":500,"stock":10,"image_url":"/w.webp"}`
	rec := adminProductsRequest(t, store, aStaffUser, http.MethodPost, "/admin/products", body, bearerFor(t, "1"))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.gotProduct.Name != "Widget" || store.gotProduct.PriceCents != 500 {
		t.Errorf("CreateProduct got %+v", store.gotProduct)
	}
}

// A validation failure (the store's Validate, run on create) is a 422, not
// a 500 — the store reports domain.ErrInvalid, as a blank name would.
func TestAdminProductsCreateReports422OnValidationFailure(t *testing.T) {
	store := &fakeProductsStore{err: domain.ErrInvalid}
	rec := adminProductsRequest(t, store, aStaffUser, http.MethodPost, "/admin/products", `{"name":""}`, bearerFor(t, "1"))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422: %s", rec.Code, rec.Body)
	}
}

func TestAdminProductsUpdateMergesOntoTheExisting(t *testing.T) {
	store := &fakeProductsStore{
		product: domain.Product{ID: 5, Name: "Widget", PriceCents: 500, Stock: 10, ImageUrl: "/w.webp"},
	}
	body := `{"name":"Widget v2","description":"updated","price_cents":600,"stock":8,"image_url":"/w2.webp"}`
	rec := adminProductsRequest(t, store, aStaffUser, http.MethodPatch, "/admin/products/5", body, bearerFor(t, "1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.gotProduct.ID != 5 || store.gotProduct.Name != "Widget v2" || store.gotProduct.PriceCents != 600 {
		t.Errorf("UpdateProduct got %+v, want id 5 carried over with the new fields", store.gotProduct)
	}
}

func TestAdminProductsDestroyReturns204(t *testing.T) {
	store := &fakeProductsStore{}
	rec := adminProductsRequest(t, store, aStaffUser, http.MethodDelete, "/admin/products/5", "", bearerFor(t, "1"))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if !store.deleted || store.gotID != "5" {
		t.Errorf("DeleteProduct: deleted=%v gotID=%q, want true %q", store.deleted, store.gotID, "5")
	}
}

func TestAdminProductsArchiveAndRestore(t *testing.T) {
	when := time.Now()
	store := &fakeProductsStore{product: domain.Product{ID: 5, Name: "Widget", ArchivedAt: &when}}
	rec := adminProductsRequest(t, store, aStaffUser, http.MethodPost, "/admin/products/5/archive", "", bearerFor(t, "1"))

	if rec.Code != http.StatusOK || !store.archived || store.gotID != "5" {
		t.Fatalf("archive: status %d archived=%v id=%q: %s", rec.Code, store.archived, store.gotID, rec.Body)
	}
	if strings.Contains(rec.Body.String(), `"archived_at":null`) {
		t.Errorf("body = %s, want an archived_at time", rec.Body)
	}

	store = &fakeProductsStore{product: domain.Product{ID: 5, Name: "Widget"}}
	rec = adminProductsRequest(t, store, aStaffUser, http.MethodPost, "/admin/products/5/restore", "", bearerFor(t, "1"))
	if rec.Code != http.StatusOK || !store.restored {
		t.Fatalf("restore: status %d restored=%v: %s", rec.Code, store.restored, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"archived_at":null`) {
		t.Errorf("body = %s, want archived_at null", rec.Body)
	}
}

// Archiving is staff-only like every other admin action.
func TestAdminProductsArchiveRequiresStaff(t *testing.T) {
	customer := fakeStaffStore{user: domain.User{ID: 9, Role: domain.RoleCustomer}}
	store := &fakeProductsStore{}
	rec := adminProductsRequest(t, store, customer, http.MethodPost, "/admin/products/5/archive", "", bearerFor(t, "9"))
	if rec.Code != http.StatusForbidden || store.archived {
		t.Errorf("customer archive: %d archived=%v, want 403 and nothing archived", rec.Code, store.archived)
	}
}

// The admin list includes archived products unless told not to, and pages.
func TestAdminProductsIndexPagesAndFiltersArchived(t *testing.T) {
	store := &fakeProductsStore{total: 45}
	rec := adminProductsRequest(t, store, aStaffUser, http.MethodGet, "/admin/products", "", bearerFor(t, "1"))
	if !store.gotArchived || store.gotLimit != 20 || store.gotOffset != 0 {
		t.Errorf("default: archived %v limit %d offset %d, want true 20 0", store.gotArchived, store.gotLimit, store.gotOffset)
	}
	if !strings.Contains(rec.Body.String(), `"total_pages":3`) {
		t.Errorf("body = %s, want 3 total pages of 20 for 45", rec.Body)
	}

	adminProductsRequest(t, store, aStaffUser, http.MethodGet, "/admin/products?include_archived=false&page=2&per_page=10", "", bearerFor(t, "1"))
	if store.gotArchived || store.gotLimit != 10 || store.gotOffset != 10 {
		t.Errorf("page 2: archived %v limit %d offset %d, want false 10 10", store.gotArchived, store.gotLimit, store.gotOffset)
	}

	rec = adminProductsRequest(t, store, aStaffUser, http.MethodGet, "/admin/products?page=0", "", bearerFor(t, "1"))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("page=0: status = %d, want 422", rec.Code)
	}
}
