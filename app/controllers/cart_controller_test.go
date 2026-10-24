package controllers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers"
	"ecommerce/app/domain"
	"ecommerce/app/middlewares"
	"ecommerce/app/pricing"
	"ecommerce/app/services/token"
)

// testTokens mints and verifies bearer tokens for every controller test in
// this package that sits behind s.WithUser, the same Maker
// middlewares.RequireUser checks against in production.
var testTokens = token.Maker{Secret: []byte("a-test-secret-of-at-least-thirty-two-bytes"), Issuer: "ecommerce-test"}

// bearerFor mints a short-lived token for subject, for a test to present.
func bearerFor(subject string) string {
	signed, err := testTokens.Mint(subject, time.Hour)
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
	return "Bearer " + signed
}

// fakeCartStore is a fake of the store interface declared beside the
// controller: it records what the controller asked for and answers with
// what the test chose.
type fakeCartStore struct {
	cart  domain.Cart
	lines []domain.CartLine
	rules []domain.PriceRule
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

func (f *fakeCartStore) ListActivePriceRules(_ context.Context) ([]domain.PriceRule, error) {
	return f.rules, nil
}

// cartRequest mounts the controller behind the same RequireUser middleware
// s.WithUser wraps it with in production, so these tests see exactly what a
// real request would: no token, no access. authorization, if not "", is
// sent as the request's Authorization header verbatim — use bearerFor for a
// valid one.
func cartRequest(store *fakeCartStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	guarded := router.Group("", middlewares.RequireUser(testTokens))
	controllers.NewCartController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(guarded)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("GET /cart", func() {
	var store *fakeCartStore

	BeforeEach(func() {
		store = &fakeCartStore{}
	})

	// Every action is mounted: a route that answers 404 was never registered.
	It("is mounted", func() {
		rec := cartRequest(store, http.MethodGet, "/cart", "", bearerFor("visitor-1"))
		Expect(rec.Code).NotTo(Equal(http.StatusNotFound))
	})

	Context("with a cart of lines", func() {
		BeforeEach(func() {
			store.cart = domain.Cart{ID: 1}
			store.lines = []domain.CartLine{
				{ID: 1, ProductID: 10, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2},
			}
		})

		// Index returns the current cart's contents, with each line's subtotal and
		// the cart's total computed, not stored.
		It("returns the contents with computed totals", func() {
			rec := cartRequest(store, http.MethodGet, "/cart", "", bearerFor("visitor-1"))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(rec.Body.String()).To(ContainSubstring(`"subtotal_cents":1000`), "a line subtotal of 1000")
			Expect(rec.Body.String()).To(ContainSubstring(`"total_cents":1000`), "a cart total of 1000")
		})

		// An active price rule discounts the cart's total, and the response says
		// which rule did it and by how much — the pricing engine wired in, not just
		// unit-tested in isolation.
		It("applies active price rules", func() {
			percentage, err := json.Marshal(pricing.PercentageDiscountRule{Percentage: 20})
			Expect(err).NotTo(HaveOccurred())
			store.rules = []domain.PriceRule{{ID: 1, Name: "Flash Sale", Kind: "percentage_discount", Properties: percentage, Active: true}}

			rec := cartRequest(store, http.MethodGet, "/cart", "", bearerFor("visitor-1"))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(rec.Body.String()).To(ContainSubstring(`"subtotal_cents":1000`))
			Expect(rec.Body.String()).To(ContainSubstring(`"discount_cents":200`))
			Expect(rec.Body.String()).To(ContainSubstring(`"total_cents":800`), "a total of 800 after the discount")
			Expect(rec.Body.String()).To(ContainSubstring(`"name":"Flash Sale"`))
		})
	})

	// The cart is resolved by the token's subject: whichever visitor the
	// bearer token names is whose cart comes back.
	It("resolves the cart by the token's subject", func() {
		cartRequest(store, http.MethodGet, "/cart", "", bearerFor("visitor-42"))
		Expect(store.gotToken).To(Equal("visitor-42"))
	})

	// No bearer token at all is 401 — the detailed cases (expired, garbage,
	// wrong scheme) are already covered by
	// app/middlewares/require_user_test.go; this just proves /cart is actually
	// mounted behind that guard, not bypassing it.
	It("requires a bearer token", func() {
		rec := cartRequest(store, http.MethodGet, "/cart", "", "")
		Expect(rec.Code).To(Equal(http.StatusUnauthorized))
	})

	// An unexpected error is a 500 that says nothing about its cause.
	It("answers 500 without detail for an unexpected error", func() {
		store.err = errors.New("connection refused to db-1")

		rec := cartRequest(store, http.MethodGet, "/cart", "", bearerFor("visitor-1"))

		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		Expect(rec.Body.String()).NotTo(ContainSubstring("db-1"))
	})
})
