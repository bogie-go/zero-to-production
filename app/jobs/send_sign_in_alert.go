package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"ecommerce/app/domain"
	"ecommerce/app/mailers"
)

// SendSignInAlertArgs is what a send_sign_in_alert job carries. Unlike the
// other jobs it carries more than an id: when, where from and with what is a
// fact about the request that is gone by the time the worker runs, and is not
// stored anywhere else.
type SendSignInAlertArgs struct {
	UserID    int64     `json:"user_id"`
	At        time.Time `json:"at"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

// Kind names the job in the database. Never change it once jobs are queued;
// a queued job of the old kind would have no worker.
func (SendSignInAlertArgs) Kind() string { return "send_sign_in_alert" }

// alertStore is the slice of the store this worker uses, declared here beside
// its user (rule 2). *models.Store satisfies it.
type alertStore interface {
	GetUser(ctx context.Context, id string) (domain.User, error)
}

// SendSignInAlertWorker emails an account's owner that someone signed in.
type SendSignInAlertWorker struct {
	river.WorkerDefaults[SendSignInAlertArgs]
	Log   *slog.Logger
	Store alertStore
	Mail  sender
}

// Work sends the email. A send failure is returned for River to retry (the
// alert still carries the original time); a user who no longer exists cancels
// the job.
func (w *SendSignInAlertWorker) Work(ctx context.Context, job *river.Job[SendSignInAlertArgs]) error {
	a := job.Args
	user, err := w.Store.GetUser(ctx, strconv.FormatInt(a.UserID, 10))
	if errors.Is(err, domain.ErrNotFound) {
		return river.JobCancel(fmt.Errorf("user %d no longer exists", a.UserID))
	}
	if err != nil {
		return err
	}
	msg := mailers.SignInAlert(user, domain.SignIn{UserID: a.UserID, At: a.At, IP: a.IP, UserAgent: a.UserAgent})
	if err := w.Mail.Send(ctx, msg); err != nil {
		return err
	}
	w.Log.InfoContext(ctx, "send_sign_in_alert: sent", "job", job.ID, "to", user.Email)
	return nil
}

// inserter is the slice of the River client SignInAlerts uses.
// *river.Client[pgx.Tx] satisfies it, so this package needs no pgx.
type inserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// SignInAlerts enqueues the alert. It is what SessionsController calls (as
// its signInAlerter), built in app/application.go with the River client. A
// plain Insert, not InsertTx: a login writes nothing, so there is no
// transaction for the job to ride, and the controller treats a failure to
// queue as a warning, never as a failed login.
type SignInAlerts struct{ Queue inserter }

// SignedIn queues the alert for one login.
func (a SignInAlerts) SignedIn(ctx context.Context, s domain.SignIn) error {
	_, err := a.Queue.Insert(ctx, SendSignInAlertArgs{UserID: s.UserID, At: s.At, IP: s.IP, UserAgent: s.UserAgent}, nil)
	return err
}
