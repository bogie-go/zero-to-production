package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ecommerce/app/services/token"
)

var testTokens = token.Maker{Secret: []byte("a-test-secret-of-at-least-thirty-two-bytes"), Issuer: "test"}

func userRequest(t *testing.T, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/me", RequireUser(testTokens), func(c *gin.Context) { c.String(http.StatusOK, Subject(c)) })
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRequireUser(t *testing.T) {
	good, _ := testTokens.Mint("user-42", time.Minute)
	expired, _ := testTokens.Mint("user-42", -time.Minute)

	if rec := userRequest(t, "Bearer "+good); rec.Code != http.StatusOK || rec.Body.String() != "user-42" {
		t.Errorf("good token: %d %q", rec.Code, rec.Body.String())
	}
	for name, header := range map[string]string{
		"none": "", "expired": "Bearer " + expired, "garbage": "Bearer nope", "wrong scheme": "Basic " + good,
	} {
		if rec := userRequest(t, header); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: %d, want 401", name, rec.Code)
		}
	}
}
