package pricing

import (
	"encoding/json"
	"errors"
	"testing"

	"ecommerce/app/domain"
)

func rule(id int64, name, kind string, properties any) domain.PriceRule {
	b, err := json.Marshal(properties)
	if err != nil {
		panic(err)
	}
	return domain.PriceRule{ID: id, Name: name, Kind: kind, Properties: b, Active: true}
}

func TestCalculate(t *testing.T) {
	tests := []struct {
		name       string
		startCents int64
		rules      []domain.PriceRule
		metadata   map[string]any
		wantFinal  int64
		wantRules  int
	}{
		{
			name:       "no rules leaves the price alone",
			startCents: 1000,
			wantFinal:  1000,
		},
		{
			name:       "percentage discount",
			startCents: 1000,
			rules:      []domain.PriceRule{rule(1, "Flash Sale", "percentage_discount", PercentageDiscountRule{Percentage: 20})},
			wantFinal:  800,
			wantRules:  1,
		},
		{
			name:       "fixed discount",
			startCents: 1000,
			rules:      []domain.PriceRule{rule(1, "Early Bird", "fixed_discount", FixedDiscountRule{AmountCents: 50})},
			wantFinal:  950,
			wantRules:  1,
		},
		{
			name:       "fixed discount never goes below zero",
			startCents: 30,
			rules:      []domain.PriceRule{rule(1, "Early Bird", "fixed_discount", FixedDiscountRule{AmountCents: 50})},
			wantFinal:  0,
			wantRules:  1,
		},
		{
			name:       "minimum purchase below threshold does not apply",
			startCents: 1000,
			rules:      []domain.PriceRule{rule(1, "Big Spender", "minimum_purchase", MinimumPurchaseRule{MinimumCents: 50000, DiscountCents: 7500})},
			wantFinal:  1000,
			wantRules:  0,
		},
		{
			name:       "minimum purchase at threshold applies",
			startCents: 50000,
			rules:      []domain.PriceRule{rule(1, "Big Spender", "minimum_purchase", MinimumPurchaseRule{MinimumCents: 50000, DiscountCents: 7500})},
			wantFinal:  42500,
			wantRules:  1,
		},
		{
			name:       "buy x get y below required quantity does not apply",
			startCents: 1000,
			rules:      []domain.PriceRule{rule(1, "BOGO", "buy_x_get_y", BuyXGetYRule{RequiredQuantity: 2, FreeQuantity: 1})},
			metadata:   map[string]any{"quantity": 1},
			wantFinal:  1000,
			wantRules:  0,
		},
		{
			name:       "buy x get y at required quantity applies",
			startCents: 1000,
			rules:      []domain.PriceRule{rule(1, "BOGO", "buy_x_get_y", BuyXGetYRule{RequiredQuantity: 2, FreeQuantity: 1})},
			metadata:   map[string]any{"quantity": 5},
			wantFinal:  1000,
			wantRules:  1,
		},
		{
			name:       "stacking two rules in order",
			startCents: 1000,
			rules: []domain.PriceRule{
				rule(1, "Flash Sale", "percentage_discount", PercentageDiscountRule{Percentage: 20}),
				rule(2, "Big Spender", "minimum_purchase", MinimumPurchaseRule{MinimumCents: 500, DiscountCents: 75}),
			},
			wantFinal: 725, // 1000 -> 800 (20% off) -> 725 (meets the $5 minimum, -$0.75)
			wantRules: 2,
		},
		{
			name:       "order changes the result: fixed then percentage",
			startCents: 1000,
			rules: []domain.PriceRule{
				rule(1, "Early Bird", "fixed_discount", FixedDiscountRule{AmountCents: 50}),
				rule(2, "Flash Sale", "percentage_discount", PercentageDiscountRule{Percentage: 20}),
			},
			wantFinal: 760, // 1000 -> 950 -> 760
			wantRules: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Calculate(tt.startCents, tt.rules, tt.metadata)
			if err != nil {
				t.Fatalf("Calculate: %v", err)
			}
			if result.FinalCents != tt.wantFinal {
				t.Errorf("FinalCents = %d, want %d", result.FinalCents, tt.wantFinal)
			}
			if len(result.AppliedRules) != tt.wantRules {
				t.Errorf("len(AppliedRules) = %d, want %d", len(result.AppliedRules), tt.wantRules)
			}
		})
	}
}

func TestCalculateUnknownKindIsInvalid(t *testing.T) {
	_, err := Calculate(1000, []domain.PriceRule{rule(1, "Mystery", "no_such_kind", map[string]any{})}, nil)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestResultSavings(t *testing.T) {
	result, err := Calculate(1000, []domain.PriceRule{rule(1, "Flash Sale", "percentage_discount", PercentageDiscountRule{Percentage: 25})}, nil)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if got := result.TotalSavingsCents(); got != 250 {
		t.Errorf("TotalSavingsCents() = %d, want 250", got)
	}
	if got := result.SavingsPercentage(); got != 25 {
		t.Errorf("SavingsPercentage() = %v, want 25", got)
	}
}

func TestRuleValidation(t *testing.T) {
	tests := []struct {
		name       string
		kind       string
		properties any
	}{
		{"negative fixed amount", "fixed_discount", FixedDiscountRule{AmountCents: -1}},
		{"zero percentage", "percentage_discount", PercentageDiscountRule{Percentage: 0}},
		{"percentage over 100", "percentage_discount", PercentageDiscountRule{Percentage: 101}},
		{"zero required quantity", "buy_x_get_y", BuyXGetYRule{RequiredQuantity: 0, FreeQuantity: 1}},
		{"negative minimum", "minimum_purchase", MinimumPurchaseRule{MinimumCents: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.properties)
			if _, err := Load(tt.kind, b); !errors.Is(err, domain.ErrInvalid) {
				t.Errorf("Load(%q, %s): err = %v, want ErrInvalid", tt.kind, b, err)
			}
		})
	}
}

// Every registered kind describes itself: a label, and fields that name real
// JSON keys of its properties, so the admin form can never ask for a field the
// decoder does not read.
func TestEveryKindHasASpecWhoseFieldsAreItsProperties(t *testing.T) {
	kinds := map[string]bool{}
	for kind := range registry {
		kinds[kind] = true
	}
	for _, s := range Specs() {
		if !kinds[s.Kind] {
			t.Errorf("spec for %q, which is not a registered kind", s.Kind)
		}
		delete(kinds, s.Kind)
		if s.Label == "" || len(s.Fields) == 0 {
			t.Errorf("%s: spec lacks a label or fields: %+v", s.Kind, s)
		}
		// Feed the decoder an object holding exactly the spec's fields: if it
		// reads a different name, the properties come back zero and invalid.
		props := map[string]any{}
		for _, f := range s.Fields {
			props[f.Name] = 5
		}
		raw, _ := json.Marshal(props)
		if _, err := Load(s.Kind, raw); err != nil {
			t.Errorf("%s: a properties object built from the spec's fields (%s) does not load: %v", s.Kind, raw, err)
		}
	}
	for kind := range kinds {
		t.Errorf("kind %q is registered but has no spec", kind)
	}
}

func TestSummarizeSaysWhatARuleDoes(t *testing.T) {
	for _, c := range []struct{ kind, props, want string }{
		{"fixed_discount", `{"amount_cents": 5000}`, "$50.00 off"},
		{"percentage_discount", `{"percentage": 20}`, "20% off"},
		{"percentage_discount", `{"percentage": 12.5}`, "12.5% off"},
		{"minimum_purchase", `{"minimum_cents": 50000, "discount_cents": 7500}`, "$75.00 off orders of $500.00 or more"},
		{"buy_x_get_y", `{"required_quantity": 2, "free_quantity": 1}`, "Buy 2, get 1 free"},
		{"no_such_kind", `{}`, "invalid no_such_kind rule"},
		{"percentage_discount", `{"percentage": 500}`, "invalid percentage_discount rule"},
	} {
		if got := Summarize(c.kind, json.RawMessage(c.props)); got != c.want {
			t.Errorf("Summarize(%s, %s) = %q, want %q", c.kind, c.props, got, c.want)
		}
	}
}
