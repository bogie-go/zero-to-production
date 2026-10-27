package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ecommerce/app/models"
	"ecommerce/app/views"
)

// productsStore is the slice of the store this controller uses, declared next
// to its caller and holding only the methods the actions call. The shop
// only ever sees active products: an archived one is not found.
// *adapter.Store satisfies it, and app/application.go passes it in; the
// test's fake is a few lines.
type productsStore interface {
	GetActiveProduct(ctx context.Context, id string) (models.Product, error)
	ListActiveProducts(ctx context.Context, categoryID *int64, limit, offset int32) ([]models.Product, error)
	CountActiveProducts(ctx context.Context, categoryID *int64) (int64, error)
}

// The catalog's page size: twelve unless the client asks (?per_page=), never
// more than fifty.
const (
	productsPerPage    = 12
	productsMaxPerPage = 50
)

// ProductsController is the HTTP surface of products: its dependencies, its
// routes and its actions, all in one file.
type ProductsController struct {
	Store productsStore
	Log   *slog.Logger
}

// NewProductsController builds the controller.
func NewProductsController(store productsStore, log *slog.Logger) *ProductsController {
	return &ProductsController{Store: store, Log: log}
}

// SetupRoutes mounts the controller under its own prefix. Middleware the
// whole controller needs goes here; which group it sits in, and so how it is
// exposed, is decided by the routes.go that calls this.
func (ctl *ProductsController) SetupRoutes(router *gin.RouterGroup) {
	products := router.Group("/products")
	products.GET("", ctl.Index)
	products.GET("/:id", ctl.Show)
}

// Index handles GET /products, optionally ?category_id=N, ?page=N and
// ?per_page=N.
//
// A handler converts HTTP into domain values, calls one service function,
// and converts the result back. Nothing below app/controllers sees
// *gin.Context.
func (ctl *ProductsController) Index(c *gin.Context) {
	// ?category_id=N narrows the catalog to one category; without it, all.
	var categoryID *int64
	if raw := c.Query("category_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, views.ErrorMessage("category_id must be a number"))
			return
		}
		categoryID = &id
	}
	page, err := views.NewPageRequest(c.Query("page"), c.Query("per_page"), productsPerPage, productsMaxPerPage)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, views.Error(err))
		return
	}
	products, err := ctl.Store.ListActiveProducts(c.Request.Context(), categoryID, page.PerPage, page.Offset())
	if err != nil {
		ctl.fail(c, err)
		return
	}
	total, err := ctl.Store.CountActiveProducts(c.Request.Context(), categoryID)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, productsView(products, page.Pagination(total)))
}

// Show handles GET /products/:id.
//
// A handler converts HTTP into domain values, calls one service function,
// and converts the result back. Nothing below app/controllers sees
// *gin.Context.
func (ctl *ProductsController) Show(c *gin.Context) {
	product, err := ctl.Store.GetActiveProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, productView(product))
}

// fail maps a domain error to a status, the one place that mapping lives:
// not found is 404, and anything else is logged and answered 500 without
// leaking its text.
func (ctl *ProductsController) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, views.ErrorMessage("no such product"))
	default:
		ctl.Log.ErrorContext(c.Request.Context(), "products: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
	}
}

// The controller converts between the domain and the wire. views never
// imports models, so this is where ProductResponse is built.

func productView(p models.Product) views.ProductResponse {
	return views.ProductResponse{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		PriceCents:   p.PriceCents,
		Stock:        p.Stock,
		ImageUrl:     p.ImageUrl,
		ArchivedAt:   p.ArchivedAt,
		CategoryID:   p.CategoryID,
		CategoryName: p.CategoryName,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func productsView(products []models.Product, pagination views.Pagination) views.ProductsResponse {
	out := views.ProductsResponse{Products: make([]views.ProductResponse, 0, len(products)), Pagination: pagination}
	for _, p := range products {
		out.Products = append(out.Products, productView(p))
	}
	return out
}
