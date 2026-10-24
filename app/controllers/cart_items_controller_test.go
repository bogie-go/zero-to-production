package controllers_test

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

	"ecommerce/app/controllers"
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
	gotCartID    int64
}

func (f *fakeCartItemsStore) FindOrCreateCartByToken(_ context.Context, token string) (domain.Cart, error) {
	return f.cart, f.err
}

func (f *fakeCartItemsStore) AddCartItem(_ context.Context, cartID, productID int64, quantity int32) (domain.CartItem, error) {
	f.gotProductID, f.gotQuantity = productID, quantity
	return f.item, f.err
}

func (f *fakeCartItemsStore) UpdateCartItemQuantity(_ context.Context, cartID int64, id string, quantity int32) (domain.CartItem, error) {
	f.gotCartID, f.gotID, f.gotQuantity = cartID, id, quantity
	return f.item, f.err
}

func (f *fakeCartItemsStore) DeleteCartItemForCart(_ context.Context, cartID int64, id string) error {
	f.gotCartID, f.gotID = cartID, id
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
func cartItemsRequest(store *fakeCartItemsStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	guarded := router.Group("", middlewares.RequireUser(testTokens))
	controllers.NewCartItemsController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(guarded)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("cart_items", func() {
	var (
		store *fakeCartItemsStore
		auth  string
	)

	BeforeEach(func() {
		store = &fakeCartItemsStore{}
		auth = bearerFor("visitor-1")
	})

	// Every action is mounted: a route that answers 404 was never registered.
	DescribeTable("routes",
		func(method, path, body string) {
			rec := cartItemsRequest(store, method, path, body, auth)
			Expect(rec.Code).NotTo(Equal(http.StatusNotFound))
		},
		Entry("POST /cart_items", http.MethodPost, "/cart_items", "{}"),
		Entry("PATCH /cart_items/:id", http.MethodPatch, "/cart_items/1", "{}"),
		Entry("DELETE /cart_items/:id", http.MethodDelete, "/cart_items/1", ""),
	)

	// No bearer token at all is 401 — the detailed cases are already covered
	// by app/middlewares/require_user_test.go; this just proves cart_items is
	// actually mounted behind that guard.
	It("requires a bearer token", func() {
		rec := cartItemsRequest(store, http.MethodPost, "/cart_items", "{}", "")
		Expect(rec.Code).To(Equal(http.StatusUnauthorized))
	})

	Describe("POST /cart_items", func() {
		// Create passes the body through to AddCartItem and answers with the
		// cart's contents, not just the one line.
		It("passes the body through and returns the cart", func() {
			store.cart = domain.Cart{ID: 1}
			store.lines = []domain.CartLine{{ID: 5, ProductID: 10, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2}}

			rec := cartItemsRequest(store, http.MethodPost, "/cart_items", `{"product_id":10,"quantity":2}`, auth)

			Expect(rec.Code).To(Equal(http.StatusCreated), rec.Body.String())
			Expect(store.gotProductID).To(Equal(int64(10)))
			Expect(store.gotQuantity).To(Equal(int32(2)))
			Expect(rec.Body.String()).To(ContainSubstring(`"total_cents":1000`))
		})

		// Validation (quantity <= 0, more than stock, no such product) is the
		// store's; the controller's only job is to say 422.
		It("answers 422 when the store says invalid", func() {
			store.err = errors.Join(domain.ErrInvalid, errors.New("only 1 in stock"))

			rec := cartItemsRequest(store, http.MethodPost, "/cart_items", `{"product_id":10,"quantity":99}`, auth)

			Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
		})

		It("answers 400 for malformed JSON", func() {
			rec := cartItemsRequest(store, http.MethodPost, "/cart_items", `{"nope":`, auth)
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		// An unexpected error is a 500 that says nothing about its cause.
		It("answers 500 without detail for an unexpected error", func() {
			store.err = errors.New("connection refused to db-1")

			rec := cartItemsRequest(store, http.MethodPost, "/cart_items", `{"product_id":1,"quantity":1}`, auth)

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			Expect(rec.Body.String()).NotTo(ContainSubstring("db-1"))
		})
	})

	Describe("PATCH /cart_items/:id", func() {
		It("changes the quantity and returns the cart", func() {
			store.cart = domain.Cart{ID: 1}
			store.item = domain.CartItem{ID: 5, CartID: 1}

			rec := cartItemsRequest(store, http.MethodPatch, "/cart_items/5", `{"quantity":3}`, auth)

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(store.gotID).To(Equal("5"))
			Expect(store.gotQuantity).To(Equal(int32(3)))
		})

		// Update resolves the cart from the caller's own token before touching the
		// line, so the store is always asked "is this line in cart X", never
		// "update this line" with no cart attached — the fix for the cross-cart
		// ownership bug found in task C2.
		It("scopes to the caller's cart", func() {
			store.cart = domain.Cart{ID: 42}

			cartItemsRequest(store, http.MethodPatch, "/cart_items/5", `{"quantity":3}`, auth)

			Expect(store.gotCartID).To(Equal(int64(42)))
		})

		It("answers 404 for an unknown line", func() {
			store.err = domain.ErrNotFound

			rec := cartItemsRequest(store, http.MethodPatch, "/cart_items/nope", `{"quantity":1}`, auth)

			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("DELETE /cart_items/:id", func() {
		It("removes the line and returns the cart", func() {
			store.cart = domain.Cart{ID: 1}

			rec := cartItemsRequest(store, http.MethodDelete, "/cart_items/5", "", auth)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(store.gotID).To(Equal("5"))
			Expect(store.gotCartID).To(Equal(int64(1)))
		})
	})
})
