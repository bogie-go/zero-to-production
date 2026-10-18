package controllers

import (
	"context"
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
// Index stands in for what would be a singular show). Task C starts every
// visitor on the one shared cart named by the empty token; a later task
// swaps currentToken for a cookie, and nothing else here changes.
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

// currentToken names the visitor's cart. Always the empty string for now:
// every visitor shares the one demo cart. A cookie-backed version replaces
// just this function later; every caller already takes it as a token, not
// as "the demo cart", so nothing above it changes.
func currentToken(c *gin.Context) string { return "" }

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
