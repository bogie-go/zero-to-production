package middlewares

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce/app/models"
	"ecommerce/app/views"
)

// staffStore is the slice of the store RequireStaff uses, declared here
// (rule 2).
type staffStore interface {
	GetUser(ctx context.Context, id string) (models.User, error)
}

// RequireStaff guards a route for a signed-in user whose role is staff. It
// must sit behind RequireUser, since it reads the subject RequireUser set
// (Subject(c)), not the Authorization header itself.
//
// The role is looked up fresh from the store on every request, never
// trusted from the bearer token's claims: a demotion has to take effect on
// the very next request, not wait for a token to expire.
//
// A guest's token (a bare uuid subject, no users row behind it), an
// unknown or malformed subject, and a real customer who isn't staff all
// end up here indistinguishable from each other — every one of them is
// simply "lacks access," the same way a wrong password and an unknown
// email both answer login identically (SessionsController).
func RequireStaff(store staffStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := store.GetUser(c.Request.Context(), Subject(c))
		if err != nil || user.Role != models.RoleStaff {
			c.AbortWithStatusJSON(http.StatusForbidden, views.ErrorMessage("staff access required"))
			return
		}
		c.Next()
	}
}
