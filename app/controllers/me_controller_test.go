package controllers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
	"ecommerce/app/middlewares"
)

type fakeMeStore struct {
	user domain.User
	err  error
}

func (f fakeMeStore) GetUser(_ context.Context, id string) (domain.User, error) {
	return f.user, f.err
}

func meRequest(t *testing.T, store fakeMeStore, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	guarded := router.Group("", middlewares.RequireUser(testTokens))
	NewMeController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(guarded)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func TestMeReturnsTheAccount(t *testing.T) {
	store := fakeMeStore{user: domain.User{ID: 3, Email: "a@example.com", Role: domain.RoleCustomer}}
	rec := meRequest(t, store, bearerFor(t, "3"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"email":"a@example.com"`) {
		t.Errorf("body = %s", rec.Body)
	}
}

func TestMeRejectsAGuestToken(t *testing.T) {
	rec := meRequest(t, fakeMeStore{err: domain.ErrNotFound}, bearerFor(t, "a-guest-uuid"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("guest: %d, want 403", rec.Code)
	}
}

func TestMeNeedsAToken(t *testing.T) {
	if rec := meRequest(t, fakeMeStore{}, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: %d, want 401", rec.Code)
	}
}
