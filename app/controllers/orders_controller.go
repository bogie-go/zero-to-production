package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ecommerce/app/middlewares"
	"ecommerce/app/models"
	"ecommerce/app/views"
)

// ordersStore is the slice of the store this controller uses, declared next
// to its caller (rule 2) and holding only the methods the actions call.
type ordersStore interface {
	FindOrCreateCartByToken(ctx context.Context, token string) (models.Cart, error)
	CreateOrderFromCart(ctx context.Context, userID, cartID int64) (models.Order, []models.OrderLine, error)
	ListOrdersForUser(ctx context.Context, userID int64, limit int32) ([]models.Order, error)
	GetOrderForUser(ctx context.Context, userID int64, id string) (models.Order, []models.OrderLine, error)
}

// OrdersController is the HTTP surface of checkout and order history.
// Every action needs a real account, not just a bearer token: a guest's
// subject is a bare uuid with no users row behind it, so it can never name
// the user_id an order needs. userID below is what tells the two apart.
type OrdersController struct {
	Store ordersStore
	Log   *slog.Logger
}

// NewOrdersController builds the controller.
func NewOrdersController(store ordersStore, log *slog.Logger) *OrdersController {
	return &OrdersController{Store: store, Log: log}
}

// SetupRoutes mounts the controller under its own prefix. Middleware the
// whole controller needs goes here; which group it sits in, and so how it
// is exposed, is decided by the routes.go that calls this.
func (ctl *OrdersController) SetupRoutes(router *gin.RouterGroup) {
	orders := router.Group("/orders")
	orders.POST("", ctl.Create)
	orders.GET("", ctl.Index)
	orders.GET("/:id", ctl.Show)
}

// Create handles POST /orders: checkout. It turns the caller's own cart
// into an order — stock decremented, prices snapshotted, cart emptied, all
// in one transaction (Store.CreateOrderFromCart) — or answers 422 if any
// line can no longer be filled, with nothing changed.
func (ctl *OrdersController) Create(c *gin.Context) {
	userID, ok := ctl.userID(c)
	if !ok {
		c.JSON(http.StatusForbidden, views.ErrorMessage("create an account to place an order"))
		return
	}
	cart, err := ctl.Store.FindOrCreateCartByToken(c.Request.Context(), middlewares.Subject(c))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	order, lines, err := ctl.Store.CreateOrderFromCart(c.Request.Context(), userID, cart.ID)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, orderView(order, lines))
}

// Index handles GET /orders: the caller's own order history, newest first.
func (ctl *OrdersController) Index(c *gin.Context) {
	userID, ok := ctl.userID(c)
	if !ok {
		c.JSON(http.StatusForbidden, views.ErrorMessage("create an account to see your orders"))
		return
	}
	orders, err := ctl.Store.ListOrdersForUser(c.Request.Context(), userID, 50)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	out := make([]views.OrderSummaryResponse, 0, len(orders))
	for _, o := range orders {
		out = append(out, views.OrderSummaryResponse{ID: o.ID, Status: o.Status, TotalCents: o.TotalCents, CreatedAt: o.CreatedAt})
	}
	c.JSON(http.StatusOK, out)
}

// Show handles GET /orders/:id: one order as a receipt, scoped to the
// caller — an id naming someone else's order 404s exactly as one naming
// nothing would, the same ownership scoping cart items and users use.
func (ctl *OrdersController) Show(c *gin.Context) {
	userID, ok := ctl.userID(c)
	if !ok {
		c.JSON(http.StatusForbidden, views.ErrorMessage("create an account to see your orders"))
		return
	}
	order, lines, err := ctl.Store.GetOrderForUser(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, orderView(order, lines))
}

// userID reports the caller's user id, and false if the bearer subject is
// not one — a guest's token, or anything else that isn't a bigint id.
func (ctl *OrdersController) userID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(middlewares.Subject(c), 10, 64)
	return id, err == nil
}

// orderView converts an order and its lines to the wire shape.
func orderView(o models.Order, lines []models.OrderLine) views.OrderResponse {
	items := make([]views.OrderLineResponse, 0, len(lines))
	for _, l := range lines {
		items = append(items, views.OrderLineResponse{
			ProductID:      l.ProductID,
			Name:           l.ProductName,
			ImageUrl:       l.ImageUrl,
			UnitPriceCents: l.UnitPriceCents,
			Quantity:       l.Quantity,
			SubtotalCents:  l.UnitPriceCents * l.Quantity,
		})
	}
	return views.OrderResponse{
		ID:            o.ID,
		Status:        o.Status,
		Items:         items,
		SubtotalCents: o.SubtotalCents,
		DiscountCents: o.DiscountCents,
		TotalCents:    o.TotalCents,
	}
}

// fail maps a domain error to a status, the one place that mapping lives:
// not found is 404, a validation failure (empty cart, a line out of stock)
// is 422 as in Rails, and anything else is logged and answered 500 without
// leaking its text.
func (ctl *OrdersController) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, views.ErrorMessage("no such order"))
	case errors.Is(err, models.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, views.Error(err))
	default:
		ctl.Log.ErrorContext(c.Request.Context(), "orders: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
	}
}
