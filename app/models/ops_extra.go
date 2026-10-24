package models

import (
	"context"
	"fmt"
	"time"

	"ecommerce/app/domain"
)

// Ops reads the schema version and the job queue's state. The jobs live in
// River's own tables, which sqlc does not know, so this is plain SQL on the
// pool rather than a generated query.
func (s *Store) Ops(ctx context.Context) (domain.Ops, error) {
	ops := domain.Ops{Jobs: map[string]int64{}}

	if err := s.pool.QueryRow(ctx,
		"SELECT coalesce(max(version_id), 0) FROM goose_db_version WHERE is_applied").Scan(&ops.SchemaVersion); err != nil {
		return domain.Ops{}, fmt.Errorf("models: schema version: %w", err)
	}

	rows, err := s.pool.Query(ctx, "SELECT state::text, count(*) FROM river_job GROUP BY state")
	if err != nil {
		return domain.Ops{}, fmt.Errorf("models: job counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		var n int64
		if err := rows.Scan(&state, &n); err != nil {
			return domain.Ops{}, fmt.Errorf("models: job counts: %w", err)
		}
		ops.Jobs[state] = n
	}
	if err := rows.Err(); err != nil {
		return domain.Ops{}, fmt.Errorf("models: job counts: %w", err)
	}

	var waited *float64
	if err := s.pool.QueryRow(ctx,
		"SELECT extract(epoch FROM now() - min(scheduled_at)) FROM river_job WHERE state = 'available'").Scan(&waited); err != nil {
		return domain.Ops{}, fmt.Errorf("models: oldest waiting job: %w", err)
	}
	if waited != nil && *waited > 0 {
		ops.OldestWaiting = time.Duration(*waited * float64(time.Second))
	}
	return ops, nil
}
