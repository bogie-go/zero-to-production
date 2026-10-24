package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/middlewares"
)

// SetupRoutes is the whole routing table, in one readable place: the one
// Rails convention that translates without compromise. The answer to "what
// does this service expose" is this file, and "how is this endpoint
// authenticated" is the group it sits in.
func (s *Server) SetupRoutes() {
	r := s.Router

	r.Use(
		middlewares.RequestID(),
		middlewares.Recover(s.Log),
		middlewares.Logger(s.Log),
	)

	// Unauthenticated, no side effects: what a load balancer polls.
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	// Ready to serve: asks the database. See HealthController.
	s.Health.SetupRoutes(&r.RouterGroup)

	// Each controller mounts its own routes under a group it names; a
	// namespace (`bogie g controller admin/reports`) mounts its group and
	// the controllers in it, with the middleware they share, in its own
	// routes.go. `bogie g controller` inserts one line above the marker; by
	// hand it is the same one line:
	//
	//	s.Posts.SetupRoutes(&r.RouterGroup)
	//
	s.Products.SetupRoutes(&r.RouterGroup)
	s.GuestSessions.SetupRoutes(&r.RouterGroup)
	s.Users.SetupRoutes(&r.RouterGroup)
	s.Sessions.SetupRoutes(&r.RouterGroup)
	s.Cart.SetupRoutes(s.WithUser(&r.RouterGroup))
	s.CartItems.SetupRoutes(s.WithUser(&r.RouterGroup))
	s.Orders.SetupRoutes(s.WithUser(&r.RouterGroup))
	s.Admin.SetupRoutes(&r.RouterGroup)
	s.Me.SetupRoutes(s.WithUser(&r.RouterGroup))
	s.Categories.SetupRoutes(&r.RouterGroup)
	// bogie:routes

	// Every request nothing above claims, in order: a file in public/,
	// then (development only) the welcome page at /, then public/404.html.
	// See app/controllers/public.go and welcome.go.
	r.NoRoute(s.staticFile, s.welcome, s.notFound)
}
