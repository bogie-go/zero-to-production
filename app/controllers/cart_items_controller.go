package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/middlewares"
	"ecommerce/app/views"
)

// cartItemsStore is the slice of the store this controller uses, declared
// next to its caller (rule 2) and holding only the methods the actions call.
type cartItemsStore interface {
	FindOrCreateCartByToken(ctx context.Context, token string) (domain.Cart, error)
	AddCartItem(ctx context.Context, cartID, productID int64, quantity int32) (domain.CartItem, error)
	UpdateCartItemQuantity(ctx context.Context, id string, quantity int32) (domain.CartItem, error)
	DeleteCartItem(ctx context.Context, id string) error
	CartContents(ctx context.Context, cartID int64) ([]domain.CartLine, error)
}

// CartItemsController is the HTTP surface of a cart's line items: adding,
// changing the quantity of, and removing one. Every action answers with the
// cart's full contents afterward (CartController.cartView), the same shape
// GET /cart gives, so the UI can always just replace its cart state with
// whatever the response was — no action needs to say which line it was.
type CartItemsController struct {
	Store cartItemsStore
	Log   *slog.Logger
}

// NewCartItemsController builds the controller.
func NewCartItemsController(store cartItemsStore, log *slog.Logger) *CartItemsController {
	return &CartItemsController{Store: store, Log: log}
}

// SetupRoutes mounts the controller under its own prefix. Middleware the
// whole controller needs goes here; which group it sits in, and so how it is
// exposed, is decided by the routes.go that calls this.
func (ctl *CartItemsController) SetupRoutes(router *gin.RouterGroup) {
	cart_items := router.Group("/cart_items")
	cart_items.POST("", ctl.Create)
	cart_items.PATCH("/:id", ctl.Update)
	cart_items.DELETE("/:id", ctl.Destroy)
}

// Create handles POST /cart_items: add a product to the current cart, or
// increase its quantity if it is already there.
func (ctl *CartItemsController) Create(c *gin.Context) {
	var req views.AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	cart, err := ctl.Store.FindOrCreateCartByToken(c.Request.Context(), middlewares.Subject(c))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	if _, err := ctl.Store.AddCartItem(c.Request.Context(), cart.ID, req.ProductID, req.Quantity); err != nil {
		ctl.fail(c, err)
		return
	}
	ctl.respondWithCart(c, http.StatusCreated, cart.ID)
}

// Update handles PATCH /cart_items/:id: set a line's quantity.
func (ctl *CartItemsController) Update(c *gin.Context) {
	var req views.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	item, err := ctl.Store.UpdateCartItemQuantity(c.Request.Context(), c.Param("id"), req.Quantity)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	ctl.respondWithCart(c, http.StatusOK, item.CartID)
}

// Destroy handles DELETE /cart_items/:id: remove a line. The cart it
// belonged to has to be resolved before the delete, not after — there is
// nothing left to ask once the line is gone.
func (ctl *CartItemsController) Destroy(c *gin.Context) {
	cart, err := ctl.Store.FindOrCreateCartByToken(c.Request.Context(), middlewares.Subject(c))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	if err := ctl.Store.DeleteCartItem(c.Request.Context(), c.Param("id")); err != nil {
		ctl.fail(c, err)
		return
	}
	ctl.respondWithCart(c, http.StatusOK, cart.ID)
}

// respondWithCart loads the cart's current contents and answers with them,
// the shape every action in this controller answers with.
func (ctl *CartItemsController) respondWithCart(c *gin.Context, status int, cartID int64) {
	lines, err := ctl.Store.CartContents(c.Request.Context(), cartID)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(status, cartView(lines))
}

// fail maps a domain error to a status, the one place that mapping lives:
// not found is 404, a validation failure (quantity <= 0, more than the
// product's stock, no such product) is 422 as in Rails, and anything else
// is logged and answered 500 without leaking its text.
func (ctl *CartItemsController) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, views.ErrorMessage("no such cart item"))
	case errors.Is(err, domain.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, views.Error(err))
	default:
		ctl.Log.ErrorContext(c.Request.Context(), "cart_items: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
	}
}
