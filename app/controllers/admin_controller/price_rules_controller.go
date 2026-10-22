package admin_controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/pricing"
	"ecommerce/app/views"
)

// priceRulesStore is the slice of the store this controller uses, declared
// next to its caller (rule 2). *models.Store satisfies it.
type priceRulesStore interface {
	ListAllPriceRules(ctx context.Context) ([]domain.PriceRule, error)
	GetPriceRule(ctx context.Context, id string) (domain.PriceRule, error)
	CreatePriceRule(ctx context.Context, p domain.PriceRule) (domain.PriceRule, error)
	UpdatePriceRule(ctx context.Context, p domain.PriceRule) (domain.PriceRule, error)
	DeletePriceRule(ctx context.Context, id string) error
	SetPriceRuleActive(ctx context.Context, id string, active bool) (domain.PriceRule, error)
}

// PriceRulesController is the admin surface of price rules: the engine's
// rules, switched on and off and edited without a deploy. Reachable only
// under /admin, behind the namespace's WithStaff. A change takes effect on
// the next cart request: the cart reads the active rules every time.
type PriceRulesController struct {
	Store priceRulesStore
	Log   *slog.Logger
	// Now is the clock, so a rule's status ("scheduled", "ended") can be
	// tested; nil means time.Now.
	Now func() time.Time
}

// NewPriceRulesController builds the controller.
func NewPriceRulesController(store priceRulesStore, log *slog.Logger) *PriceRulesController {
	return &PriceRulesController{Store: store, Log: log}
}

func (ctl *PriceRulesController) now() time.Time {
	if ctl.Now != nil {
		return ctl.Now()
	}
	return time.Now()
}

// SetupRoutes mounts the controller under its own prefix. kinds is mounted
// before /:id so it is not read as an id.
func (ctl *PriceRulesController) SetupRoutes(router *gin.RouterGroup) {
	rules := router.Group("/price_rules")
	rules.GET("", ctl.Index)
	rules.GET("/kinds", ctl.Kinds)
	rules.GET("/:id", ctl.Show)
	rules.POST("", ctl.Create)
	rules.PATCH("/:id", ctl.Update)
	rules.DELETE("/:id", ctl.Destroy)
	rules.POST("/:id/activate", ctl.Activate)
	rules.POST("/:id/deactivate", ctl.Deactivate)
}

// Index handles GET /admin/price_rules, in the order the engine runs them.
func (ctl *PriceRulesController) Index(c *gin.Context) {
	rules, err := ctl.Store.ListAllPriceRules(c.Request.Context())
	if err != nil {
		ctl.fail(c, err)
		return
	}
	out := views.PriceRulesResponse{PriceRules: make([]views.PriceRuleResponse, 0, len(rules))}
	for _, r := range rules {
		out.PriceRules = append(out.PriceRules, ctl.view(r))
	}
	c.JSON(http.StatusOK, out)
}

// Kinds handles GET /admin/price_rules/kinds: every kind of rule and the
// fields it takes, which is what the form is built from. A new kind registers
// itself in app/pricing and shows up here with no change to this file.
func (ctl *PriceRulesController) Kinds(c *gin.Context) {
	specs := pricing.Specs()
	out := views.KindsResponse{Kinds: make([]views.KindResponse, 0, len(specs))}
	for _, s := range specs {
		k := views.KindResponse{Kind: s.Kind, Label: s.Label, Help: s.Help, Note: s.Note, Fields: make([]views.FieldResponse, 0, len(s.Fields))}
		for _, f := range s.Fields {
			k.Fields = append(k.Fields, views.FieldResponse{Name: f.Name, Label: f.Label, Type: f.Type, Help: f.Help})
		}
		out.Kinds = append(out.Kinds, k)
	}
	c.JSON(http.StatusOK, out)
}

// Show handles GET /admin/price_rules/:id.
func (ctl *PriceRulesController) Show(c *gin.Context) {
	rule, err := ctl.Store.GetPriceRule(c.Request.Context(), c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, ctl.view(rule))
}

// Create handles POST /admin/price_rules.
func (ctl *PriceRulesController) Create(c *gin.Context) {
	var req views.PriceRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	rule, err := ruleFromRequest(domain.PriceRule{}, req)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	created, err := ctl.Store.CreatePriceRule(c.Request.Context(), rule)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, ctl.view(created))
}

