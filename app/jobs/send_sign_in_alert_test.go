package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"ecommerce/app/domain"
)

func alert(t *testing.T, store fakeWelcomeStore, mail *fakeSender) error {
	t.Helper()
	w := &SendSignInAlertWorker{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Store: store, Mail: mail}
	job := &river.Job[SendSignInAlertArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   SendSignInAlertArgs{UserID: 3, At: time.Date(2026, 10, 22, 14, 30, 0, 0, time.UTC), IP: "203.0.113.9", UserAgent: "TestBrowser/1.0"},
	}
	return w.Work(context.Background(), job)
}

func TestAlertWorkSendsWhoWhenAndFromWhere(t *testing.T) {
	mail := &fakeSender{}
	if err := alert(t, fakeWelcomeStore{user: domain.User{ID: 3, Email: "ann@example.com"}}, mail); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 1 || mail.sent[0].To != "ann@example.com" {
		t.Fatalf("sent = %+v, want one alert to ann", mail.sent)
	}
	for _, want := range []string{"203.0.113.9", "TestBrowser/1.0", "22 Oct 2026 14:30 UTC"} {
		if !strings.Contains(mail.sent[0].Text, want) {
			t.Errorf("alert lacks %q:\n%s", want, mail.sent[0].Text)
		}
	}
}

func TestAlertWorkRetriesASendFailureAndCancelsAGoneUser(t *testing.T) {
	boom := errors.New("connection refused")
	var cancel *rivertype.JobCancelError
	if err := alert(t, fakeWelcomeStore{user: domain.User{ID: 3, Email: "a@example.com"}}, &fakeSender{err: boom}); !errors.Is(err, boom) || errors.As(err, &cancel) {
		t.Errorf("send failure: err = %v, want the plain error River retries", err)
	}
	mail := &fakeSender{}
	if err := alert(t, fakeWelcomeStore{err: domain.ErrNotFound}, mail); !errors.As(err, &cancel) || len(mail.sent) != 0 {
		t.Errorf("gone user: err = %v, sent %d; want a cancel and no email", err, len(mail.sent))
	}
}

type fakeQueue struct{ args []river.JobArgs }

func (f *fakeQueue) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	f.args = append(f.args, args)
	return &rivertype.JobInsertResult{}, nil
}

// SignInAlerts turns a login into one job carrying what the request knew.
func TestSignInAlertsEnqueueOneJobWithTheRequestsFacts(t *testing.T) {
	q := &fakeQueue{}
	at := time.Date(2026, 10, 22, 14, 30, 0, 0, time.UTC)
	if err := (SignInAlerts{Queue: q}).SignedIn(context.Background(), domain.SignIn{UserID: 3, At: at, IP: "1.2.3.4", UserAgent: "UA"}); err != nil {
		t.Fatal(err)
	}
	if len(q.args) != 1 {
		t.Fatalf("%d jobs, want 1", len(q.args))
	}
	got, ok := q.args[0].(SendSignInAlertArgs)
	if !ok || got.UserID != 3 || got.IP != "1.2.3.4" || got.UserAgent != "UA" || !got.At.Equal(at) {
		t.Errorf("args = %+v", q.args[0])
	}
}
