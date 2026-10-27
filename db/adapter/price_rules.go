package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"ecommerce/app/models"
	"ecommerce/db/sqlc"
)

// PriceRules, as the models see them. Each method converts models.PriceRule to and from
// the row type sqlc generated, and maps pgx's "no rows" to models.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in price_rules_extra.go, not here.

// CreatePriceRule validates a price_rule, inserts it, and returns it with its id and
// timestamps. Validation runs on save, as Rails's does, so every caller
// gets it and none can forget it.
func (s *Store) CreatePriceRule(ctx context.Context, p models.PriceRule) (models.PriceRule, error) {
	if err := p.Validate(); err != nil {
		return models.PriceRule{}, err
	}
	row, err := s.q.CreatePriceRule(ctx, sqlc.CreatePriceRuleParams{Name: p.Name, Kind: p.Kind, Properties: p.Properties, Active: p.Active, Priority: p.Priority, StartsAt: timePtrToTimestamptz(p.StartsAt), EndsAt: timePtrToTimestamptz(p.EndsAt)})
	if err != nil {
		return models.PriceRule{}, fmt.Errorf("store: create price_rule: %w", err)
	}
	return price_ruleFromRow(row), nil
}

// GetPriceRule finds one price_rule. An id that is not a valid id is simply not found.
func (s *Store) GetPriceRule(ctx context.Context, id string) (models.PriceRule, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.PriceRule{}, models.ErrNotFound
	}
	row, err := s.q.GetPriceRule(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.PriceRule{}, models.ErrNotFound
	case err != nil:
		return models.PriceRule{}, fmt.Errorf("store: get price_rule %s: %w", id, err)
	}
	return price_ruleFromRow(row), nil
}

// ListPriceRules returns the newest price_rules first, at most limit of them.
func (s *Store) ListPriceRules(ctx context.Context, limit int32) ([]models.PriceRule, error) {
	rows, err := s.q.ListPriceRules(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list price_rules: %w", err)
	}
	out := make([]models.PriceRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, price_ruleFromRow(r))
	}
	return out, nil
}

// UpdatePriceRule validates a price_rule, writes its columns, and returns the stored row.
func (s *Store) UpdatePriceRule(ctx context.Context, p models.PriceRule) (models.PriceRule, error) {
	if err := p.Validate(); err != nil {
		return models.PriceRule{}, err
	}
	pk := p.ID
	row, err := s.q.UpdatePriceRule(ctx, sqlc.UpdatePriceRuleParams{ID: pk, Name: p.Name, Kind: p.Kind, Properties: p.Properties, Active: p.Active, Priority: p.Priority, StartsAt: timePtrToTimestamptz(p.StartsAt), EndsAt: timePtrToTimestamptz(p.EndsAt)})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.PriceRule{}, models.ErrNotFound
	case err != nil:
		return models.PriceRule{}, fmt.Errorf("store: update price_rule %v: %w", p.ID, err)
	}
	return price_ruleFromRow(row), nil
}

// DeletePriceRule removes a price_rule, and says so if there was none.
func (s *Store) DeletePriceRule(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return models.ErrNotFound
	}
	n, err := s.q.DeletePriceRule(ctx, pk)
	if err != nil {
		return fmt.Errorf("store: delete price_rule %s: %w", id, err)
	}
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func price_ruleFromRow(r sqlc.PriceRule) models.PriceRule {
	return models.PriceRule{
		ID:         r.ID,
		Name:       r.Name,
		Kind:       r.Kind,
		Properties: r.Properties,
		Active:     r.Active,
		Priority:   r.Priority,
		StartsAt:   timestamptzToTimePtr(r.StartsAt),
		EndsAt:     timestamptzToTimePtr(r.EndsAt),
		CreatedAt:  r.CreatedAt.Time,
		UpdatedAt:  r.UpdatedAt.Time,
	}
}

// starts_at and ends_at are the first nullable columns in this app: every
// other column is NOT NULL with a type-appropriate default (AGENTS.md), but
// an unscheduled price rule has neither a start nor an end. pgtype.Timestamptz
// carries that nullability from Postgres; *time.Time carries it in Go.
func timestamptzToTimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func timePtrToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
