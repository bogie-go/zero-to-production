package models_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"ecommerce/app/domain"
	"ecommerce/test/testdb"
)

// Whatever the id looks like, a price_rule that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownPriceRulesAreNotFound(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetPriceRule(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetPriceRule(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeletePriceRule(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeletePriceRule(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	// A valid-looking rule, so Validate (no longer a no-op) passes and the
	// not-found check is what actually gets exercised.
	if _, err := store.UpdatePriceRule(ctx, domain.PriceRule{ID: 999999999, Name: "Nope", Kind: "fixed_discount", Properties: []byte(`{}`)}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdatePriceRule(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

// The admin's list is every rule in the order the engine runs them, active or
// not; the cart's own list is only the active ones, in the same order.
func TestAdminListIncludesInactiveRulesInRunningOrder(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()
	mk := func(name string, priority int32, active bool) domain.PriceRule {
		r, err := store.CreatePriceRule(ctx, domain.PriceRule{Name: name, Kind: "fixed_discount", Properties: []byte(`{"amount_cents":100}`), Active: active, Priority: priority})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	mk("second", 20, true)
	mk("off", 5, false)
	mk("first", 10, true)

	all, err := store.ListAllPriceRules(ctx)
	if err != nil || len(all) != 3 || all[0].Name != "off" || all[1].Name != "first" || all[2].Name != "second" {
		t.Errorf("all = %v, %v; want off, first, second by priority", names(all), err)
	}
	active, err := store.ListActivePriceRules(ctx)
	if err != nil || len(active) != 2 || active[0].Name != "first" || active[1].Name != "second" {
		t.Errorf("active = %v, %v; want first, second", names(active), err)
	}
}

func names(rs []domain.PriceRule) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Name
	}
	return out
}

// Switching a rule off changes nothing but whether it applies: the very next
// read of the active rules (the cart's) no longer has it, and switching it
// back on restores it.
func TestSetPriceRuleActiveTakesEffectOnTheNextRead(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()
	rule, err := store.CreatePriceRule(ctx, domain.PriceRule{Name: "Sale", Kind: "fixed_discount", Properties: []byte(`{"amount_cents":100}`), Active: true})
	if err != nil {
		t.Fatal(err)
	}
	id := strconv.FormatInt(rule.ID, 10)

	off, err := store.SetPriceRuleActive(ctx, id, false)
	if err != nil || off.Active || off.Name != "Sale" {
		t.Fatalf("off = %+v, %v", off, err)
	}
	if active, _ := store.ListActivePriceRules(ctx); len(active) != 0 {
		t.Errorf("a switched-off rule is still active: %v", names(active))
	}
	if _, err := store.SetPriceRuleActive(ctx, id, true); err != nil {
		t.Fatal(err)
	}
	if active, _ := store.ListActivePriceRules(ctx); len(active) != 1 {
		t.Errorf("switched back on: %v, want the rule", names(active))
	}
	for _, bad := range []string{"nope", "", "999999999"} {
		if _, err := store.SetPriceRuleActive(ctx, bad, true); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("SetPriceRuleActive(%q): err = %v, want ErrNotFound", bad, err)
		}
	}
}

// A rule needs a name and a kind, and cannot end before it starts.
func TestPriceRulesAreValidatedOnSave(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()
	start, before := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for name, r := range map[string]domain.PriceRule{
		"no name":               {Kind: "fixed_discount", Properties: []byte(`{}`)},
		"no kind":               {Name: "x", Properties: []byte(`{}`)},
		"ends before it starts": {Name: "x", Kind: "fixed_discount", Properties: []byte(`{}`), StartsAt: &start, EndsAt: &before},
	} {
		if _, err := store.CreatePriceRule(ctx, r); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
}

// The cart's rules are the active ones inside their dates: a rule that has not
// started, or has ended, is left out even though it is switched on, and one
// with no dates (or only one end) applies whenever that allows. This is the
// scheduling the admin's start and end dates rely on.
func TestActivePriceRulesRespectTheirDates(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()
	hour := time.Hour
	past, future := time.Now().Add(-48*hour), time.Now().Add(48*hour)
	earlier := time.Now().Add(-24 * hour)
	later := time.Now().Add(24 * hour)

	mk := func(name string, priority int32, starts, ends *time.Time) {
		_, err := store.CreatePriceRule(ctx, domain.PriceRule{
			Name: name, Kind: "fixed_discount", Properties: []byte(`{"amount_cents":100}`),
			Active: true, Priority: priority, StartsAt: starts, EndsAt: ends,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	mk("no dates", 10, nil, nil)
	mk("inside its window", 20, &past, &future)
	mk("started, no end", 30, &earlier, nil)
	mk("ends later, no start", 40, nil, &later)
	mk("not started yet", 50, &later, &future)
	mk("already ended", 60, &past, &earlier)
	mk("ended, no start", 70, nil, &earlier)

	active, err := store.ListActivePriceRules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := names(active)
	want := []string{"no dates", "inside its window", "started, no end", "ends later, no start"}
	if len(got) != len(want) {
		t.Fatalf("active = %v, want %v (in priority order)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("active[%d] = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}
