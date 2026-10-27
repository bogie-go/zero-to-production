package seeds

import (
	"context"
	"fmt"
	"os"

	"ecommerce/app/models"
	"ecommerce/app/services/password"
	"ecommerce/config"
	"ecommerce/db/sqlc"
)

// staffEmail and staffPassword are a development-only login: no admin UI
// exists yet to promote anyone, so this is how a reader reaches
// /admin/products at all before Task F grows a way to manage staff
// properly. Not meant for a real environment — see docs/bogie-gaps.md.
const (
	staffEmail    = "staff@shop.test"
	staffPassword = "staff-password-please-change"
)

// SeedStaffUser seeds the one staff account. db seed can run this any
// number of times: FindOrCreateUserByEmail leaves it alone once it
// exists, so re-running never resets a password or role changed by hand
// since.
func SeedStaffUser(ctx context.Context, s *db.Store) error {
	// A login whose password is in this repository must never exist on a
	// deployed system (a deployment runs as production). A real
	// environment promotes an account that signed up: ecommerce staff promote.
	if os.Getenv(config.EnvVar) == config.EnvProduction {
		return nil
	}
	digest, err := password.Hash(staffPassword)
	if err != nil {
		return fmt.Errorf("seeds: hash staff password: %w", err)
	}
	_, err = s.FindOrCreateUserByEmail(ctx, models.User{
		Email: staffEmail, PasswordDigest: digest, Role: models.RoleStaff,
	})
	return err
}
