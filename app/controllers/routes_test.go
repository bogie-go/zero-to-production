package controllers_test

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers"
	"ecommerce/config"
	"ecommerce/public"
)

// testServer builds a root server and mounts its routes. Every controller
// and namespace field is given an empty value first, by reflection so this
// test needs no edit when `bogie g controller` adds one: mounting only binds
// method values and never calls a handler, and app/application.go is what
// builds the real ones. `bogie doctor` checks that every controller is both
// constructed and mounted.
func testServer(env string) *controllers.Server {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := controllers.NewServer(config.Config{Env: env, Server: config.ServerConfig{Addr: ":0", LogLevel: "info"}}, log)
	fields := reflect.ValueOf(s).Elem()
	for i := range fields.NumField() {
		if f := fields.Field(i); f.Kind() == reflect.Pointer && f.IsNil() && f.CanSet() {
			f.Set(reflect.New(f.Type().Elem()))
		}
	}
	s.SetupRoutes()
	return s
}

func get(s *controllers.Server, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

var _ = Describe("the server's routes", func() {
	Describe("GET /healthz", func() {
		It("answers ok through the middleware chain", func() {
			rec := get(testServer(config.EnvTest), "/healthz")

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(Equal("ok"))
			Expect(rec.Header().Get("X-Request-Id")).NotTo(BeEmpty(), "the middleware chain is not mounted")
		})
	})

	// The welcome page greets a developer at / and nobody else: development
	// only, GET / only. Exercised directly here, bypassing staticFile, since
	// public/index.html now answers / in every environment before welcome ever
	// gets a turn — the same way a real route at / retires it.
	Describe("the welcome page", func() {
		var s *controllers.Server
		call := func() *httptest.ResponseRecorder {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			s.Welcome(c)
			return rec
		}

		Context("in development", func() {
			BeforeEach(func() {
				s = testServer(config.EnvDevelopment)
			})

			It("greets", func() {
				Expect(call().Body.String()).To(ContainSubstring("is running"))
			})

			It("leaves an unknown path a 404: nothing in public/ answers it either", func() {
				Expect(get(s, "/nothing-here").Code).To(Equal(http.StatusNotFound))
			})

			Context("with a real route at /", func() {
				BeforeEach(func() {
					s.Router.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "mine") })
				})

				It("replaces the welcome page", func() {
					Expect(get(s, "/").Body.String()).To(Equal("mine"))
				})
			})
		})

		Context("in production", func() {
			BeforeEach(func() {
				s = testServer(config.EnvProduction)
			})

			It("writes nothing", func() {
				Expect(call().Body.Len()).To(BeZero())
			})
		})
	})

	// public/index.html answers / before the welcome page gets a turn, in every
	// environment: there is no dev-only exception once the app has its own page.
	DescribeTable("GET / is answered by public/index.html, not the welcome page",
		func(env string) {
			rec := get(testServer(env), "/")

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(ContainSubstring("<title>Ecommerce</title>"))
			Expect(rec.Body.String()).NotTo(ContainSubstring("is running"))
		},
		Entry("in development", config.EnvDevelopment),
		Entry("in production", config.EnvProduction),
	)

	// public/index.html, once the app has one, answers / in every environment,
	// ahead of the welcome page and the 404.
	DescribeTable("GET / serves public/index.html verbatim",
		func(env string) {
			want, err := fs.ReadFile(public.Files, "index.html")
			if err != nil {
				Skip("no public/index.html: the welcome page answers / in development")
			}

			rec := get(testServer(env), "/")

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(Equal(string(want)))
		},
		Entry("in development", config.EnvDevelopment),
		Entry("in production", config.EnvProduction),
	)

	Describe("static files", func() {
		// public/ is tried before the welcome page and the final 404: a request for
		// a file that is actually there is served as itself, status 200, in every
		// environment.
		It("answers before the welcome page", func() {
			rec := get(testServer(config.EnvDevelopment), "/500.html")

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(ContainSubstring("500 Internal Server Error"))
			Expect(rec.Body.String()).NotTo(ContainSubstring("404 Not Found"), "welcome and notFound must not also run")
		})

		// The images the error pages reference are embedded too, not just the HTML:
		// public/'s go:embed has to list "images" alongside "*.html" or these 404.
		DescribeTable("serves embedded images",
			func(path string) {
				Expect(get(testServer(config.EnvProduction), path).Code).To(Equal(http.StatusOK))
			},
			Entry("/images/bogie-404.webp", "/images/bogie-404.webp"),
			Entry("/images/bogie-500.webp", "/images/bogie-500.webp"),
			Entry("/images/products/wireless-mouse.webp", "/images/products/wireless-mouse.webp"),
		)

		// A folder in public/ answers with its index.html, redirecting to the
		// trailing slash first so the page's relative links resolve; a folder with
		// no index.html is simply not found. /admin/index.html itself is the
		// page the staff gate lives in (the API behind it is guarded separately).
		Describe("folders", func() {
			var s *controllers.Server

			Context("in production", func() {
				BeforeEach(func() {
					s = testServer(config.EnvProduction)
				})

				It("redirects to the trailing slash", func() {
					rec := get(s, "/admin")

					Expect(rec.Code).To(Equal(http.StatusMovedPermanently))
					Expect(rec.Header().Get("Location")).To(Equal("/admin/"))
				})

				It("serves the folder's index.html", func() {
					rec := get(s, "/admin/")

					Expect(rec.Code).To(Equal(http.StatusOK))
					Expect(rec.Body.String()).To(ContainSubstring("<title>Admin</title>"))
				})

				It("is not found when there is no index.html", func() {
					Expect(get(s, "/images").Code).To(Equal(http.StatusNotFound))
				})
			})
		})
	})

	// Nothing above answering, in any environment, is public/404.html: the last
	// of the three no-route handlers.
	DescribeTable("an unmatched route serves the 404 page",
		func(env string) {
			rec := get(testServer(env), "/nothing-here")

			Expect(rec.Code).To(Equal(http.StatusNotFound))
			Expect(rec.Body.String()).To(ContainSubstring("404 Not Found"))
		},
		Entry("in development", config.EnvDevelopment),
		Entry("in production", config.EnvProduction),
	)

	// A panicking handler gets public/500.html from middlewares.Recover, not a
	// bare status with no body.
	Context("with a handler that panics", func() {
		var s *controllers.Server

		BeforeEach(func() {
			s = testServer(config.EnvTest)
			s.Router.GET("/boom", func(c *gin.Context) { panic("boom") })
		})

		It("serves the 500 page for a recovered panic", func() {
			rec := get(s, "/boom")

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			Expect(rec.Body.String()).To(ContainSubstring("500 Internal Server Error"))
		})
	})
})
