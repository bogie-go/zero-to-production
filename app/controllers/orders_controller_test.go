package controllers

import (
	"context"
	"fmt"
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

// fakeOrdersStore is a fake of the store interface declared beside the
// controller: it records what the controller asked for and answers with
// what the test chose.
type fakeOrdersStore struct {
	cart    domain.Cart
	cartErr error
	order   domain.Order
	lines   []domain.OrderLine
	err     error
	orders  []domain.Order

	gotCartToken  string
	gotUserID     int64
	gotCartID     int64
	gotOrderID    string
	checkoutCalls int
}

func (f *fakeOrdersStore) FindOrCreateCartByToken(_ context.Context, token string) (domain.Cart, error) {
	f.gotCartToken = token
	return f.cart, f.cartErr
}

func (f *fakeOrdersStore) CreateOrderFromCart(_ context.Context, userID, cartID int64) (domain.Order, []domain.OrderLine, error) {
	f.checkoutCalls++
	f.gotUserID = userID
	f.gotCartID = cartID
	return f.order, f.lines, f.err
}

func (f *fakeOrdersStore) ListOrdersForUser(_ context.Context, userID int64, limit int32) ([]domain.Order, error) {
	f.gotUserID = userID
	return f.orders, f.err
}

func (f *fakeOrdersStore) GetOrderForUser(_ context.Context, userID int64, id string) (domain.Order, []domain.OrderLine, error) {
	f.gotUserID = userID
	f.gotOrderID = id
	return f.order, f.lines, f.err
}

// ordersRequest mounts the controller behind the same RequireUser
// middleware s.WithUser wraps it with in production, so these tests see
// exactly what a real request would: no token, no access.
func ordersRequest(t *testing.T, store *fakeOrdersStore, method, path, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	guarded := router.Group("", middlewares.RequireUser(testTokens))
	NewOrdersController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(guarded)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

// Every action is mounted: a route that answers 404 was never registered.
func TestOrdersRoutesAreMounted(t *testing.T) {
	rec := ordersRequest(t, &fakeOrdersStore{}, http.MethodGet, "/orders", bearerFor(t, "1"))
	if rec.Code == http.StatusNotFound {
		t.Errorf("GET /orders is not mounted")
	}
}

// A guest's token is a valid bearer token — RequireUser lets it through —
// but its subject is a uuid, not a user id, so checkout answers 403
// without ever reaching the store: there is no account to put on the order.
func TestOrdersCreateRejectsAGuestToken(t *testing.T) {
	store := &fakeOrdersStore{}
	rec := ordersRequest(t, store, http.MethodPost, "/orders", bearerFor(t, "a-guest-uuid-not-an-id"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", rec.Code, rec.Body)
	}
	if store.checkoutCalls != 0 {
		t.Errorf("CreateOrderFromCart was called %d times, want 0", store.checkoutCalls)
	}
}

// A real user's checkout resolves their own cart by subject, then checks
// out that cart under their own user id — never an id read from the
// request.
func TestOrdersCreateChecksOutTheCallersOwnCart(t *testing.T) {
	store := &fakeOrdersStore{
		cart:  domain.Cart{ID: 42},
		order: domain.Order{ID: 7, Status: "pending", SubtotalCents: 1000, TotalCents: 1000},
		lines: []domain.OrderLine{{ProductID: 10, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2}},
	}
	rec := ordersRequest(t, store, http.MethodPost, "/orders", bearerFor(t, "9"))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body)
	}
	if store.gotCartToken != "9" {
		t.Errorf("cart resolved for token %q, want the caller's own subject %q", store.gotCartToken, "9")
	}
	if store.gotUserID != 9 {
		t.Errorf("CreateOrderFromCart userID = %d, want 9", store.gotUserID)
	}
	if store.gotCartID != 42 {
		t.Errorf("CreateOrderFromCart cartID = %d, want 42 (the caller's own cart)", store.gotCartID)
	}
	if !strings.Contains(rec.Body.String(), `"id":7`) {
		t.Errorf("body = %s, want the created order's id", rec.Body)
	}
}

// A line out of stock by checkout time is a 422, same as any other
// validation failure, not a 500.
func TestOrdersCreateReportsAnOutOfStockLineAs422(t *testing.T) {
	store := &fakeOrdersStore{
		cart: domain.Cart{ID: 1},
		err:  fmt.Errorf("%w: Widget no longer has enough stock", domain.ErrInvalid),
	}
	rec := ordersRequest(t, store, http.MethodPost, "/orders", bearerFor(t, "9"))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422: %s", rec.Code, rec.Body)
	}
}

// GET /orders/:id is scoped to the caller's own user id; a request for
// someone else's order is the store's job to 404, proven in
// app/models/orders_extra_test.go — this just confirms the controller
// passes the caller's own id through, never one from the URL or body.
func TestOrdersShowUsesTheCallersOwnUserID(t *testing.T) {
	store := &fakeOrdersStore{order: domain.Order{ID: 5}}
	rec := ordersRequest(t, store, http.MethodGet, "/orders/5", bearerFor(t, "3"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.gotUserID != 3 {
		t.Errorf("GetOrderForUser userID = %d, want 3 (the caller's own subject)", store.gotUserID)
	}
	if store.gotOrderID != "5" {
		t.Errorf("GetOrderForUser id = %q, want %q", store.gotOrderID, "5")
	}
}
