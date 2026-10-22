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
)

type fakeWelcomeStore struct {
	user domain.User
	err  error
}

func (f fakeWelcomeStore) GetUser(context.Context, string) (domain.User, error) { return f.user, f.err }

func welcome(t *testing.T, store fakeWelcomeStore, mail *fakeSender) error {
	t.Helper()
	w := &SendWelcomeEmailWorker{
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Store: store, Mail: mail, SiteURL: "https://shop.example.com",
	}
	job := &river.Job[SendWelcomeEmailArgs]{JobRow: &rivertype.JobRow{ID: 1}, Args: SendWelcomeEmailArgs{UserID: 3}}
	return w.Work(context.Background(), job)
}

func TestWelcomeWorkSendsTheEmailWithTheShopLink(t *testing.T) {
	mail := &fakeSender{}
	if err := welcome(t, fakeWelcomeStore{user: domain.User{ID: 3, Email: "ann@example.com"}}, mail); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 1 || mail.sent[0].To != "ann@example.com" || !strings.Contains(mail.sent[0].Text, "https://shop.example.com") {
		t.Errorf("sent = %+v, want one welcome with the shop's address", mail.sent)
	}
}

func TestWelcomeWorkRetriesASendFailureAndCancelsAGoneUser(t *testing.T) {
	boom := errors.New("connection refused")
	var cancel *rivertype.JobCancelError

	err := welcome(t, fakeWelcomeStore{user: domain.User{ID: 3, Email: "a@example.com"}}, &fakeSender{err: boom})
	if !errors.Is(err, boom) || errors.As(err, &cancel) {
		t.Errorf("send failure: err = %v, want the plain error River retries", err)
	}

	mail := &fakeSender{}
	err = welcome(t, fakeWelcomeStore{err: domain.ErrNotFound}, mail)
	if !errors.As(err, &cancel) || len(mail.sent) != 0 {
		t.Errorf("gone user: err = %v, sent %d; want a cancel and no email", err, len(mail.sent))
	}
}
