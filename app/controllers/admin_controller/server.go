// Package admin_controller is the admin namespace: the controllers mounted under
// /admin, as Rails's "namespace :admin" and the Admin:: module. One file per
// controller; the group and its shared middleware in routes.go.
package admin_controller

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/middlewares"
	"ecommerce/app/services/token"
)

// staffStore is the slice of the store the namespace's own auth check
// uses, declared here (rule 2) rather than in any one controller, since
// WithStaff below guards the whole /admin group, not one resource.
type staffStore interface {
	GetUser(ctx context.Context, id string) (domain.User, error)
}

// staffTokens is the slice of the token service WithStaff uses.
type staffTokens interface {
	Verify(raw string) (token.Claims, error)
}

// Server holds the namespace's controllers, one NAMED field each, so adding
// one is one line here, one in app/application.go and one in routes.go.
// "bogie g controller admin/NAME" inserts all three.
type Server struct {
	Store  staffStore
	Tokens staffTokens
	Log    *slog.Logger

	Products   *ProductsController
	Categories *CategoriesController
	PriceRules *PriceRulesController
	// bogie:controllers
}

// NewServer builds the namespace. Its controllers are constructed and
// assigned in app/application.go, where every dependency is wired. store
// and tokens are the root Server's own *models.Store and Tokens() — the
// admin namespace shares both rather than minting a parallel login system;
// a staff account is a user row like any other, just with Role ==
// domain.RoleStaff (see docs/bogie-gaps.md and db/seeds/users.go).
func NewServer(store staffStore, tokens staffTokens, log *slog.Logger) *Server {
	return &Server{Store: store, Tokens: tokens, Log: log}
}

// WithStaff is the route group this whole namespace sits behind: a
// signed-in user (RequireUser) whose role is staff (RequireStaff), checked
// fresh against the store on every request. Every controller in this
// package is mounted here, in routes.go, never on the bare router.
func (s *Server) WithStaff(r *gin.RouterGroup) *gin.RouterGroup {
	return r.Group("", middlewares.RequireUser(s.Tokens), middlewares.RequireStaff(s.Store))
}
