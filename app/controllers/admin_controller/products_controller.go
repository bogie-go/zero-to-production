package admin_controller

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/views"
)

// productsStore is the slice of the store this controller uses, declared
// next to its caller (rule 2) and holding only the methods the actions
// call. *models.Store satisfies it.
type productsStore interface {
	GetProduct(ctx context.Context, id string) (domain.Product, error)
	ListProducts(ctx context.Context, limit int32) ([]domain.Product, error)
	CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error)
	UpdateProduct(ctx context.Context, p domain.Product) (domain.Product, error)
	DeleteProduct(ctx context.Context, id string) error
	ArchiveProduct(ctx context.Context, id string) (domain.Product, error)
	RestoreProduct(ctx context.Context, id string) (domain.Product, error)
}

// productsListLimit mirrors the public ProductsController's own.
const productsListLimit = 100

// ProductsController is the admin surface of products: the full five
// actions, unlike the public ProductsController's read-only two. Only
// reachable under /admin, behind the namespace's WithStaff.
type ProductsController struct {
	Store productsStore
	Log   *slog.Logger
}

// NewProductsController builds the controller.
func NewProductsController(store productsStore, log *slog.Logger) *ProductsController {
	return &ProductsController{Store: store, Log: log}
}

// SetupRoutes mounts the controller under its own prefix. Middleware the
// whole controller needs goes here; which group it sits in, and so how it
// is exposed, is decided by the routes.go that calls this.
func (ctl *ProductsController) SetupRoutes(router *gin.RouterGroup) {
	products := router.Group("/products")
	products.GET("", ctl.Index)
	products.GET("/:id", ctl.Show)
	products.POST("", ctl.Create)
	products.PATCH("/:id", ctl.Update)
	products.DELETE("/:id", ctl.Destroy)
	products.POST("/:id/archive", ctl.Archive)
	products.POST("/:id/restore", ctl.Restore)
}

// Index handles GET /admin/products.
func (ctl *ProductsController) Index(c *gin.Context) {
	products, err := ctl.Store.ListProducts(c.Request.Context(), productsListLimit)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, productsView(products))
}

// Show handles GET /admin/products/:id.
func (ctl *ProductsController) Show(c *gin.Context) {
	product, err := ctl.Store.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, productView(product))
}

// Create handles POST /admin/products.
func (ctl *ProductsController) Create(c *gin.Context) {
	var req views.ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	product, err := ctl.Store.CreateProduct(c.Request.Context(), domain.Product{
		Name: req.Name, Description: req.Description, PriceCents: req.PriceCents, Stock: req.Stock, ImageUrl: req.ImageUrl,
		CategoryID: req.CategoryID,
	})
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, productView(product))
}

// Update handles PATCH /admin/products/:id.
func (ctl *ProductsController) Update(c *gin.Context) {
	var req views.ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	existing, err := ctl.Store.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	existing.Name = req.Name
	existing.Description = req.Description
	existing.PriceCents = req.PriceCents
	existing.Stock = req.Stock
	existing.ImageUrl = req.ImageUrl
	existing.CategoryID = req.CategoryID
	product, err := ctl.Store.UpdateProduct(c.Request.Context(), existing)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, productView(product))
}

// Destroy handles DELETE /admin/products/:id: a real delete, which the
// database refuses (422) once a cart or an order names the product — use
// Archive for those.
func (ctl *ProductsController) Destroy(c *gin.Context) {
	if err := ctl.Store.DeleteProduct(c.Request.Context(), c.Param("id")); err != nil {
		ctl.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Archive handles POST /admin/products/:id/archive: the soft delete. The
// product leaves the shop and new carts but stays in every old order.
func (ctl *ProductsController) Archive(c *gin.Context) {
	product, err := ctl.Store.ArchiveProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, productView(product))
}

// Restore handles POST /admin/products/:id/restore: back on sale.
func (ctl *ProductsController) Restore(c *gin.Context) {
	product, err := ctl.Store.RestoreProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, productView(product))
}

// fail maps a domain error to a status, the one place that mapping lives:
// not found is 404, a validation failure (a blank name, a negative price
// or stock) is 422 as in Rails, and anything else is logged and answered
// 500 without leaking its text.
func (ctl *ProductsController) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, views.ErrorMessage("no such product"))
	case errors.Is(err, domain.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, views.Error(err))
	default:
		ctl.Log.ErrorContext(c.Request.Context(), "admin/products: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
	}
}

// The controller converts between the domain and the wire. views never
// imports domain, so this is where the conversions live — the same split
// the public ProductsController uses, duplicated rather than shared, since
// admin_controller and controllers are different packages and this is the
// only place it's needed twice.

func productView(p domain.Product) views.ProductResponse {
	return views.ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		PriceCents:  p.PriceCents,
		Stock:       p.Stock,
		ImageUrl:    p.ImageUrl,
		ArchivedAt:  p.ArchivedAt,
		CategoryID:  p.CategoryID,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func productsView(products []domain.Product) views.ProductsResponse {
	out := views.ProductsResponse{Products: make([]views.ProductResponse, 0, len(products))}
	for _, p := range products {
		out.Products = append(out.Products, productView(p))
	}
	return out
}
