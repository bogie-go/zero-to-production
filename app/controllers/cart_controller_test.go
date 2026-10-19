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
	"time"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/middlewares"
	"ecommerce/app/services/token"
)

// testTokens mints and verifies bearer tokens for every controller test in
// this package that sits behind s.WithUser, the same Maker
// middlewares.RequireUser checks against in production.
var testTokens = token.Maker{Secret: []byte("a-test-secret-of-at-least-thirty-two-bytes"), Issuer: "ecommerce-test"}

// bearerFor mints a short-lived token for subject, for a test to present.
func bearerFor(t *testing.T, subject string) string {
	t.Helper()
	signed, err := testTokens.Mint(subject, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + signed
}

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

// cartRequest mounts the controller behind the same RequireUser middleware
// s.WithUser wraps it with in production, so these tests see exactly what a
// real request would: no token, no access. authorization, if not "", is
// sent as the request's Authorization header verbatim — use bearerFor for a
// valid one.
func cartRequest(t *testing.T, store *fakeCartStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	guarded := router.Group("", middlewares.RequireUser(testTokens))
	NewCartController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(guarded)

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
func TestCartRoutesAreMounted(t *testing.T) {
	rec := cartRequest(t, &fakeCartStore{}, http.MethodGet, "/cart", "", bearerFor(t, "visitor-1"))
	if rec.Code == http.StatusNotFound {
		t.Errorf("GET /cart is not mounted")
	}
}

// Index returns the current cart's contents, with each line's subtotal and
// the cart's total computed, not stored.
func TestCartIndexReturnsContentsWithComputedTotals(t *testing.T) {
	store := &fakeCartStore{
		cart: domain.Cart{ID: 1},
		lines: []domain.CartLine{
			{ID: 1, ProductID: 10, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2},
		},
	}
	rec := cartRequest(t, store, http.MethodGet, "/cart", "", bearerFor(t, "visitor-1"))

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

// The cart is resolved by the token's subject: whichever visitor the
// bearer token names is whose cart comes back.
func TestCartIndexResolvesByTokenSubject(t *testing.T) {
	store := &fakeCartStore{}
	cartRequest(t, store, http.MethodGet, "/cart", "", bearerFor(t, "visitor-42"))
	if store.gotToken != "visitor-42" {
		t.Errorf("store got token %q, want %q", store.gotToken, "visitor-42")
	}
}

// No bearer token at all is 401 — the detailed cases (expired, garbage,
// wrong scheme) are already covered by
// app/middlewares/require_user_test.go; this just proves /cart is actually
// mounted behind that guard, not bypassing it.
func TestCartRequiresABearerToken(t *testing.T) {
	rec := cartRequest(t, &fakeCartStore{}, http.MethodGet, "/cart", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

// An unexpected error is a 500 that says nothing about its cause.
func TestCartUnexpectedErrorIs500WithoutDetail(t *testing.T) {
	rec := cartRequest(t, &fakeCartStore{err: errors.New("connection refused to db-1")}, http.MethodGet, "/cart", "", bearerFor(t, "visitor-1"))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db-1") {
		t.Errorf("body leaks the internal error: %s", rec.Body)
	}
}
