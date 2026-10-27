package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/riverqueue/river"

	"ecommerce/app/mailers"
	"ecommerce/app/models"
	"ecommerce/app/services/mailer"
)

// SendOrderConfirmationArgs is what a send_order_confirmation job carries. It
// is stored as JSON in the job row, so it holds ids and nothing else: the
// worker loads the order when it runs, so an email always shows what was
// saved, not a copy made at enqueue time.
type SendOrderConfirmationArgs struct {
	UserID  int64 `json:"user_id"`
	OrderID int64 `json:"order_id"`
}

// Kind names the job in the database. Never change it once jobs are queued;
// a queued job of the old kind would have no worker.
func (SendOrderConfirmationArgs) Kind() string { return "send_order_confirmation" }

// confirmationStore is the slice of the store this worker uses, declared here
// beside its user. *adapter.Store satisfies it.
type confirmationStore interface {
	GetUser(ctx context.Context, id string) (models.User, error)
	GetOrderForUser(ctx context.Context, userID int64, id string) (models.Order, []models.OrderLine, error)
}

// sender is the slice of the mailer service this worker uses.
// *mailer.Mailer satisfies it.
type sender interface {
	Send(ctx context.Context, msg mailer.Message) error
}

// SendOrderConfirmationWorker emails a customer their order. It is enqueued
// by the checkout, inside the order's own transaction (app/application.go
// wires that to Store.OnOrderPlaced), so there is a job for exactly the
// orders that exist:
//
//	queue.InsertTx(ctx, tx, jobs.SendOrderConfirmationArgs{UserID: u, OrderID: o}, nil)
type SendOrderConfirmationWorker struct {
	river.WorkerDefaults[SendOrderConfirmationArgs]
	Log   *slog.Logger
	Store confirmationStore
	Mail  sender
}

// Work loads the order and sends the email. A failure to send (the mail
// server is down) is returned, so River retries it with backoff; the order is
// already placed either way. An order or user that is no longer there can
// never succeed, so that cancels the job instead of retrying it 25 times.
// Delivery is at-least-once: if the email goes out and River then fails to
// record it, the retry sends it again.
func (w *SendOrderConfirmationWorker) Work(ctx context.Context, job *river.Job[SendOrderConfirmationArgs]) error {
	args := job.Args
	user, err := w.Store.GetUser(ctx, strconv.FormatInt(args.UserID, 10))
	if errors.Is(err, models.ErrNotFound) {
		return river.JobCancel(fmt.Errorf("user %d no longer exists", args.UserID))
	}
	if err != nil {
		return err
	}
	order, lines, err := w.Store.GetOrderForUser(ctx, user.ID, strconv.FormatInt(args.OrderID, 10))
	if errors.Is(err, models.ErrNotFound) {
		return river.JobCancel(fmt.Errorf("order %d no longer exists", args.OrderID))
	}
	if err != nil {
		return err
	}

	if err := w.Mail.Send(ctx, mailers.OrderConfirmation(user, order, lines)); err != nil {
		return err
	}
	w.Log.InfoContext(ctx, "send_order_confirmation: sent", "job", job.ID, "order", order.ID, "to", user.Email)
	return nil
}
