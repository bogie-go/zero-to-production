// Package jobs holds the background jobs: one file per job, an Args type
// with a Kind and a Worker with Work. They run on River, on the same Postgres
// as everything else, so a job enqueued inside a transaction commits with the
// business row or not at all.
//
// `bogie g job send_welcome` writes a job and registers it below.
package jobs

import (
	"log/slog"

	"github.com/riverqueue/river"
)

// Deps is what the workers need from the rest of the app, each as the
// smallest interface the workers declare beside themselves: one struct, so a
// new job's dependency is a field here and not another parameter on Workers.
// app/application.go builds it.
type Deps struct {
	Store interface {
		confirmationStore
		welcomeStore
	}
	Mail    sender
	SiteURL string
}

// Workers is every worker the queue can run, and whether there is one. Each
// is registered on one line above the marker, with the dependencies it
// declares; `bogie g job` inserts that line and `bogie d job` removes it.
//
// River refuses an empty bundle, and a new app has no jobs yet, so the second
// value lets app/application.go run insert-only until the first one exists.
func Workers(log *slog.Logger, d Deps) (*river.Workers, bool) {
	workers := river.NewWorkers()
	registered := 0
	registered += add(workers, &SendOrderConfirmationWorker{Log: log, Store: d.Store, Mail: d.Mail})
	registered += add(workers, &SendWelcomeEmailWorker{Log: log, Store: d.Store, Mail: d.Mail, SiteURL: d.SiteURL})
	// bogie:jobs
	return workers, registered > 0
}

// add registers one worker and counts it.
func add[T river.JobArgs](workers *river.Workers, worker river.Worker[T]) int {
	river.AddWorker(workers, worker)
	return 1
}
