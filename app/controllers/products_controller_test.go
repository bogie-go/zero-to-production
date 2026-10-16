package controllers

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Every action is mounted: a route that answers 404 was never registered.
func TestProductsRoutesAreMounted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	NewProductsController(log).SetupRoutes(&router.RouterGroup)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/products"},
		{http.MethodGet, "/products/1"},
	}
	for _, r := range routes {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(r.method, r.path, nil))
		if rec.Code == http.StatusNotFound {
			t.Errorf("%s %s is not mounted", r.method, r.path)
		}
	}
}