// Update handles PATCH /admin/price_rules/:id. Every field is sent, as for
// every other write here; the id and timestamps carry over.
func (ctl *PriceRulesController) Update(c *gin.Context) {
	var req views.PriceRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	existing, err := ctl.Store.GetPriceRule(c.Request.Context(), c.Param("id"))
	if err != nil {
		ctl.fail(c, err)
		return
	}
	rule, err := ruleFromRequest(existing, req)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	updated, err := ctl.Store.UpdatePriceRule(c.Request.Context(), rule)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, ctl.view(updated))
}

// Destroy handles DELETE /admin/price_rules/:id. An order keeps the totals it
// was placed with, not a reference to the rule, so deleting one never changes
// a receipt.
func (ctl *PriceRulesController) Destroy(c *gin.Context) {
	if err := ctl.Store.DeletePriceRule(c.Request.Context(), c.Param("id")); err != nil {
		ctl.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Activate handles POST /admin/price_rules/:id/activate: the rule applies from
// the next cart request, within its dates.
func (ctl *PriceRulesController) Activate(c *gin.Context) { ctl.setActive(c, true) }

// Deactivate handles POST /admin/price_rules/:id/deactivate: the rule stops
// applying at once and is kept, to switch back on.
func (ctl *PriceRulesController) Deactivate(c *gin.Context) { ctl.setActive(c, false) }

func (ctl *PriceRulesController) setActive(c *gin.Context, active bool) {
	rule, err := ctl.Store.SetPriceRuleActive(c.Request.Context(), c.Param("id"), active)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, ctl.view(rule))
}

// ruleFromRequest builds the rule to save. The properties are validated by the
// kind's own decoder (pricing.Load), and what is stored is that decoded rule
// written back out, so unknown keys in the request never reach the database
// and the JSON is always in the shape the engine reads.
func ruleFromRequest(base domain.PriceRule, req views.PriceRuleRequest) (domain.PriceRule, error) {
	props := req.Properties
	if len(props) == 0 {
		props = json.RawMessage(`{}`)
	}
	if props[0] != '{' {
		return domain.PriceRule{}, fmt.Errorf("%w: properties must be an object", domain.ErrInvalid)
	}
	loaded, err := pricing.Load(req.Kind, props)
	if err != nil {
		return domain.PriceRule{}, err
	}
	canonical, err := json.Marshal(loaded)
	if err != nil {
		return domain.PriceRule{}, err
	}
	base.Name, base.Kind, base.Properties = req.Name, req.Kind, canonical
	base.Active, base.Priority, base.StartsAt, base.EndsAt = req.Active, req.Priority, req.StartsAt, req.EndsAt
	return base, nil
}

// status says where a rule stands at now.
func status(r domain.PriceRule, now time.Time) string {
	switch {
	case !r.Active:
		return "off"
	case r.StartsAt != nil && now.Before(*r.StartsAt):
		return "scheduled"
	case r.EndsAt != nil && !now.Before(*r.EndsAt):
		return "ended"
	}
	return "active"
}

func (ctl *PriceRulesController) view(r domain.PriceRule) views.PriceRuleResponse {
	label := r.Kind
	for _, s := range pricing.Specs() {
		if s.Kind == r.Kind {
			label = s.Label
		}
	}
	return views.PriceRuleResponse{
		ID: r.ID, Name: r.Name, Kind: r.Kind, KindLabel: label, Summary: pricing.Summarize(r.Kind, r.Properties),
		Properties: r.Properties, Active: r.Active, Priority: r.Priority, StartsAt: r.StartsAt, EndsAt: r.EndsAt,
		Status: status(r, ctl.now()),
	}
}

// fail maps a domain error to a status, the one place that mapping lives: not
// found is 404, a validation failure (a bad kind, properties the kind
// refuses, a blank name, dates out of order) is 422, and anything else is
// logged and answered 500 without leaking its text.
func (ctl *PriceRulesController) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, views.ErrorMessage("no such price rule"))
	case errors.Is(err, domain.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, views.Error(err))
	default:
		ctl.Log.ErrorContext(c.Request.Context(), "admin/price_rules: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
	}
}
