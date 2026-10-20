package controllers

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ecommerce/config"
	"ecommerce/public"
)

// testServer builds a root server and mounts its routes. Every controller
// and namespace field is given an empty value first, by reflection so this
// test needs no edit when `bogie g controller` adds one: mounting only binds
// method values and never calls a handler, and app/application.go is what
// builds the real ones. `bogie doctor` checks that every controller is both
// constructed and mounted.
func testServer(env string) *Server {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := NewServer(config.Config{Env: env, App: config.AppConfig{Addr: ":0", LogLevel: "info"}}, log)
	fields := reflect.ValueOf(s).Elem()
	for i := range fields.NumField() {
		if f := fields.Field(i); f.Kind() == reflect.Pointer && f.IsNil() && f.CanSet() {
			f.Set(reflect.New(f.Type().Elem()))
		}
	}
	s.SetupRoutes()
	return s
}

func TestHealthz(t *testing.T) {
	s := testServer(config.EnvTest)

	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("GET /healthz = %d %q, want 200 ok", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("no X-Request-Id header: the middleware chain is not mounted")
	}
}

func get(s *Server, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// The welcome page greets a developer at / and nobody else: development
// only, GET / only. Exercised directly here, bypassing staticFile, since
// public/index.html now answers / in every environment before welcome ever
// gets a turn — the same way TestARouteAtRootReplacesTheWelcomePage shows a
// real route retiring it.
func TestWelcomeInDevelopmentOnly(t *testing.T) {
	call := func(env string) *httptest.ResponseRecorder {
		s := testServer(env)
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		s.welcome(c)
		return rec
	}

	if rec := call(config.EnvDevelopment); !strings.Contains(rec.Body.String(), "is running") {
		t.Errorf("development: welcome() wrote %q, want the welcome page", rec.Body.String())
	}
	if rec := call(config.EnvProduction); rec.Body.Len() != 0 {
		t.Errorf("production: welcome() wrote %q, want nothing", rec.Body.String())
	}

	dev := testServer(config.EnvDevelopment)
	if rec := get(dev, "/nothing-here"); rec.Code != http.StatusNotFound {
		t.Errorf("development GET /nothing-here = %d, want 404: nothing in public/ answers it either", rec.Code)
	}
}

// public/index.html answers / before the welcome page gets a turn, in every
// environment: there is no dev-only exception once the app has its own page.
func TestPublicIndexAnsweredBeforeTheWelcomePage(t *testing.T) {
	for _, env := range []string{config.EnvDevelopment, config.EnvProduction} {
		s := testServer(env)
		rec := get(s, "/")
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<title>Ecommerce</title>") ||
			strings.Contains(rec.Body.String(), "is running") {
			t.Errorf("%s GET / = %d %q, want public/index.html, not the welcome page", env, rec.Code, rec.Body.String())
		}
	}
}

// public/index.html, once the app has one, answers / in every environment,
// ahead of the welcome page and the 404.
func TestPublicIndexAnswersRoot(t *testing.T) {
	want, err := fs.ReadFile(public.Files, "index.html")
	if err != nil {
		t.Skip("no public/index.html: the welcome page answers / in development")
	}
	for _, env := range []string{config.EnvDevelopment, config.EnvProduction} {
		if rec := get(testServer(env), "/"); rec.Code != http.StatusOK || rec.Body.String() != string(want) {
			t.Errorf("%s GET / = %d %q, want 200 and public/index.html", env, rec.Code, rec.Body.String())
		}
	}
}

func TestARouteAtRootReplacesTheWelcomePage(t *testing.T) {
	s := testServer(config.EnvDevelopment)
	s.Router.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "mine") })

	if rec := get(s, "/"); rec.Body.String() != "mine" {
		t.Errorf("GET / = %q, want the app's own route", rec.Body.String())
	}
}

// public/ is tried before the welcome page and the final 404: a request for
// a file that is actually there is served as itself, status 200, in every
// environment.
func TestStaticFileAnsweredBeforeTheWelcomePage(t *testing.T) {
	s := testServer(config.EnvDevelopment)
	rec := get(s, "/500.html")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "500 Internal Server Error") ||
		strings.Contains(rec.Body.String(), "404 Not Found") {
		t.Errorf("GET /500.html = %d %q, want 200 and only the file's own content: welcome and notFound must not also run", rec.Code, rec.Body.String())
	}
}

// The images the error pages reference are embedded too, not just the HTML:
// public/'s go:embed has to list "images" alongside "*.html" or these 404.
func TestStaticFileServesImages(t *testing.T) {
	s := testServer(config.EnvProduction)
	for _, path := range []string{"/images/bogie-404.webp", "/images/bogie-500.webp", "/images/products/wireless-mouse.webp"} {
		if rec := get(s, path); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
	}
}

// Nothing above answering, in any environment, is public/404.html: the last
// of the three no-route handlers.
func TestUnmatchedRouteServesThe404Page(t *testing.T) {
	for _, env := range []string{config.EnvDevelopment, config.EnvProduction} {
		s := testServer(env)
		rec := get(s, "/nothing-here")
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "404 Not Found") {
			t.Errorf("%s GET /nothing-here = %d %q, want 404 and public/404.html", env, rec.Code, rec.Body.String())
		}
	}
}

// A panicking handler gets public/500.html from middlewares.Recover, not a
// bare status with no body.
func TestPanicRecoveredServesThe500Page(t *testing.T) {
	s := testServer(config.EnvTest)
	s.Router.GET("/boom", func(c *gin.Context) { panic("boom") })

	rec := get(s, "/boom")
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "500 Internal Server Error") {
		t.Errorf("GET /boom = %d %q, want 500 and public/500.html", rec.Code, rec.Body.String())
	}
}
