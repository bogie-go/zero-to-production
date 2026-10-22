package pricing

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Spec describes a rule kind to the admin: what to call it, which fields its
// properties take, and how to say it in a sentence. Each kind registers its
// own beside its decoder, so adding a kind is still one new file and the admin
// form learns about it without a change anywhere else.
type Spec struct {
	Kind  string
	Label string
	Help  string
	// Note is a caveat worth showing beside the kind, such as a limitation.
	Note   string
	Fields []Field
}

// Field is one property of a kind.
type Field struct {
	Name  string // the JSON key in properties: amount_cents
	Label string // what the form calls it
	// Type says how the form reads and writes it: "cents" is typed as dollars
	// and stored as cents, "percent" is a number from 0 to 100, "int" a whole
	// number.
	Type string
	Help string
}

var specs = map[string]Spec{}

// RegisterSpec adds a kind's description. Called from the kind's init, next
// to Register.
func RegisterSpec(s Spec) { specs[s.Kind] = s }

// Specs is every registered kind, by label, for the admin's kind picker.
func Specs() []Spec {
	out := make([]Spec, 0, len(specs))
	for _, s := range specs {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// Describer is the optional half of a Rule: a sentence about it, like "20%
// off". A rule that does not implement it is summarized by its kind's label.
type Describer interface{ Summary() string }

// Summarize says what a stored rule does, in a sentence, so a list can show it
// without every caller knowing each kind's properties. A row that does not
// load (an unknown kind, properties that no longer validate) says so.
func Summarize(kind string, properties json.RawMessage) string {
	rule, err := Load(kind, properties)
	if err != nil {
		return fmt.Sprintf("invalid %s rule", kind)
	}
	if d, ok := rule.(Describer); ok {
		return d.Summary()
	}
	if s, ok := specs[kind]; ok {
		return s.Label
	}
	return kind
}

// money formats cents as dollars for a Summary.
func money(cents int64) string { return fmt.Sprintf("$%d.%02d", cents/100, cents%100) }
