package middlewares_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/middlewares"
)

var _ = Describe("Logger", func() {
	var (
		out    *bytes.Buffer
		router *gin.Engine
		level  slog.Level
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		out = &bytes.Buffer{}
		level = slog.LevelDebug
	})

	JustBeforeEach(func() {
		log := slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: level}))
		router = gin.New()
		router.Use(middlewares.RequestID(), middlewares.Logger(log))
		router.GET("/posts/:id", func(c *gin.Context) { c.String(http.StatusOK, "post") })
		router.GET("/boom", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })
		router.GET("/missing", func(c *gin.Context) { c.Status(http.StatusNotFound) })
		router.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
		router.GET("/readyz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	})

	// lines is every record the logger wrote, decoded.
	lines := func() []map[string]any {
		var all []map[string]any
		for _, raw := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			if raw == "" {
				continue
			}
			var line map[string]any
			Expect(json.Unmarshal([]byte(raw), &line)).To(Succeed())
			all = append(all, line)
		}
		return all
	}
	get := func(path string, with ...func(*http.Request)) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		for _, f := range with {
			f(req)
		}
		router.ServeHTTP(httptest.NewRecorder(), req)
	}

	Context("with an ordinary request", func() {
		JustBeforeEach(func() {
			get("/posts/42?email=ada@example.com&token=s3cret", func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer very-secret-token")
				r.Header.Set("X-Request-Id", "req-123")
			})
		})

		It("writes one info line with what happened", func() {
			Expect(lines()).To(HaveLen(1))
			line := lines()[0]
			Expect(line).To(HaveKeyWithValue("level", "INFO"))
			Expect(line).To(HaveKeyWithValue("method", "GET"))
			Expect(line).To(HaveKeyWithValue("status", BeEquivalentTo(200)))
			Expect(line).To(HaveKeyWithValue("request_id", "req-123"))
			Expect(line).To(HaveKey("took"))
			Expect(line).To(HaveKey("bytes"))
			Expect(line).To(HaveKey("ip"))
		})

		It("names the route, not the URL", func() {
			Expect(lines()[0]).To(HaveKeyWithValue("path", "/posts/:id"))
		})

		It("never writes a query string or a header", func() {
			for _, secret := range []string{"ada@example.com", "s3cret", "very-secret-token", "Bearer"} {
				Expect(out.String()).NotTo(ContainSubstring(secret))
			}
		})
	})

	Context("when the server fails", func() {
		JustBeforeEach(func() { get("/boom") })

		It("is an error line", func() {
			Expect(lines()[0]).To(HaveKeyWithValue("level", "ERROR"))
		})
	})

	Context("when the caller is at fault", func() {
		JustBeforeEach(func() { get("/missing") })

		It("is not an error: a 404 is the caller's, not ours", func() {
			Expect(lines()[0]).To(HaveKeyWithValue("level", "INFO"))
		})
	})

	Context("with a health probe", func() {
		JustBeforeEach(func() {
			get("/healthz")
			get("/readyz")
		})

		It("is debug, so it can be silenced", func() {
			for _, line := range lines() {
				Expect(line).To(HaveKeyWithValue("level", "DEBUG"))
			}
		})

		Context("at the usual info level", func() {
			BeforeEach(func() {
				level = slog.LevelInfo
			})

			It("writes nothing", func() {
				Expect(lines()).To(BeEmpty())
			})
		})
	})
})
