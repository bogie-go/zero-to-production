package controllers

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"ecommerce/public"
)

// staticFile is the first of three no-route handlers mounted in
// routes.go (`r.NoRoute(s.staticFile, s.welcome, s.notFound)`): a request
// that matched no route is tried against public/ before anything else, the
// way a web server in front of an app serves public/ ahead of it. A
// file found there is served as itself, whatever it is, and c.Abort() stops
// welcome and notFound from also running: in Gin, returning from a handler
// does not by itself stop the chain, only Abort does. Anything not found is
// left to the next handler by simply returning.
func (s *Server) staticFile(c *gin.Context) {
	name := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
	if name == "" || name == "." {
		name = "index.html"
	}
	info, err := fs.Stat(public.Files, name)
	if err != nil {
		return
	}
	if info.IsDir() {
		// A folder answers with its index.html, as a web server would:
		// /admin/ is public/admin/index.html. Without the
		// trailing slash, redirect, so the page's relative links resolve.
		if _, err := fs.Stat(public.Files, name+"/index.html"); err != nil {
			return
		}
		if !strings.HasSuffix(c.Request.URL.Path, "/") {
			c.Redirect(http.StatusMovedPermanently, c.Request.URL.Path+"/")
			c.Abort()
			return
		}
		name += "/index.html"
	}
	http.ServeFileFS(c.Writer, c.Request, public.Files, name)
	c.Abort()
}

// notFound is the last of the three: a request neither a route, public/,
// nor (in development) the welcome page answered gets public/404.html.
func (s *Server) notFound(c *gin.Context) {
	c.Data(http.StatusNotFound, "text/html; charset=utf-8", public.NotFound)
}
