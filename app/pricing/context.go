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

// Line is one cart line as a rule sees it: a product, what one unit costs
// and how many are bought. Rules that depend on a product's own price, like
// buy-X-get-Y, work line by line; the ones that only need the total ignore
// Lines and work on CurrentCents.
type Line struct {
	ProductID int64
	UnitCents int64
	Quantity  int64
}

// PriceContext carries a price through the rule pipeline, the same role
// Rack's env plays for Rails middleware: each rule reads it, changes
// CurrentCents, and the next rule sees the result.
type PriceContext struct {
	OriginalCents int64
	CurrentCents  int64
	AppliedRules  []AppliedRule

	// Lines are the cart's lines, when the price being worked out is a
	// cart's. Empty for a bare price (Calculate), where a per-line rule has
	// nothing to act on.
	Lines []Line

	// Metadata is read-and-written by individual rules for whatever they
	// need beyond a price. Keeping it a loose map, rather than named fields
	// on PriceContext, means a new rule kind never needs a change here.
	Metadata map[string]any
}

// NewPriceContext starts a pipeline at a price. metadata may be nil.
func NewPriceContext(cents int64, metadata map[string]any) *PriceContext {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return &PriceContext{OriginalCents: cents, CurrentCents: cents, Metadata: metadata}
}
