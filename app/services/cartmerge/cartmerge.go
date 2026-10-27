// Package cartmerge folds a guest's cart into the account they have just
// signed in or signed up as. Both POST /sessions and POST /users do it, so it
// lives here rather than in either controller.
package cartmerge

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"ecommerce/app/models"
	"ecommerce/app/services/token"
)

// store is the slice of the store the merge uses, declared here beside its
// user. *adapter.Store satisfies it.
type store interface {
	GetCartByToken(ctx context.Context, token string) (models.Cart, error)
	FindOrCreateCartByToken(ctx context.Context, token string) (models.Cart, error)
	CartContents(ctx context.Context, cartID int64) ([]models.CartLine, error)
	AddCartItem(ctx context.Context, cartID, productID int64, quantity int32) (models.CartItem, error)
	DeleteCartAndItems(ctx context.Context, cartID int64) error
}

// verifier is the slice of the token service the merge uses: reading who the
// request's own bearer token is for.
type verifier interface {
	Verify(raw string) (token.Claims, error)
}

// Merger merges carts.
type Merger struct {
	Store  store
	Tokens verifier
	Log    *slog.Logger
}

// FromAuthorization folds the cart named by the bearer token in an
// Authorization header into newSubject's cart, then removes the old one.
// Best effort by design: no header, an invalid or expired token, the account's
// own token, or no cart under it is not an error, just nothing to merge. A
// line that cannot be added (out of stock, say) is logged and skipped rather
// than failing, since the person is signing in regardless of what became of
// one guest cart line. Only a failure of the store itself is returned.
func (m Merger) FromAuthorization(ctx context.Context, authorization, newSubject string) error {
	raw, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok || raw == "" {
		return nil
	}
	claims, err := m.Tokens.Verify(raw)
	if err != nil || claims.Subject == newSubject {
		return nil
	}

	oldCart, err := m.Store.GetCartByToken(ctx, claims.Subject)
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil
	case err != nil:
		return err
	}

	lines, err := m.Store.CartContents(ctx, oldCart.ID)
	if err != nil {
		return err
	}
	if len(lines) > 0 {
		newCart, err := m.Store.FindOrCreateCartByToken(ctx, newSubject)
		if err != nil {
			return err
		}
		for _, l := range lines {
			if _, err := m.Store.AddCartItem(ctx, newCart.ID, l.ProductID, l.Quantity); err != nil {
				m.Log.WarnContext(ctx, "cartmerge: could not merge a guest cart line",
					"product_id", l.ProductID, "quantity", l.Quantity, "err", err)
			}
		}
	}
	return m.Store.DeleteCartAndItems(ctx, oldCart.ID)
}
