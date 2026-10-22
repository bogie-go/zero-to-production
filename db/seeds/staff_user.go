package seeds

import (
	"context"
	"fmt"

	"ecommerce/app/domain"
	"ecommerce/app/models"
	"ecommerce/app/services/password"
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
func SeedStaffUser(ctx context.Context, s *models.Store) error {
	digest, err := password.Hash(staffPassword)
	if err != nil {
		return fmt.Errorf("seeds: hash staff password: %w", err)
	}
	_, err = s.FindOrCreateUserByEmail(ctx, domain.User{
		Email: staffEmail, PasswordDigest: digest, Role: domain.RoleStaff,
	})
	return err
}
