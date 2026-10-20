package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/services/password"
	"ecommerce/app/services/token"
	"ecommerce/app/views"
)

// sessionsStore is the slice of the store this controller uses, declared
// next to its caller (rule 2) and holding only the methods the actions
// call.
type sessionsStore interface {
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetCartByToken(ctx context.Context, token string) (domain.Cart, error)
	FindOrCreateCartByToken(ctx context.Context, token string) (domain.Cart, error)
	CartContents(ctx context.Context, cartID int64) ([]domain.CartLine, error)
	AddCartItem(ctx context.Context, cartID, productID int64, quantity int32) (domain.CartItem, error)
	DeleteCartAndItems(ctx context.Context, cartID int64) error
}

// sessionTokens is the slice of the token service this controller uses: it
// both mints a new token (a successful login) and verifies one presented
// with the request (an existing guest cart to merge in, if any).
type sessionTokens interface {
	userTokenMaker
	Verify(raw string) (token.Claims, error)
}

// SessionsController is the HTTP surface of logging in.
type SessionsController struct {
	Store  sessionsStore
	Tokens sessionTokens
	Log    *slog.Logger
}

// NewSessionsController builds the controller.
func NewSessionsController(store sessionsStore, tokens sessionTokens, log *slog.Logger) *SessionsController {
	return &SessionsController{Store: store, Tokens: tokens, Log: log}
}

// SetupRoutes mounts the controller. Unauthenticated, like guest_sessions
// and users: logging in is how a caller gets a token, not something a token
// guards — though a caller may already be carrying a guest one, which
// Create reads to merge that guest's cart in.
func (ctl *SessionsController) SetupRoutes(router *gin.RouterGroup) {
	router.POST("/sessions", ctl.Create)
}

// Create handles POST /sessions: check a password, mint a token for the
// account it belongs to, and fold whatever was in the caller's guest cart
// into that account's cart before answering.
//
// A wrong email and a wrong password answer identically — a caller can't
// tell "no such account" from "wrong password" apart, the same way Rails'
// authenticate doesn't either.
func (ctl *SessionsController) Create(c *gin.Context) {
	var req views.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	user, err := ctl.Store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || !password.Check(user.PasswordDigest, req.Password) {
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			ctl.fail(c, err)
			return
		}
		c.JSON(http.StatusUnauthorized, views.ErrorMessage("email or password is incorrect"))
		return
	}

	subject := strconv.FormatInt(user.ID, 10)
	if err := ctl.mergeGuestCart(c, subject); err != nil {
		ctl.fail(c, err)
		return
	}

	signed, err := ctl.Tokens.Mint(subject, userTokenTTL)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, views.SessionResponse{Token: signed})
}

// mergeGuestCart folds the cart named by whatever bearer token the request
// already carried into newSubject's cart, then removes the old one — best
// effort: a missing or invalid existing token, or no cart under it, is not
// an error, just nothing to merge. A line that can't be added (out of
// stock, say) is logged and skipped rather than failing the whole login;
// the customer is signing in regardless of what became of one guest cart
// line.
func (ctl *SessionsController) mergeGuestCart(c *gin.Context, newSubject string) error {
	ctx := c.Request.Context()
	raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !ok || raw == "" {
		return nil
	}
	claims, err := ctl.Tokens.Verify(raw)
	if err != nil || claims.Subject == newSubject {
		return nil
	}

	oldCart, err := ctl.Store.GetCartByToken(ctx, claims.Subject)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return nil
	case err != nil:
		return err
	}

	lines, err := ctl.Store.CartContents(ctx, oldCart.ID)
	if err != nil {
		return err
	}
	if len(lines) > 0 {
		newCart, err := ctl.Store.FindOrCreateCartByToken(ctx, newSubject)
		if err != nil {
			return err
		}
		for _, l := range lines {
			if _, err := ctl.Store.AddCartItem(ctx, newCart.ID, l.ProductID, l.Quantity); err != nil {
				ctl.Log.WarnContext(ctx, "sessions: could not merge a guest cart line",
					"product_id", l.ProductID, "quantity", l.Quantity, "err", err)
			}
		}
	}
	return ctl.Store.DeleteCartAndItems(ctx, oldCart.ID)
}

// fail maps a domain error to a status, the one place that mapping lives:
// anything unexpected is logged and answered 500 without leaking its text.
func (ctl *SessionsController) fail(c *gin.Context, err error) {
	ctl.Log.ErrorContext(c.Request.Context(), "sessions: request failed",
		"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
	c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
}
