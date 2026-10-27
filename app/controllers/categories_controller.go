package controllers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/models"
	"ecommerce/app/views"
)

// categoriesStore is the slice of the store this controller uses, declared
// next to its caller (rule 2). *db.Store satisfies it.
type categoriesStore interface {
	ListCategoriesWithCounts(ctx context.Context) ([]models.Category, error)
}

// CategoriesController is the shop's read-only view of categories: what the
// catalog's filter lists. Managing them is the admin's (admin_controller).
type CategoriesController struct {
	Store categoriesStore
	Log   *slog.Logger
}

// NewCategoriesController builds the controller.
func NewCategoriesController(store categoriesStore, log *slog.Logger) *CategoriesController {
	return &CategoriesController{Store: store, Log: log}
}

// SetupRoutes mounts the controller under its own prefix.
func (ctl *CategoriesController) SetupRoutes(router *gin.RouterGroup) {
	categories := router.Group("/categories")
	categories.GET("", ctl.Index)
}

// Index handles GET /categories: every category by name, each with the number
// of active products in it.
func (ctl *CategoriesController) Index(c *gin.Context) {
	categories, err := ctl.Store.ListCategoriesWithCounts(c.Request.Context())
	if err != nil {
		ctl.Log.ErrorContext(c.Request.Context(), "categories: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
		return
	}
	out := views.CategoriesResponse{Categories: make([]views.CategoryResponse, 0, len(categories))}
	for _, cat := range categories {
		out.Categories = append(out.Categories, views.CategoryResponse{ID: cat.ID, Name: cat.Name, ProductCount: cat.ProductCount})
	}
	c.JSON(http.StatusOK, out)
}
