// Package pricing is the price rule engine: a registry of rule kinds behind
// one interface. Rails would store this family of types in one table with a
// `type` discriminator column (Single Table Inheritance) and let the class
// table auto-instantiate the right subclass. Go has no open class table, so
// a price_rule row's `kind` column plus this registry is the explicit stand
// in: new rule kinds are a new file registering itself, never a change here.
package pricing

import (
	"encoding/json"
	"fmt"

	"ecommerce/app/models"
)

// Rule is one pricing rule: given the price so far, it decides whether it
// applies and, if so, changes it. Every kind of discount is a different Go
// type implementing this one method, the same way every Rails subclass
// implements `transform`.
type Rule interface {
	// Apply changes ctx.CurrentCents if the rule applies, and returns a
	// detail describing what it did, for the breakdown a caller can show.
	// A nil detail means the rule did not apply — MinimumPurchaseRule below
	// its threshold, for instance — and nothing is recorded.
	Apply(ctx *PriceContext) (detail map[string]any, err error)
}

// Decoder builds a Rule from a price_rule row's properties column. Each kind
// registers its own in an init func, in its own file.
type Decoder func(properties json.RawMessage) (Rule, error)

var registry = map[string]Decoder{}

// Register adds a kind to the registry under the name its price_rule rows
// use in their kind column.
func Register(kind string, decode Decoder) {
	registry[kind] = decode
}

// Load builds the Rule a price_rule row names. An unregistered kind is
// ErrInvalid: the row itself is malformed, not merely absent.
func Load(kind string, properties json.RawMessage) (Rule, error) {
	decode, ok := registry[kind]
	if !ok {
		return nil, fmt.Errorf("%w: no such price rule kind %q", models.ErrInvalid, kind)
	}
	return decode(properties)
}
