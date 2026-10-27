package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"ecommerce/app/models"
	"ecommerce/app/services/database"
	"ecommerce/config"
	"ecommerce/db/sqlc"
)

const staffUsage = `Usage: ecommerce staff promote EMAIL

Makes an existing account a staff member, who can use /admin. The person
signs up on the site first, then an operator runs this; no password is typed
here or stored anywhere else. On a deployed host:

  kamal app exec "ecommerce staff promote ada@example.com"
`

// runStaff handles `ecommerce staff ...`.
func runStaff(ctx context.Context, args []string) (err error) {
	if len(args) != 2 || args[0] != "promote" {
		fmt.Fprint(os.Stderr, staffUsage)
		return fmt.Errorf("staff: expected: staff promote EMAIL")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	defer func() { err = database.Hint(err, cfg.Database.URL) }()
	store, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer store.Close()

	user, err := store.PromoteToStaff(ctx, args[1])
	if errors.Is(err, models.ErrNotFound) {
		return fmt.Errorf("staff: no account with the email %s; they must sign up first", args[1])
	}
	if err != nil {
		return err
	}
	fmt.Printf("%s is staff\n", user.Email)
	return nil
}
