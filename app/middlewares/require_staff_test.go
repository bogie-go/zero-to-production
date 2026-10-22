package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
)

type fakeStaffStore struct {
	user domain.User
	err  error
}

func (f fakeStaffStore) GetUser(_ context.Context, id string) (domain.User, error) {
	return f.user, f.err
}

func staffRequest(t *testing.T, store staffStore, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/me", RequireUser(testTokens), RequireStaff(store), func(c *gin.Context) {
		c.String(http.StatusOK, Subject(c))
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/me", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRequireStaffAllowsAStaffUser(t *testing.T) {
	good, _ := testTokens.Mint("7", time.Minute)
	store := fakeStaffStore{user: domain.User{ID: 7, Role: domain.RoleStaff}}

	rec := staffRequest(t, store, "Bearer "+good)
	if rec.Code != http.StatusOK || rec.Body.String() != "7" {
		t.Errorf("staff user: %d %q", rec.Code, rec.Body.String())
	}
}

func TestRequireStaffRejectsACustomer(t *testing.T) {
	good, _ := testTokens.Mint("7", time.Minute)
	store := fakeStaffStore{user: domain.User{ID: 7, Role: domain.RoleCustomer}}

	if rec := staffRequest(t, store, "Bearer "+good); rec.Code != http.StatusForbidden {
		t.Errorf("customer: %d, want 403", rec.Code)
	}
}

// A guest's subject is a bare uuid, so the store answers not-found exactly
// as an unknown or malformed subject would.
func TestRequireStaffRejectsAGuestToken(t *testing.T) {
	good, _ := testTokens.Mint("a-guest-uuid", time.Minute)
	store := fakeStaffStore{err: domain.ErrNotFound}

	if rec := staffRequest(t, store, "Bearer "+good); rec.Code != http.StatusForbidden {
		t.Errorf("guest token: %d, want 403", rec.Code)
	}
}

// No token at all never reaches RequireStaff; RequireUser answers first.
func TestRequireStaffNeedsRequireUserFirst(t *testing.T) {
	store := fakeStaffStore{user: domain.User{ID: 7, Role: domain.RoleStaff}}

	if rec := staffRequest(t, store, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: %d, want 401", rec.Code)
	}
}
