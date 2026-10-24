package admin_controller

import "github.com/gin-gonic/gin"

// SetupRoutes mounts every controller in the namespace under /admin.
// Middleware the whole namespace shares, an auth check or a shared secret,
// goes on the group here, so the exposure boundary is visible in one place.
func (s *Server) SetupRoutes(router *gin.RouterGroup) {
	admin := s.WithStaff(router.Group("/admin"))
	s.Products.SetupRoutes(admin)
	s.Categories.SetupRoutes(admin)
	s.PriceRules.SetupRoutes(admin)
	s.Ops.SetupRoutes(admin)
	// bogie:routes
}
