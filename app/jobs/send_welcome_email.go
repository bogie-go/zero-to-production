package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/riverqueue/river"

	"ecommerce/app/domain"
	"ecommerce/app/mailers"
)

// SendWelcomeEmailArgs is what a send_welcome_email job carries: the new
// user's id, and nothing else. The worker loads the user when it runs.
type SendWelcomeEmailArgs struct {
	UserID int64 `json:"user_id"`
}

// Kind names the job in the database. Never change it once jobs are queued;
// a queued job of the old kind would have no worker.
func (SendWelcomeEmailArgs) Kind() string { return "send_welcome_email" }

// welcomeStore is the slice of the store this worker uses, declared here
// beside its user (rule 2). *models.Store satisfies it.
type welcomeStore interface {
	GetUser(ctx context.Context, id string) (domain.User, error)
}

// SendWelcomeEmailWorker emails a new account its welcome. Like the order
// confirmation it is enqueued inside the transaction that creates what it is
// about (Store.RegisterUser, via Store.OnUserRegistered wired in
// app/application.go), so a signup that commits has a welcome email coming
// and one that rolls back has none.
type SendWelcomeEmailWorker struct {
	river.WorkerDefaults[SendWelcomeEmailArgs]
	Log     *slog.Logger
	Store   welcomeStore
	Mail    sender
	SiteURL string
}

// Work sends the email. A send failure is returned for River to retry with
// backoff; a user who no longer exists cancels the job, since no retry can
// help.
func (w *SendWelcomeEmailWorker) Work(ctx context.Context, job *river.Job[SendWelcomeEmailArgs]) error {
	user, err := w.Store.GetUser(ctx, strconv.FormatInt(job.Args.UserID, 10))
	if errors.Is(err, domain.ErrNotFound) {
		return river.JobCancel(fmt.Errorf("user %d no longer exists", job.Args.UserID))
	}
	if err != nil {
		return err
	}
	if err := w.Mail.Send(ctx, mailers.Welcome(user, w.SiteURL)); err != nil {
		return err
	}
	w.Log.InfoContext(ctx, "send_welcome_email: sent", "job", job.ID, "to", user.Email)
	return nil
}
