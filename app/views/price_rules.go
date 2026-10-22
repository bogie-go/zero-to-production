package views

import (
	"encoding/json"
	"time"
)

// PriceRuleResponse is one price rule on the wire. Summary is the rule in a
// sentence ("20% off"), and Status where it stands right now: "active",
// "off" (switched off), "scheduled" (on, but not started) or "ended" (on, but
// past its end).
type PriceRuleResponse struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Kind       string          `json:"kind"`
	KindLabel  string          `json:"kind_label"`
	Summary    string          `json:"summary"`
	Properties json.RawMessage `json:"properties"`
	Active     bool            `json:"active"`
	Priority   int32           `json:"priority"`
	StartsAt   *time.Time      `json:"starts_at"`
	EndsAt     *time.Time      `json:"ends_at"`
	Status     string          `json:"status"`
}

// PriceRulesResponse is a listing, in the order the engine runs the rules.
type PriceRulesResponse struct {
	PriceRules []PriceRuleResponse `json:"price_rules"`
}

// PriceRuleRequest is the body of POST and PATCH /admin/price_rules.
// Properties is whatever the kind takes (see KindsResponse); the server
// validates it with the kind's own rules.
type PriceRuleRequest struct {
	Name       string          `json:"name"`
	Kind       string          `json:"kind"`
	Properties json.RawMessage `json:"properties"`
	Active     bool            `json:"active"`
	Priority   int32           `json:"priority"`
	StartsAt   *time.Time      `json:"starts_at"`
	EndsAt     *time.Time      `json:"ends_at"`
}

// FieldResponse is one property a rule kind takes, so the form can ask for it.
// Type is "cents" (typed as dollars, stored as cents), "percent" or "int".
type FieldResponse struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Help  string `json:"help"`
}

// KindResponse describes one kind of rule.
type KindResponse struct {
	Kind   string          `json:"kind"`
	Label  string          `json:"label"`
	Help   string          `json:"help"`
	Note   string          `json:"note"`
	Fields []FieldResponse `json:"fields"`
}

// KindsResponse is GET /admin/price_rules/kinds.
type KindsResponse struct {
	Kinds []KindResponse `json:"kinds"`
}
