package models

import (
	"fmt"
	"time"
)

// Order is a order. Imports nothing third-party: ids are int64s here and
// become bigint at the store boundary.
//
// SubtotalCents, DiscountCents and TotalCents are the cart's totals at the
// moment checkout happened, not a live recalculation — a later price
// change or sale must never alter a receipt someone already has.
type Order struct {
	ID            int64
	UserID        int64
	Status        string
	SubtotalCents int32
	DiscountCents int32
	TotalCents    int32
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OrderStatuses are the values Status may hold. There is no "paid" or
// "cancelled" transition yet — that is payment webhook handling's job,
// still ahead in the plan — so every order is created "pending" and stays
// there.
var OrderStatuses = []string{"pending", "paid", "cancelled"}

// Validate is the model's validations, as Rails would put them. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule.
func (o Order) Validate() error {
	if o.UserID == 0 {
		return fmt.Errorf("%w: user is required", ErrInvalid)
	}
	known := false
	for _, s := range OrderStatuses {
		if o.Status == s {
			known = true
			break
		}
	}
	if !known {
		return fmt.Errorf("%w: status %q is not recognized", ErrInvalid, o.Status)
	}
	if o.SubtotalCents < 0 || o.DiscountCents < 0 || o.TotalCents < 0 {
		return fmt.Errorf("%w: amounts cannot be negative", ErrInvalid)
	}
	return nil
}
