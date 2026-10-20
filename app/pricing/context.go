package pricing

// AppliedRule records one rule's effect, in the order it ran — the
// breakdown a cart response or an admin screen can show, without either
// needing to understand every rule kind's own detail fields.
type AppliedRule struct {
	Name          string
	Kind          string
	PreviousCents int64
	NewCents      int64
	Detail        map[string]any
}

// PriceContext carries a price through the rule pipeline, the same role
// Rack's env plays for Rails middleware: each rule reads it, changes
// CurrentCents, and the next rule sees the result.
type PriceContext struct {
	OriginalCents int64
	CurrentCents  int64
	AppliedRules  []AppliedRule

	// Metadata is read-and-written by individual rules for whatever they
	// need beyond a price — BuyXGetYRule reads a "quantity" here, for
	// instance. Keeping it a loose map, rather than named fields on
	// PriceContext, means a new rule kind never needs a change here either.
	Metadata map[string]any
}

// NewPriceContext starts a pipeline at a price. metadata may be nil.
func NewPriceContext(cents int64, metadata map[string]any) *PriceContext {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return &PriceContext{OriginalCents: cents, CurrentCents: cents, Metadata: metadata}
}
