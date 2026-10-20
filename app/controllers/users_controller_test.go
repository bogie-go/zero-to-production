package controllers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
)

// fakeUsersStore is a fake of the store interface declared beside the
// controller: it records what the controller asked for and answers with
// what the test chose.
type fakeUsersStore struct {
	user domain.User
	err  error

	gotUser domain.User
}

func (f *fakeUsersStore) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	f.gotUser = u
	if f.err != nil {
		return domain.User{}, f.err
	}
	return f.user, nil
}

func usersRequest(t *testing.T, store *fakeUsersStore, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewUsersController(store, testTokens, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(&router.RouterGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func TestUsersRoutesAreMounted(t *testing.T) {
	rec := usersRequest(t, &fakeUsersStore{}, `{"email":"a@example.com","password":"at-least-8"}`)
	if rec.Code == http.StatusNotFound {
		t.Error("POST /users is not mounted")
	}
}

// A signup hashes the password (never stores it plain), creates the
// account, and mints a bearer token immediately — no separate login needed
// right after registering.
func TestUsersCreateHashesPasswordAndMintsAToken(t *testing.T) {
	store := &fakeUsersStore{user: domain.User{ID: 7, Email: "a@example.com"}}
	rec := usersRequest(t, store, `{"email":"a@example.com","password":"at-least-8"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"token":"`) {
		t.Errorf("body = %s, want a token", rec.Body)
	}
	if store.gotUser.Email != "a@example.com" {
		t.Errorf("store got email %q", store.gotUser.Email)
	}
	if store.gotUser.PasswordDigest == "" || store.gotUser.PasswordDigest == "at-least-8" {
		t.Errorf("store got password digest %q, want a bcrypt digest, not the plain password", store.gotUser.PasswordDigest)
	}
}

// A password too short to hash never reaches the store at all.
func TestUsersCreateShortPasswordIs422(t *testing.T) {
	store := &fakeUsersStore{}
	rec := usersRequest(t, store, `{"email":"a@example.com","password":"short"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422: %s", rec.Code, rec.Body)
	}
	if store.gotUser.Email != "" {
		t.Error("the store was called even though the password was invalid")
	}
}

// An email already taken (the store's :uniq violation, mapped to
// ErrInvalid) is 422, same as any other validation failure.
func TestUsersCreateDuplicateEmailIs422(t *testing.T) {
	store := &fakeUsersStore{err: errors.Join(domain.ErrInvalid, errors.New("email is already taken"))}
	rec := usersRequest(t, store, `{"email":"a@example.com","password":"at-least-8"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422: %s", rec.Code, rec.Body)
	}
}

func TestUsersCreateMalformedJSONIs400(t *testing.T) {
	rec := usersRequest(t, &fakeUsersStore{}, `{"nope":`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// An unexpected error is a 500 that says nothing about its cause.
func TestUsersCreateUnexpectedErrorIs500WithoutDetail(t *testing.T) {
	store := &fakeUsersStore{err: errors.New("connection refused to db-1")}
	rec := usersRequest(t, store, `{"email":"a@example.com","password":"at-least-8"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db-1") {
		t.Errorf("body leaks the internal error: %s", rec.Body)
	}
}
