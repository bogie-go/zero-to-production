// Package app wires the process together: configuration, connections,
// controllers. It is the only place that knows how every part fits, which
// keeps that knowledge out of the packages themselves.
package app

import (
	"context"
	"log/slog"
	"os"

	"ecommerce/app/controllers"
	"ecommerce/app/models"
	"ecommerce/app/services/database"
	"ecommerce/app/services/migrate"
	"ecommerce/config"
	"ecommerce/lib/logger"
)

// Application holds everything the process needs.
type Application struct {
	Config config.Config
	Log    *slog.Logger
	Store  *models.Store
	Server *controllers.Server
}

// NewApplication loads configuration and opens every connection. It returns an
// error rather than exiting: only main decides to stop the process.
func NewApplication(ctx context.Context) (*Application, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	log := logger.New(cfg.App.LogLevel, cfg.Env)

	// Printed before anything connects, so a wrong setting shows up next to
	// the failure it causes rather than after it.
	if cfg.Env != config.EnvTest {
		cfg.PrintTo(os.Stderr)
	}

	// Schema first, then everything that depends on it. The migrations are
	// embedded in this binary, so the schema a container runs against is
	// always the one its code was built for. Failing here stops the deploy,
	// which is the point: serving traffic against a half-applied schema is
	// worse.
	if cfg.App.MigrateAtBoot {
		if err := migrate.Run(ctx, cfg.App.DatabaseURL, log); err != nil {
			// database.Hint tells apart the three ways this fails -- nothing
			// listening, credentials refused, database not created yet --
			// because the pgx error alone reads the same for all of them.
			return nil, database.Hint(err, cfg.App.DatabaseURL)
		}
	}

	store, err := models.Open(ctx, cfg.App.DatabaseURL)
	if err != nil {
		return nil, database.Hint(err, cfg.App.DatabaseURL)
	}

	server := controllers.NewServer(cfg, log)

	// Each controller is built here, with its dependencies, and assigned to
	// its NAMED field on the root server, or on its namespace's. `bogie g
	// controller` inserts one line above the marker; by hand it is the same
	// one line, with the controller's dependencies passed in:
	//
	//	server.Posts = controllers.NewPostsController(posts.New(store), log)
	//
	server.Products = controllers.NewProductsController(log)
	// bogie:wire
	server.SetupRoutes()

	return &Application{Config: cfg, Log: log, Store: store, Server: server}, nil
}

// Start serves HTTP until ctx is cancelled.
func (a *Application) Start(ctx context.Context) error {
	defer a.Close()
	return a.Server.Start(ctx)
}

// Close releases connections.
func (a *Application) Close() {
	if a.Store != nil {
		a.Store.Close()
	}
}
