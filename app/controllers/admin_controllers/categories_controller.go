package admin_controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/models"
	"ecommerce/app/views"
)

// categoriesStore is the slice of the store this controller uses, declared
// next to its caller. *adapter.Store satisfies it.
type categoriesStore interface {
	ListCategoriesWithCounts(ctx context.Context) ([]models.Category, error)
	GetCategory(ctx context.Context, id string) (models.Category, error)
	CreateCategory(ctx context.Context, c models.Category) (models.Category, error)
	UpdateCategory(ctx context.Context, c models.Category) (models.Category, error)
	DeleteCategory(ctx context.Context, id string) error
}

// CategoriesController is the admin surface of categories: the full five
// actions, only reachable under /admin, behind the namespace's WithStaff.
// Deleting a category does not delete its products: the database sets their
// category to none (ON DELETE SET NULL).
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
	categories.GET("/:id", ctl.Show)
	categories.POST("", ctl.Create)
	categories.PATCH("/:id", ctl.Update)
	categories.DELETE("/:id", ctl.Destroy)
}

// Index handles GET /admin/categories, each with its active-product count.
func (ctl *CategoriesController) Index(c *gin.Context) {
	categories, err := ctl.Store.ListCategoriesWithCounts(c.Request.Context())
	if err != nil {
		ctl.fail(c, err)
		return
	}
	out := views.AdminCategoriesResponse{Categories: make([]views.AdminCategoryResponse, 0, len(categories))}
	for _, cat := range categories {
		out.Categories = append(out.Categories, categoryView(cat))
	}
	c.JSON(http.StatusOK, out)
}

// Show handles GET /admin/categories/:id.
func (ctl *CategoriesController) Show(c *gin.Context) {
	cat, err := ctl.Store.GetCategory(c.Request.Context(), c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, categoryView(cat))
}

// Create handles POST /admin/categories.
func (ctl *CategoriesController) Create(c *gin.Context) {
	var req views.AdminCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	cat, err := ctl.Store.CreateCategory(c.Request.Context(), models.Category{Name: req.Name})
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, categoryView(cat))
}

// Update handles PATCH /admin/categories/:id.
func (ctl *CategoriesController) Update(c *gin.Context) {
	var req views.AdminCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	existing, err := ctl.Store.GetCategory(c.Request.Context(), c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	existing.Name = req.Name
	cat, err := ctl.Store.UpdateCategory(c.Request.Context(), existing)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, categoryView(cat))
}

// Destroy handles DELETE /admin/categories/:id.
func (ctl *CategoriesController) Destroy(c *gin.Context) {
	if err := ctl.Store.DeleteCategory(c.Request.Context(), c.Param("id")); err != nil {
		ctl.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// fail maps a domain error to a status, the one place that mapping lives: not
// found is 404, a validation failure (a blank or duplicate name) is 422, and
// anything else is logged and answered 500 without leaking its text.
func (ctl *CategoriesController) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, views.ErrorMessage("no such category"))
	case errors.Is(err, models.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, views.Error(err))
	default:
		ctl.Log.ErrorContext(c.Request.Context(), "admin/categories: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
	}
}

func categoryView(c models.Category) views.AdminCategoryResponse {
	return views.AdminCategoryResponse{ID: c.ID, Name: c.Name, ProductCount: c.ProductCount}
}
