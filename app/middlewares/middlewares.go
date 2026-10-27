// Package middlewares holds the cross-cutting request filters, declared once in
// routes.go and never scattered through handlers.
package middlewares

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ecommerce/public"
)

// RequestIDHeader is echoed back so a client can correlate a report with a log
// line.
const RequestIDHeader = "X-Request-Id"

// RequestID assigns an id to every request and puts it in the context, so it
// reaches the logger and, later, the trace.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// Logger records one structured line per request, at a level that says what
// to do about it: a server error (5xx) is an error, everything else is info,
// and the health probes, which fire every few seconds from every proxy and
// monitor, are debug so they do not drown the lines that matter.
//
// What is logged is chosen to be safe to ship to a log service: the route
// template and never the URL (a query string can carry an email or a token),
// the status, the time taken, the size, the request id that also appears in
// the response header and in every error line, the caller's address, and who
// they are signed in as (the token's subject: a user id or a guest id), never
// a header or a body.
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		// The route template (/posts/:id) groups a route's lines; a request
		// that matched no template, / included, logs its own path.
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case path == "/healthz" || path == "/readyz":
			level = slog.LevelDebug
		}
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("took", time.Since(start)),
			slog.Int("bytes", c.Writer.Size()),
			slog.String("request_id", c.GetString("request_id")),
			slog.String("ip", c.ClientIP()),
		}
		if who := Subject(c); who != "" {
			attrs = append(attrs, slog.String("subject", who))
		}
		log.LogAttrs(c.Request.Context(), level, "request", attrs...)
	}
}

// Recover turns a panic into public/500.html and a log line, rather than a
// dead process.
func Recover(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorContext(c.Request.Context(), "panic recovered",
					"panic", r, "path", c.FullPath(), "request_id", c.GetString("request_id"))
				c.Abort()
				c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", public.InternalServerError)
			}
		}()
		c.Next()
	}
}
