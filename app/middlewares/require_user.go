package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ecommerce/app/services/token"
	"ecommerce/app/views"
)

// userTokens is the slice of the token service this middleware uses,
// declared here (rule 2). token.Maker satisfies it.
type userTokens interface {
	Verify(raw string) (token.Claims, error)
}

const subjectKey = "subject"

// RequireUser guards routes for a signed-in user: a bearer token in the
// Authorization header that the token service verifies. The token's subject
// is then on the context for handlers, through Subject. Who the subject is,
// a user id usually, is for the caller and the handler to agree on;
// nothing is loaded here.
func RequireUser(tokens userTokens) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			c.Header("WWW-Authenticate", `Bearer realm="ecommerce"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, views.ErrorMessage("a bearer token is required"))
			return
		}
		claims, err := tokens.Verify(raw)
		if err != nil {
			c.Header("WWW-Authenticate", `Bearer realm="ecommerce", error="invalid_token"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, views.Error(err))
			return
		}
		c.Set(subjectKey, claims.Subject)
		c.Next()
	}
}

// Subject is who the request's token was minted for, or "" on a route that
// RequireUser does not guard.
func Subject(c *gin.Context) string { return c.GetString(subjectKey) }
