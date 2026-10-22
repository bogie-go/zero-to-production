package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"ecommerce/app/domain"
	"ecommerce/app/services/mailer"
)

type fakeConfirmationStore struct {
	user     domain.User
	userErr  error
	order    domain.Order
	lines    []domain.OrderLine
	orderErr error

	gotUserID  int64
	gotOrderID string
}

func (f *fakeConfirmationStore) GetUser(_ context.Context, id string) (domain.User, error) {
	return f.user, f.userErr
}

func (f *fakeConfirmationStore) GetOrderForUser(_ context.Context, userID int64, id string) (domain.Order, []domain.OrderLine, error) {
	f.gotUserID, f.gotOrderID = userID, id
	return f.order, f.lines, f.orderErr
}

type fakeSender struct {
	err  error
	sent []mailer.Message
}

func (f *fakeSender) Send(_ context.Context, msg mailer.Message) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, msg)
	return nil
}

func work(t *testing.T, store *fakeConfirmationStore, mail *fakeSender) error {
	t.Helper()
	w := &SendOrderConfirmationWorker{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Store: store, Mail: mail}
	job := &river.Job[SendOrderConfirmationArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   SendOrderConfirmationArgs{UserID: 3, OrderID: 7},
	}
	return w.Work(context.Background(), job)
}

func goodStore() *fakeConfirmationStore {
	return &fakeConfirmationStore{
		user:  domain.User{ID: 3, Email: "ann@example.com"},
		order: domain.Order{ID: 7, SubtotalCents: 500, TotalCents: 500},
		lines: []domain.OrderLine{{ProductName: "Widget", UnitPriceCents: 500, Quantity: 1}},
	}
}

func TestWorkSendsTheConfirmation(t *testing.T) {
	store, mail := goodStore(), &fakeSender{}
	if err := work(t, store, mail); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 1 || mail.sent[0].To != "ann@example.com" || !strings.Contains(mail.sent[0].Subject, "#7") {
		t.Errorf("sent = %+v, want one email to ann about order 7", mail.sent)
	}
	if store.gotUserID != 3 || store.gotOrderID != "7" {
		t.Errorf("order loaded for user %d, id %q; want user 3, order 7", store.gotUserID, store.gotOrderID)
	}
}

// A mail server that is down is River's to retry: the error comes back as is.
func TestWorkReturnsASendFailureSoRiverRetries(t *testing.T) {
	boom := errors.New("connection refused")
	err := work(t, goodStore(), &fakeSender{err: boom})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the send error", err)
	}
	var cancel *rivertype.JobCancelError
	if errors.As(err, &cancel) {
		t.Error("a send failure cancelled the job; it should be retried")
	}
}

// An order that is gone can never be emailed: cancel, do not retry 25 times.
func TestWorkCancelsWhenTheOrderOrUserIsGone(t *testing.T) {
	for name, store := range map[string]*fakeConfirmationStore{
		"order gone": func() *fakeConfirmationStore { s := goodStore(); s.orderErr = domain.ErrNotFound; return s }(),
		"user gone":  func() *fakeConfirmationStore { s := goodStore(); s.userErr = domain.ErrNotFound; return s }(),
	} {
		mail := &fakeSender{}
		err := work(t, store, mail)
		var cancel *rivertype.JobCancelError
		if !errors.As(err, &cancel) {
			t.Errorf("%s: err = %v, want a cancel", name, err)
		}
		if len(mail.sent) != 0 {
			t.Errorf("%s: an email was sent", name)
		}
	}
}

// Any other store error is transient: retried, not cancelled.
func TestWorkRetriesOnADatabaseError(t *testing.T) {
	store := goodStore()
	store.userErr = errors.New("database is restarting")
	err := work(t, store, &fakeSender{})
	var cancel *rivertype.JobCancelError
	if err == nil || errors.As(err, &cancel) {
		t.Errorf("err = %v, want a plain error River will retry", err)
	}
}
