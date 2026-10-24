package jobs_test

import (
	"context"
	"errors"
	"io"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"ecommerce/app/domain"
	"ecommerce/app/jobs"
)

type fakeWelcomeStore struct {
	user domain.User
	err  error
}

func (f fakeWelcomeStore) GetUser(context.Context, string) (domain.User, error) { return f.user, f.err }

func welcome(store fakeWelcomeStore, mail *fakeSender) error {
	w := &jobs.SendWelcomeEmailWorker{
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Store: store, Mail: mail, SiteURL: "https://shop.example.com",
	}
	job := &river.Job[jobs.SendWelcomeEmailArgs]{JobRow: &rivertype.JobRow{ID: 1}, Args: jobs.SendWelcomeEmailArgs{UserID: 3}}
	return w.Work(context.Background(), job)
}

var _ = Describe("SendWelcomeEmailWorker", func() {
	var (
		mail   *fakeSender
		cancel *rivertype.JobCancelError
	)

	BeforeEach(func() {
		mail = &fakeSender{}
		cancel = nil
	})

	It("sends the email with the shop link", func() {
		Expect(welcome(fakeWelcomeStore{user: domain.User{ID: 3, Email: "ann@example.com"}}, mail)).To(Succeed())
		Expect(mail.sent).To(HaveLen(1))
		Expect(mail.sent[0].To).To(Equal("ann@example.com"))
		Expect(mail.sent[0].Text).To(ContainSubstring("https://shop.example.com"))
	})

	It("retries a send failure", func() {
		boom := errors.New("connection refused")
		mail.err = boom
		err := welcome(fakeWelcomeStore{user: domain.User{ID: 3, Email: "a@example.com"}}, mail)
		Expect(err).To(MatchError(boom))
		Expect(errors.As(err, &cancel)).To(BeFalse())
	})

	It("cancels for a gone user", func() {
		err := welcome(fakeWelcomeStore{err: domain.ErrNotFound}, mail)
		Expect(errors.As(err, &cancel)).To(BeTrue())
		Expect(mail.sent).To(BeEmpty())
	})
})
