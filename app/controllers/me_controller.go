package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/middlewares"
	"ecommerce/app/models"
	"ecommerce/app/views"
)

// meStore is the slice of the store this controller uses, declared next to
// its caller (rule 2).
type meStore interface {
	GetUser(ctx context.Context, id string) (models.User, error)
}

// MeController answers one question for the UI: who is the bearer token
// for? A guest's token is as valid as a real login's, and the UI cannot tell
// them apart (it is an opaque token), so a page that needs a real account
// asks here instead of decoding the token itself. There is no singular
// resource in Bogie yet (docs/bogie-gaps.md), so Index stands in for show,
// as it does for the cart.
type MeController struct {
	Store meStore
	Log   *slog.Logger
}

// NewMeController builds the controller.
func NewMeController(store meStore, log *slog.Logger) *MeController {
	return &MeController{Store: store, Log: log}
}

// SetupRoutes mounts the controller; routes.go puts it behind s.WithUser.
func (ctl *MeController) SetupRoutes(router *gin.RouterGroup) {
	router.GET("/me", ctl.Index)
}

// Index handles GET /me: the signed-in account, or 403 for a guest token
// (a bare uuid subject with no users row), the same answer and message
// OrdersController gives a guest.
func (ctl *MeController) Index(c *gin.Context) {
	user, err := ctl.Store.GetUser(c.Request.Context(), middlewares.Subject(c))
	switch {
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusForbidden, views.ErrorMessage("create an account to continue"))
		return
	case err != nil:
		ctl.Log.ErrorContext(c.Request.Context(), "me: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
		return
	}
	c.JSON(http.StatusOK, views.MeResponse{ID: user.ID, Email: user.Email, Role: user.Role})
}
