// Package seeds is ecommerce's db/seeds.rb: data every environment can load
// safely, any number of times.
//
// Unlike a migration, nothing here is tracked in a version table: `db seed`
// runs every seed below every time it is called. That makes each one's own
// job to be safe to run twice — an upsert, or ON CONFLICT DO NOTHING.
//
// `bogie g seed NAME` writes a seed and registers it below.
package seeds

import (
	"context"

	"ecommerce/db/adapter"
)

// Run applies every seed below, in order. The first failure stops the rest:
// a later seed may depend on rows an earlier one inserted.
func Run(ctx context.Context, s *adapter.Store) error {
	for _, seed := range []func() error{
		func() error { return SeedProduct(ctx, s) },
		func() error { return SeedPriceRules(ctx, s) },
		func() error { return SeedStaffUser(ctx, s) },
		func() error { return SeedCategories(ctx, s) },
		// bogie:seeds
	} {
		if err := seed(); err != nil {
			return err
		}
	}
	return nil
}
