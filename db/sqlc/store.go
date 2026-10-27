// Package db is the database: sqlc's generated queries and the Store written
// around them.
//
// Nothing here is hand-written SQL. The *.sql.go files are compiled by sqlc
// from db/queries/*.sql (make sqlc) and speak pgtype; the Store's own methods
// speak app/models, validate on save, and convert between the two, so nothing
// above this package imports pgx. The generated Queries is held privately
// rather than embedded, so that boundary cannot leak by accident.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ecommerce/app/models"
)

// Store wraps the connection pool and the generated queries.
type Store struct {
	pool *pgxpool.Pool
	q    *Queries

	// onOrderPlaced runs inside CreateOrderFromCart's transaction; see
	// OnOrderPlaced in orders_extra.go.
	onOrderPlaced func(ctx context.Context, tx pgx.Tx, order models.Order) error
	// onUserRegistered runs inside RegisterUser's transaction; see
	// OnUserRegistered in users_extra.go.
	onUserRegistered func(ctx context.Context, tx pgx.Tx, user models.User) error
}

// Open connects and verifies the database is reachable, so a bad URL fails at
// boot rather than on the first request.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: open database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping database: %w", err)
	}
	return &Store{pool: pool, q: New(pool)}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Ping reports whether the database answers.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// withTx runs fn against a *Queries bound to one database transaction, and
// hands fn the transaction itself for the rare caller that needs to pass it
// on (a hook enqueueing a job in the same transaction). Every call fn makes
// through the queries happens on the same connection and the
// same not-yet-committed transaction, however many tables it touches. If fn
// returns an error, everything it did is rolled back as if none of it had
// happened; only a nil return commits. This is Go's version of Rails'
// `ActiveRecord::Base.transaction do ... end` — explicit about the
// boundary (the closure) rather than implicit about which calls join it.
//
// Unexported: a hand-written *Store method that needs a real transaction —
// a checkout, a transfer, anything that writes to more than one table and
// must not half-succeed — calls it from its own sibling file in
// db/sqlc/, such as orders_extra.go, not from above this package.
// CreateOrderFromCart (orders_extra.go) and DeleteCartAndItems
// (carts_extra.go) both already do.
func (s *Store) withTx(ctx context.Context, fn func(tx pgx.Tx, q *Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin transaction: %w", err)
	}
	if err := fn(tx, s.q.WithTx(tx)); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("store: %w (rollback also failed: %v)", err, rbErr)
		}
		return err
	}
	return tx.Commit(ctx)
}

// Pool is the connection pool, for River and nothing else: the job queue
// shares the database so an enqueue can ride a business transaction. Every
// other caller goes through the methods above.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }
