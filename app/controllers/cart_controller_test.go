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
)

// fakeCartStore is a fake of the store interface declared beside the
// controller: it records what the controller asked for and answers with
// what the test chose.
type fakeCartStore struct {
	cart  domain.Cart
	lines []domain.CartLine
	err   error

	gotToken string
}

func (f *fakeCartStore) FindOrCreateCartByToken(_ context.Context, token string) (domain.Cart, error) {
	f.gotToken = token
	return f.cart, f.err
}

func (f *fakeCartStore) CartContents(_ context.Context, cartID int64) ([]domain.CartLine, error) {
	return f.lines, f.err
}

// cartRequest mounts the controller alone on a bare router, so these tests
// see the controller and nothing else: no middleware, no other routes.
// token, if not "", is sent as the request's X-Cart-Token header.
func cartRequest(t *testing.T, store *fakeCartStore, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewCartController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(&router.RouterGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set(cartTokenHeader, token)
	}
	router.ServeHTTP(rec, req)
	return rec
}

// Every action is mounted: a route that answers 404 was never registered.
func TestCartRoutesAreMounted(t *testing.T) {
	rec := cartRequest(t, &fakeCartStore{}, http.MethodGet, "/cart", "", "")
	if rec.Code == http.StatusNotFound {
		t.Errorf("GET /cart is not mounted")
	}
}

// Index creates the cart on first visit and returns its contents, with
// each line's subtotal and the cart's total computed, not stored.
func TestCartIndexReturnsContentsWithComputedTotals(t *testing.T) {
	store := &fakeCartStore{
		cart: domain.Cart{ID: 1},
		lines: []domain.CartLine{
			{ID: 1, ProductID: 10, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2},
		},
	}
	rec := cartRequest(t, store, http.MethodGet, "/cart", "", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"subtotal_cents":1000`) {
		t.Errorf("body = %s, want a line subtotal of 1000", rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"total_cents":1000`) {
		t.Errorf("body = %s, want a cart total of 1000", rec.Body)
	}
}

// A first-time visitor, sending no token, gets one minted for them, used to
// resolve the cart and echoed back so they can send it next time.
func TestCartIndexMintsATokenOnFirstVisit(t *testing.T) {
	store := &fakeCartStore{}
	rec := cartRequest(t, store, http.MethodGet, "/cart", "", "")

	got := rec.Header().Get(cartTokenHeader)
	if got == "" {
		t.Fatalf("response did not carry an %s header", cartTokenHeader)
	}
	if store.gotToken != got {
		t.Errorf("store got token %q, want the minted %q", store.gotToken, got)
	}
}

// A returning visitor's token is reused as is, not replaced.
func TestCartIndexReusesAPresentedToken(t *testing.T) {
	store := &fakeCartStore{}
	rec := cartRequest(t, store, http.MethodGet, "/cart", "", "visitor-123")

	if rec.Header().Get(cartTokenHeader) != "visitor-123" {
		t.Errorf("response token = %q, want it echoed back unchanged", rec.Header().Get(cartTokenHeader))
	}
	if store.gotToken != "visitor-123" {
		t.Errorf("store got token %q, want %q", store.gotToken, "visitor-123")
	}
}

// An unexpected error is a 500 that says nothing about its cause.
func TestCartUnexpectedErrorIs500WithoutDetail(t *testing.T) {
	rec := cartRequest(t, &fakeCartStore{err: errors.New("connection refused to db-1")}, http.MethodGet, "/cart", "", "")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db-1") {
		t.Errorf("body leaks the internal error: %s", rec.Body)
	}
}
