package controllers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/views"
)

// ProductsController is the HTTP surface of products: its dependencies, its routes
// and its actions, one Rails controller in one file. Dependencies are small
// interfaces declared here (rule 2), holding only the methods the actions
// call, and the real thing is passed in from app/application.go, where this
// controller is wired:
//
//	type productsService interface {
//		Get(ctx context.Context, id string) (domain.Thing, error)
//	}
type ProductsController struct {
	Log *slog.Logger
}

// NewProductsController builds the controller.
func NewProductsController(log *slog.Logger) *ProductsController {
	return &ProductsController{Log: log}
}

// SetupRoutes mounts the controller under its own prefix. Middleware the
// whole controller needs goes here; which group it sits in, and so how it is
// exposed, is decided by the routes.go that calls this.
func (ctl *ProductsController) SetupRoutes(router *gin.RouterGroup) {
	products := router.Group("/products")
	products.GET("", ctl.Index)
	products.GET("/:id", ctl.Show)
}

// Index handles GET /products.
//
// A handler converts HTTP into domain values, calls one service function,
// and converts the result back. Nothing below app/controllers sees
// *gin.Context (rule 5).
func (ctl *ProductsController) Index(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, views.ErrorMessage("products#index is not implemented yet"))
}

// Show handles GET /products/:id.
//
// A handler converts HTTP into domain values, calls one service function,
// and converts the result back. Nothing below app/controllers sees
// *gin.Context (rule 5).
func (ctl *ProductsController) Show(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, views.ErrorMessage("products#show is not implemented yet"))
}
