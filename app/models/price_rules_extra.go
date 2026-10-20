package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

// GetPriceRuleByName finds one price rule by its name.
func (s *Store) GetPriceRuleByName(ctx context.Context, name string) (domain.PriceRule, error) {
	row, err := s.q.GetPriceRuleByName(ctx, name)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.PriceRule{}, domain.ErrNotFound
	case err != nil:
		return domain.PriceRule{}, fmt.Errorf("models: get price rule by name %q: %w", name, err)
	}
	return price_ruleFromRow(row), nil
}

// FindOrCreatePriceRuleByName returns the price rule named p.Name, creating
// it from p if none exists yet — the same find_or_create_by idiom
// FindOrCreateProductByName uses, for the same reason: a seed must be safe
// to run more than once.
func (s *Store) FindOrCreatePriceRuleByName(ctx context.Context, p domain.PriceRule) (domain.PriceRule, error) {
	existing, err := s.GetPriceRuleByName(ctx, p.Name)
	switch {
	case err == nil:
		return existing, nil
	case errors.Is(err, domain.ErrNotFound):
		return s.CreatePriceRule(ctx, p)
	default:
		return domain.PriceRule{}, err
	}
}

// ListActivePriceRules returns the price rules currently in effect — active,
// and within their starts_at/ends_at window if they have one — in priority
// order, ready to hand straight to pricing.Calculate.
func (s *Store) ListActivePriceRules(ctx context.Context) ([]domain.PriceRule, error) {
	rows, err := s.q.ListActivePriceRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("models: list active price rules: %w", err)
	}
	out := make([]domain.PriceRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, price_ruleFromRow(r))
	}
	return out, nil
}
