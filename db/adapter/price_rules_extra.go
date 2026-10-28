package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/models"
	"ecommerce/db/sqlc"
)

// GetPriceRuleByName finds one price rule by its name.
func (s *Store) GetPriceRuleByName(ctx context.Context, name string) (models.PriceRule, error) {
	row, err := s.q.GetPriceRuleByName(ctx, name)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.PriceRule{}, models.ErrNotFound
	case err != nil:
		return models.PriceRule{}, fmt.Errorf("store: get price rule by name %q: %w", name, err)
	}
	return priceRuleFromRow(row), nil
}

// FindOrCreatePriceRuleByName returns the price rule named p.Name, creating
// it from p if none exists yet — the same find_or_create_by idiom
// FindOrCreateProductByName uses, for the same reason: a seed must be safe
// to run more than once.
func (s *Store) FindOrCreatePriceRuleByName(ctx context.Context, p models.PriceRule) (models.PriceRule, error) {
	existing, err := s.GetPriceRuleByName(ctx, p.Name)
	switch {
	case err == nil:
		return existing, nil
	case errors.Is(err, models.ErrNotFound):
		return s.CreatePriceRule(ctx, p)
	default:
		return models.PriceRule{}, err
	}
}

// ListActivePriceRules returns the price rules currently in effect — active,
// and within their starts_at/ends_at window if they have one — in priority
// order, ready to hand straight to pricing.Calculate.
func (s *Store) ListActivePriceRules(ctx context.Context) ([]models.PriceRule, error) {
	rows, err := s.q.ListActivePriceRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: list active price rules: %w", err)
	}
	out := make([]models.PriceRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, priceRuleFromRow(r))
	}
	return out, nil
}

// ListAllPriceRules is every rule, active or not, in the order the engine
// runs them: the admin's list. ListActivePriceRules is what the cart uses.
func (s *Store) ListAllPriceRules(ctx context.Context) ([]models.PriceRule, error) {
	rows, err := s.q.ListAllPriceRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: list all price rules: %w", err)
	}
	out := make([]models.PriceRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, priceRuleFromRow(r))
	}
	return out, nil
}

// SetPriceRuleActive turns a rule on or off and nothing else: the next cart
// request recomputes with it or without it.
func (s *Store) SetPriceRuleActive(ctx context.Context, id string, active bool) (models.PriceRule, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.PriceRule{}, models.ErrNotFound
	}
	row, err := s.q.SetPriceRuleActive(ctx, sqlc.SetPriceRuleActiveParams{ID: pk, Active: active})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.PriceRule{}, models.ErrNotFound
	case err != nil:
		return models.PriceRule{}, fmt.Errorf("store: set price rule %s active: %w", id, err)
	}
	return priceRuleFromRow(row), nil
}
