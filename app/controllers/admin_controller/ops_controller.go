package admin_controller

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/models"
	"ecommerce/app/views"
)

// opsStore is the slice of the store this controller uses, declared next to
// its caller (rule 2). *db.Store satisfies it.
type opsStore interface {
	Ops(ctx context.Context) (models.Ops, error)
}

// OpsController shows an operator whether the system is healthy. Only
// reachable under /admin, behind the namespace's WithStaff.
type OpsController struct {
	Store   opsStore
	Log     *slog.Logger
	Version string
	Env     string
}

// NewOpsController builds the controller. version is the running release and
// env the process's environment, both read from the config by the caller.
func NewOpsController(store opsStore, log *slog.Logger, version, env string) *OpsController {
	return &OpsController{Store: store, Log: log, Version: version, Env: env}
}

// SetupRoutes mounts the controller under its own prefix.
func (ctl *OpsController) SetupRoutes(router *gin.RouterGroup) {
	router.GET("/ops", ctl.Index)
}

// Index handles GET /admin/ops.
func (ctl *OpsController) Index(c *gin.Context) {
	ops, err := ctl.Store.Ops(c.Request.Context())
	if err != nil {
		ctl.Log.ErrorContext(c.Request.Context(), "ops: request failed",
			"err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
		return
	}
	c.JSON(http.StatusOK, views.OpsResponse{
		Version:              ctl.Version,
		Env:                  ctl.Env,
		SchemaVersion:        ops.SchemaVersion,
		Jobs:                 ops.Jobs,
		OldestWaitingSeconds: ops.OldestWaiting.Seconds(),
	})
}
