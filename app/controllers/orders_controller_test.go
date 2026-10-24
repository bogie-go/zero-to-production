package controllers_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers"
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
func ordersRequest(store *fakeOrdersStore, method, path, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	guarded := router.Group("", middlewares.RequireUser(testTokens))
	controllers.NewOrdersController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(guarded)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("orders", func() {
	var store *fakeOrdersStore

	BeforeEach(func() {
		store = &fakeOrdersStore{}
	})

	// Every action is mounted: a route that answers 404 was never registered.
	It("mounts GET /orders", func() {
		rec := ordersRequest(store, http.MethodGet, "/orders", bearerFor("1"))
		Expect(rec.Code).NotTo(Equal(http.StatusNotFound))
	})

	Describe("POST /orders", func() {
		// A guest's token is a valid bearer token — RequireUser lets it through —
		// but its subject is a uuid, not a user id, so checkout answers 403
		// without ever reaching the store: there is no account to put on the order.
		It("rejects a guest token without reaching the store", func() {
			rec := ordersRequest(store, http.MethodPost, "/orders", bearerFor("a-guest-uuid-not-an-id"))

			Expect(rec.Code).To(Equal(http.StatusForbidden), rec.Body.String())
			Expect(store.checkoutCalls).To(BeZero())
		})

		// A real user's checkout resolves their own cart by subject, then checks
		// out that cart under their own user id — never an id read from the
		// request.
		It("checks out the caller's own cart", func() {
			store.cart = domain.Cart{ID: 42}
			store.order = domain.Order{ID: 7, Status: "pending", SubtotalCents: 1000, TotalCents: 1000}
			store.lines = []domain.OrderLine{{ProductID: 10, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2}}

			rec := ordersRequest(store, http.MethodPost, "/orders", bearerFor("9"))

			Expect(rec.Code).To(Equal(http.StatusCreated), rec.Body.String())
			Expect(store.gotCartToken).To(Equal("9"), "cart resolved for the caller's own subject")
			Expect(store.gotUserID).To(Equal(int64(9)))
			Expect(store.gotCartID).To(Equal(int64(42)), "the caller's own cart")
			Expect(rec.Body.String()).To(ContainSubstring(`"id":7`))
		})

		// A line out of stock by checkout time is a 422, same as any other
		// validation failure, not a 500.
		It("reports an out-of-stock line as 422", func() {
			store.cart = domain.Cart{ID: 1}
			store.err = fmt.Errorf("%w: Widget no longer has enough stock", domain.ErrInvalid)

			rec := ordersRequest(store, http.MethodPost, "/orders", bearerFor("9"))

			Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
		})
	})

	// GET /orders/:id is scoped to the caller's own user id; a request for
	// someone else's order is the store's job to 404, proven in
	// app/models/orders_extra_test.go — this just confirms the controller
	// passes the caller's own id through, never one from the URL or body.
	Describe("GET /orders/:id", func() {
		It("uses the caller's own user id", func() {
			store.order = domain.Order{ID: 5}

			rec := ordersRequest(store, http.MethodGet, "/orders/5", bearerFor("3"))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(store.gotUserID).To(Equal(int64(3)), "the caller's own subject")
			Expect(store.gotOrderID).To(Equal("5"))
		})
	})
})
