package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/middlewares"
	"ecommerce/app/models"
	"ecommerce/app/pricing"
	"ecommerce/app/views"
)

// cartStore is the slice of the store this controller uses, declared next
// to its caller and holding only the methods the actions call.
type cartStore interface {
	FindOrCreateCartByToken(ctx context.Context, token string) (models.Cart, error)
	CartContents(ctx context.Context, cartID int64) ([]models.CartLine, error)
	ListActivePriceRules(ctx context.Context) ([]models.PriceRule, error)
}

// CartController is the HTTP surface of the current cart: one cart per
// visitor, resolved by the bearer token's subject (middlewares.Subject)
// rather than an id in the URL (Bogie has no singular resource
// yet — docs/bogie-gaps.md — so Index stands in for what would be
// a singular show). The token itself, guest or (later) a real login, is
// POST /guest_sessions's job to mint, not this controller's: by the time a
// request reaches here it is required already, via s.WithUser in routes.go.
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

// Index handles GET /cart: the current cart's contents. The cart always
// already exists by the time a token reaches here — POST /guest_sessions
// creates it at mint time — so FindOrCreateCartByToken's "create" branch is
// a defensive fallback, not the normal path.
func (ctl *CartController) Index(c *gin.Context) {
	cart, err := ctl.Store.FindOrCreateCartByToken(c.Request.Context(), middlewares.Subject(c))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	lines, err := ctl.Store.CartContents(c.Request.Context(), cart.ID)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	rules, err := ctl.Store.ListActivePriceRules(c.Request.Context())
	if err != nil {
		ctl.fail(c, err)
		return
	}
	resp, err := cartView(lines, rules)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// fail maps a domain error to a status, the one place that mapping lives:
// not found is 404, a validation failure is 422, and anything
// else is logged and answered 500 without leaking its text.
func (ctl *CartController) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, views.ErrorMessage("no such cart"))
	case errors.Is(err, models.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, views.Error(err))
	default:
		ctl.Log.ErrorContext(c.Request.Context(), "cart: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
	}
}

// cartView converts a cart's lines to the wire shape and runs rules against
// the lines' total, computing every derived value here rather than storing
// any of them. Shared by CartController and CartItemsController, which both
// answer with a cart.
func cartView(lines []models.CartLine, rules []models.PriceRule) (views.CartResponse, error) {
	items := make([]views.CartLineResponse, 0, len(lines))
	var subtotal int32
	for _, l := range lines {
		lineSubtotal := l.UnitPriceCents * l.Quantity
		items = append(items, views.CartLineResponse{
			ID:             l.ID,
			ProductID:      l.ProductID,
			Name:           l.ProductName,
			ImageUrl:       l.ImageUrl,
			UnitPriceCents: l.UnitPriceCents,
			Quantity:       l.Quantity,
			SubtotalCents:  lineSubtotal,
		})
		subtotal += lineSubtotal
	}

	result, err := pricing.CalculateLines(pricing.CartLines(lines), rules)
	if err != nil {
		return views.CartResponse{}, err
	}

	applied := make([]views.AppliedRuleResponse, 0, len(result.AppliedRules))
	for _, a := range result.AppliedRules {
		applied = append(applied, views.AppliedRuleResponse{
			Name:          a.Name,
			DiscountCents: int32(a.PreviousCents - a.NewCents),
		})
	}

	return views.CartResponse{
		Items:         items,
		SubtotalCents: subtotal,
		DiscountCents: int32(result.TotalSavingsCents()),
		TotalCents:    int32(result.FinalCents),
		AppliedRules:  applied,
	}, nil
}
