package controllers

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
	"ecommerce/app/middlewares"
)

// fakeCartItemsStore is a fake of the store interface declared beside the
// controller: it records what the controller asked for and answers with
// what the test chose.
type fakeCartItemsStore struct {
	cart  domain.Cart
	item  domain.CartItem
	lines []domain.CartLine
	rules []domain.PriceRule
	err   error

	gotProductID int64
	gotQuantity  int32
	gotID        string
}

func (f *fakeCartItemsStore) FindOrCreateCartByToken(_ context.Context, token string) (domain.Cart, error) {
	return f.cart, f.err
}

func (f *fakeCartItemsStore) AddCartItem(_ context.Context, cartID, productID int64, quantity int32) (domain.CartItem, error) {
	f.gotProductID, f.gotQuantity = productID, quantity
	return f.item, f.err
}

func (f *fakeCartItemsStore) UpdateCartItemQuantity(_ context.Context, id string, quantity int32) (domain.CartItem, error) {
	f.gotID, f.gotQuantity = id, quantity
	return f.item, f.err
}

func (f *fakeCartItemsStore) DeleteCartItem(_ context.Context, id string) error {
	f.gotID = id
	return f.err
}

func (f *fakeCartItemsStore) CartContents(_ context.Context, cartID int64) ([]domain.CartLine, error) {
	return f.lines, nil
}

func (f *fakeCartItemsStore) ListActivePriceRules(_ context.Context) ([]domain.PriceRule, error) {
	return f.rules, nil
}

// cartItemsRequest mounts the controller behind the same RequireUser
// middleware s.WithUser wraps it with in production, so these tests see
// exactly what a real request would: no token, no access. authorization,
// if not "", is sent as the request's Authorization header verbatim — use
// bearerFor for a valid one.
func cartItemsRequest(t *testing.T, store *fakeCartItemsStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	guarded := router.Group("", middlewares.RequireUser(testTokens))
	NewCartItemsController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(guarded)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

// Every action is mounted: a route that answers 404 was never registered.
func TestCartItemsRoutesAreMounted(t *testing.T) {
	store := &fakeCartItemsStore{}
	auth := bearerFor(t, "visitor-1")
	routes := []struct{ method, path, body string }{
		{http.MethodPost, "/cart_items", "{}"},
		{http.MethodPatch, "/cart_items/1", "{}"},
		{http.MethodDelete, "/cart_items/1", ""},
	}
	for _, r := range routes {
		rec := cartItemsRequest(t, store, r.method, r.path, r.body, auth)
		if rec.Code == http.StatusNotFound {
			t.Errorf("%s %s is not mounted", r.method, r.path)
		}
	}
}

// No bearer token at all is 401 — the detailed cases are already covered
// by app/middlewares/require_user_test.go; this just proves cart_items is
// actually mounted behind that guard.
func TestCartItemsRequiresABearerToken(t *testing.T) {
	rec := cartItemsRequest(t, &fakeCartItemsStore{}, http.MethodPost, "/cart_items", "{}", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

// Create passes the body through to AddCartItem and answers with the
// cart's contents, not just the one line.
func TestCartItemsCreatePassesTheBodyThroughAndReturnsTheCart(t *testing.T) {
	store := &fakeCartItemsStore{
		cart:  domain.Cart{ID: 1},
		lines: []domain.CartLine{{ID: 5, ProductID: 10, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2}},
	}
	rec := cartItemsRequest(t, store, http.MethodPost, "/cart_items", `{"product_id":10,"quantity":2}`, bearerFor(t, "visitor-1"))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.gotProductID != 10 || store.gotQuantity != 2 {
		t.Errorf("store got product_id=%d quantity=%d", store.gotProductID, store.gotQuantity)
	}
	if !strings.Contains(rec.Body.String(), `"total_cents":1000`) {
		t.Errorf("body = %s", rec.Body)
	}
}

// Validation (quantity <= 0, more than stock, no such product) is the
// store's; the controller's only job is to say 422.
func TestCartItemsCreateInvalidIs422(t *testing.T) {
	store := &fakeCartItemsStore{err: errors.Join(domain.ErrInvalid, errors.New("only 1 in stock"))}
	rec := cartItemsRequest(t, store, http.MethodPost, "/cart_items", `{"product_id":10,"quantity":99}`, bearerFor(t, "visitor-1"))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422: %s", rec.Code, rec.Body)
	}
}

func TestCartItemsCreateMalformedJSONIs400(t *testing.T) {
	rec := cartItemsRequest(t, &fakeCartItemsStore{}, http.MethodPost, "/cart_items", `{"nope":`, bearerFor(t, "visitor-1"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestCartItemsUpdateChangesQuantityAndReturnsTheCart(t *testing.T) {
	store := &fakeCartItemsStore{item: domain.CartItem{ID: 5, CartID: 1}}
	rec := cartItemsRequest(t, store, http.MethodPatch, "/cart_items/5", `{"quantity":3}`, bearerFor(t, "visitor-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.gotID != "5" || store.gotQuantity != 3 {
		t.Errorf("store got id=%q quantity=%d", store.gotID, store.gotQuantity)
	}
}

func TestCartItemsUpdateUnknownIs404(t *testing.T) {
	rec := cartItemsRequest(t, &fakeCartItemsStore{err: domain.ErrNotFound}, http.MethodPatch, "/cart_items/nope", `{"quantity":1}`, bearerFor(t, "visitor-1"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestCartItemsDestroyRemovesTheLineAndReturnsTheCart(t *testing.T) {
	store := &fakeCartItemsStore{cart: domain.Cart{ID: 1}}
	rec := cartItemsRequest(t, store, http.MethodDelete, "/cart_items/5", "", bearerFor(t, "visitor-1"))

	if rec.Code != http.StatusOK || store.gotID != "5" {
		t.Errorf("status = %d, id = %q", rec.Code, store.gotID)
	}
}

// An unexpected error is a 500 that says nothing about its cause.
func TestCartItemsUnexpectedErrorIs500WithoutDetail(t *testing.T) {
	store := &fakeCartItemsStore{err: errors.New("connection refused to db-1")}
	rec := cartItemsRequest(t, store, http.MethodPost, "/cart_items", `{"product_id":1,"quantity":1}`, bearerFor(t, "visitor-1"))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db-1") {
		t.Errorf("body leaks the internal error: %s", rec.Body)
	}
}
