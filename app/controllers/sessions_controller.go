package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ecommerce/app/models"
	"ecommerce/app/services/password"
	"ecommerce/app/views"
)

// sessionsStore is the slice of the store this controller uses, declared
// next to its caller and holding only the methods the actions
// call.
type sessionsStore interface {
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
}

// cartMerger folds the guest cart a request carries into an account's cart;
// services/cartmerge.Merger is the real one, shared with signing up.
type cartMerger interface {
	FromAuthorization(ctx context.Context, authorization, newSubject string) error
}

// signInAlerter tells the account's owner someone signed in. The real one
// enqueues an email job (app/application.go); it is declared here so this
// controller knows nothing about jobs.
type signInAlerter interface {
	SignedIn(ctx context.Context, signIn models.SignIn) error
}

// SessionsController is the HTTP surface of logging in.
type SessionsController struct {
	Store  sessionsStore
	Tokens userTokenMaker
	Merger cartMerger
	Alerts signInAlerter
	Log    *slog.Logger
}

// NewSessionsController builds the controller.
func NewSessionsController(store sessionsStore, tokens userTokenMaker, merger cartMerger, alerts signInAlerter, log *slog.Logger) *SessionsController {
	return &SessionsController{Store: store, Tokens: tokens, Merger: merger, Alerts: alerts, Log: log}
}

// SetupRoutes mounts the controller. Unauthenticated, like guest_sessions
// and users: logging in is how a caller gets a token, not something a token
// guards — though a caller may already be carrying a guest one, which
// Create reads to merge that guest's cart in.
func (ctl *SessionsController) SetupRoutes(router *gin.RouterGroup) {
	router.POST("/sessions", ctl.Create)
}

// Create handles POST /sessions: check a password, mint a token for the
// account it belongs to, fold whatever was in the caller's guest cart into
// that account's cart, and queue the "new sign-in" email.
//
// A wrong email and a wrong password answer identically — a caller can't
// tell "no such account" from "wrong password" apart.
func (ctl *SessionsController) Create(c *gin.Context) {
	var req views.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	user, err := ctl.Store.GetUserByEmail(c.Request.Context(), req.Email)
	if errors.Is(err, models.ErrNotFound) {
		// Take the time a real check takes, so an unknown email is not
		// answered faster than a wrong password (see password.DummyCheck).
		password.DummyCheck(req.Password)
	}
	if err != nil || !password.Check(user.PasswordDigest, req.Password) {
		if err != nil && !errors.Is(err, models.ErrNotFound) {
			ctl.fail(c, err)
			return
		}
		c.JSON(http.StatusUnauthorized, views.ErrorMessage("email or password is incorrect"))
		return
	}

	subject := strconv.FormatInt(user.ID, 10)
	if err := ctl.Merger.FromAuthorization(c.Request.Context(), c.GetHeader("Authorization"), subject); err != nil {
		ctl.fail(c, err)
		return
	}

	signed, err := ctl.Tokens.Mint(subject, userTokenTTL)
	if err != nil {
		ctl.fail(c, err)
		return
	}

	// Tell the owner, after the login has succeeded. Best effort: an alert
	// that cannot be queued is logged, never a reason to refuse a correct
	// password.
	if err := ctl.Alerts.SignedIn(c.Request.Context(), models.SignIn{
		UserID: user.ID, At: time.Now().UTC(), IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
	}); err != nil {
		ctl.Log.WarnContext(c.Request.Context(), "sessions: could not queue the sign-in alert", "user_id", user.ID, "err", err)
	}
	c.JSON(http.StatusOK, views.SessionResponse{Token: signed})
}

// fail maps a domain error to a status, the one place that mapping lives:
// anything unexpected is logged and answered 500 without leaking its text.
func (ctl *SessionsController) fail(c *gin.Context, err error) {
	ctl.Log.ErrorContext(c.Request.Context(), "sessions: request failed",
		"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
	c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
}
