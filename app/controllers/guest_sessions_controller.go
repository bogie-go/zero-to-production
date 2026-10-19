package controllers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ecommerce/app/domain"
	"ecommerce/app/views"
)

// guestSessionsStore is the slice of the store this controller uses,
// declared next to its caller (rule 2) and holding only the method the
// action calls.
type guestSessionsStore interface {
	FindOrCreateCartByToken(ctx context.Context, token string) (domain.Cart, error)
}

// guestTokenMaker is the one method this controller needs from a
// token.Maker (rule 2): mint a bearer token for a subject. *Server.Tokens()
// satisfies it, the same Maker RequireUser verifies against.
type guestTokenMaker interface {
	Mint(subject string, ttl time.Duration) (string, error)
}

// guestTokenTTL is how long a minted guest token is good for: long enough
// that a returning visitor's cart is still theirs weeks later, the way a
// Rails session cookie usually outlives a single visit.
const guestTokenTTL = 30 * 24 * time.Hour

// GuestSessionsController is the one place a guest's bearer token is
// minted. Every other endpoint (cart, cart_items, and later anything else
// behind WithUser) only ever requires a token already in hand; this is
// where a visitor with none gets one — the same shape a real POST
// /sessions login will have, just with no credentials to check first. The
// token's subject is an opaque id with no meaning beyond "which cart":
// nothing here claims it is a user, because it is not one.
type GuestSessionsController struct {
	Store  guestSessionsStore
	Tokens guestTokenMaker
	Log    *slog.Logger
}

// NewGuestSessionsController builds the controller.
func NewGuestSessionsController(store guestSessionsStore, tokens guestTokenMaker, log *slog.Logger) *GuestSessionsController {
	return &GuestSessionsController{Store: store, Tokens: tokens, Log: log}
}

// SetupRoutes mounts the controller under its own prefix, unguarded: a
// bearer token is what this endpoint hands out, so it cannot also require
// one.
func (ctl *GuestSessionsController) SetupRoutes(router *gin.RouterGroup) {
	router.POST("/guest_sessions", ctl.Create)
}

// Create handles POST /guest_sessions: mint a bearer token for a fresh
// subject, create the empty cart it names, and hand the token back. No
// body to bind — a guest session needs nothing from the caller.
func (ctl *GuestSessionsController) Create(c *gin.Context) {
	subject := uuid.NewString()
	if _, err := ctl.Store.FindOrCreateCartByToken(c.Request.Context(), subject); err != nil {
		ctl.fail(c, err)
		return
	}
	signed, err := ctl.Tokens.Mint(subject, guestTokenTTL)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, views.GuestSessionResponse{Token: signed})
}

// fail logs and answers 500 without leaking its text: nothing here is a
// caller mistake, so there is no 404 or 422 case to map.
func (ctl *GuestSessionsController) fail(c *gin.Context, err error) {
	ctl.Log.ErrorContext(c.Request.Context(), "guest_sessions: request failed",
		"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
	c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
}
