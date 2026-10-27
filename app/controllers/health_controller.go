package controllers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// healthStore is the slice of the store the readiness check uses, declared
// here beside its user. *adapter.Store satisfies it.
type healthStore interface {
	Ping(ctx context.Context) error
}

// readyTimeout bounds the database check. A probe that waits longer than a
// proxy's own timeout reports nothing, so it answers "unavailable" instead.
const readyTimeout = 2 * time.Second

// HealthController answers the readiness probe. Liveness (/healthz) is in
// routes.go and touches nothing on purpose: "the process is up". Readiness
// asks whether this instance can do its job right now, which is what Kamal's
// proxy waits for before it moves traffic to a new container, so a release
// that cannot reach its database never takes over from the one that can.
type HealthController struct {
	Store healthStore
	Log   *slog.Logger
}

// NewHealthController builds the controller.
func NewHealthController(store healthStore, log *slog.Logger) *HealthController {
	return &HealthController{Store: store, Log: log}
}

// SetupRoutes mounts the probe at the root, unauthenticated, next to /healthz.
func (ctl *HealthController) SetupRoutes(router *gin.RouterGroup) {
	router.GET("/readyz", ctl.Readyz)
}

// Readyz handles GET /readyz: 200 when the database answers, 503 when it
// does not. The body names the checks and never the cause: a probe is
// public, and "connection refused to 10.0.3.7:5432" is not for the public.
// The cause is in the log, with the request id.
func (ctl *HealthController) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), readyTimeout)
	defer cancel()

	status, db := http.StatusOK, "ok"
	if err := ctl.Store.Ping(ctx); err != nil {
		ctl.Log.WarnContext(c.Request.Context(), "readiness: database check failed",
			"err", err, "request_id", c.GetString("request_id"))
		status, db = http.StatusServiceUnavailable, "error"
	}
	overall := "ok"
	if status != http.StatusOK {
		overall = "unavailable"
	}
	c.JSON(status, gin.H{"status": overall, "checks": gin.H{"database": db}})
}
