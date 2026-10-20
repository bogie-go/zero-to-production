package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/services/password"
	"ecommerce/app/views"
)

// usersStore is the slice of the store this controller uses, declared next
// to its caller (rule 2) and holding only the methods the actions call.
type usersStore interface {
	CreateUser(ctx context.Context, u domain.User) (domain.User, error)
}

// userTokenMaker is the slice of the token service this controller uses.
type userTokenMaker interface {
	Mint(subject string, ttl time.Duration) (string, error)
}

// userTokenTTL matches guestTokenTTL: a signed-in user's token and a
// guest's are the same kind of thing, same lifetime, until there's a real
// reason to tell them apart (a "remember me" option, say).
const userTokenTTL = 30 * 24 * time.Hour

// UsersController is the HTTP surface of signing up. There is deliberately
// no Index/Show/Update/Destroy: a reader manages their own account through
// Create only, for now.
type UsersController struct {
	Store  usersStore
	Tokens userTokenMaker
	Log    *slog.Logger
}

// NewUsersController builds the controller.
func NewUsersController(store usersStore, tokens userTokenMaker, log *slog.Logger) *UsersController {
	return &UsersController{Store: store, Tokens: tokens, Log: log}
}

// SetupRoutes mounts the controller. Unauthenticated, like guest_sessions:
// signing up is how a caller gets a token, not something a token guards.
func (ctl *UsersController) SetupRoutes(router *gin.RouterGroup) {
	router.POST("/users", ctl.Create)
}

// Create handles POST /users: sign up, and sign the new account in
// immediately — the same bearer token a login mints, so the client never
// needs a separate request right after registering.
func (ctl *UsersController) Create(c *gin.Context) {
	var req views.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, views.Error(err))
		return
	}
	digest, err := password.Hash(req.Password)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	user, err := ctl.Store.CreateUser(c.Request.Context(), domain.User{Email: req.Email, PasswordDigest: digest})
	if err != nil {
		ctl.fail(c, err)
		return
	}
	signed, err := ctl.Tokens.Mint(strconv.FormatInt(user.ID, 10), userTokenTTL)
	if err != nil {
		ctl.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, views.SessionResponse{Token: signed})
}

// fail maps a domain error to a status: a validation failure (bad email, a
// password that's too short, an email already taken — all domain.ErrInvalid,
// per AGENTS.md's new :uniq behavior) is 422 as in Rails, anything else is
// logged and answered 500 without leaking its text.
func (ctl *UsersController) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, views.Error(err))
	default:
		ctl.Log.ErrorContext(c.Request.Context(), "users: request failed",
			"path", c.FullPath(), "err", err, "request_id", c.GetString("request_id"))
		c.JSON(http.StatusInternalServerError, views.ErrorMessage("internal error"))
	}
}
