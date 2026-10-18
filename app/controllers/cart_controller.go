package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/views"
)

// cartStore is the slice of the store this controller uses, declared next
// to its caller (rule 2) and holding only the methods the actions call.
type cartStore interface {
	FindOrCreateCartByToken(ctx context.Context, token string) (domain.Cart, error)
	CartContents(ctx context.Context, cartID int64) ([]domain.CartLine, error)
}

// CartController is the HTTP surface of the current cart: one cart per
// visitor, resolved by currentToken rather than an id in the URL (there is
// no Rails-style singular resource in Bogie yet — docs/bogie-gaps.md — so
// Index stands in for what would be a singular show).
type CartController struct {
	Store cartStore
	Log   *slog.Logger
}

// NewCartController builds the controller.
func NewCartController(store cartStore, log *slog.Logger) *CartController {
	return &CartController{Store: store, Log: log}
}

// SetupRoutes mounts the controller under its own prefix. Middleware the
// whole controller needs goes here; which group it sits in, and so how it is
// exposed, is decided by the routes.go that calls this.
func (ctl *CartController) SetupRoutes(router *gin.RouterGroup) {
	cart := router.Group("/cart")
	cart.GET("", ctl.Index)
}

// cartTokenHeader names the visitor's cart, like Rails's session[:cart_id]
// but as a plain header instead of a cookie: this is an API, and Task D's
// own auth will be token-based too, so one mental model carries through
// instead of two. It is never cryptographically meaningful on its own — it
// just has to be hard to guess — unlike a Task D auth token, which proves
// who you are and would need real care (expiry, a digest in storage, the
// way Bogie's own api_key shape already does it) if the two were ever
// confused. They are not: a leaked cart token lets someone see a stranger's
// cart, nothing more.
const cartTokenHeader = "X-Cart-Token"

// currentToken resolves the visitor's cart token: whatever the client sent
// back, or a freshly minted one on a first visit. Either way it is echoed
// as a response header, so the client's only job is "always send what you
// have, always save what comes back" — no token generation of its own.
func currentToken(c *gin.Context) string {
	token := c.GetHeader(cartTokenHeader)
	if token == "" {
		token = randomToken()
	}
	c.Header(cartTokenHeader, token)
	return token
}

// randomToken is 32 bytes from the OS's CSPRNG, hex-encoded: unguessable,
// nothing to verify or decode. crypto/rand failing means the OS's entropy
// source is broken, not something a request can recover from; the panic is
// caught by middlewares.Recover like any other, and answered with
// public/500.html rather than crashing the process.
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// Index handles GET /cart: the current cart's contents, creating it empty
// on first visit.
func (ctl *CartController) Index(c *gin.Context) {
	cart, err := ctl.Store.FindOrCreateCartByToken(c.Request.Context(), currentToken(c))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	lines, err := ctl.Store.CartContents(c.Request.Context(), cart.ID)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, cartView(lines))
}

// fail maps a domain error to a status, the one place that mapping lives:
// not found is 404, a validation failure is 422 as in Rails, and anything
// else is logged and answered 500 without leaking its text.
func (ctl *CartController) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, views.ErrorMessage("no such cart"))
	case errors.Is(err, domain.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, views.Error(err))
	default:
		ctl.Log.ErrorContext(c.Request.Context(), "cart: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
	}
}

// cartView converts a cart's lines to the wire shape, computing each line's
// subtotal and the cart's total here: derived values, not stored. Shared by
// CartController and CartItemsController, which both answer with a cart.
func cartView(lines []domain.CartLine) views.CartResponse {
	out := views.CartResponse{Items: make([]views.CartLineResponse, 0, len(lines))}
	for _, l := range lines {
		subtotal := l.UnitPriceCents * l.Quantity
		out.Items = append(out.Items, views.CartLineResponse{
			ID:             l.ID,
			ProductID:      l.ProductID,
			Name:           l.ProductName,
			ImageUrl:       l.ImageUrl,
			UnitPriceCents: l.UnitPriceCents,
			Quantity:       l.Quantity,
			SubtotalCents:  subtotal,
		})
		out.TotalCents += subtotal
	}
	return out
}
