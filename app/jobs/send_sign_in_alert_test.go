package jobs_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"ecommerce/app/jobs"
	"ecommerce/app/models"
)

func alert(store fakeWelcomeStore, mail *fakeSender) error {
	w := &jobs.SendSignInAlertWorker{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Store: store, Mail: mail}
	job := &river.Job[jobs.SendSignInAlertArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   jobs.SendSignInAlertArgs{UserID: 3, At: time.Date(2026, 10, 22, 14, 30, 0, 0, time.UTC), IP: "203.0.113.9", UserAgent: "TestBrowser/1.0"},
	}
	return w.Work(context.Background(), job)
}

type fakeQueue struct{ args []river.JobArgs }

func (f *fakeQueue) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	f.args = append(f.args, args)
	return &rivertype.JobInsertResult{}, nil
}

var _ = Describe("SendSignInAlertWorker", func() {
	var (
		mail   *fakeSender
		cancel *rivertype.JobCancelError
		err    error
	)

	BeforeEach(func() {
		mail = &fakeSender{}
		cancel = nil
	})

	Context("when the user exists", func() {
		BeforeEach(func() {
			err = alert(fakeWelcomeStore{user: models.User{ID: 3, Email: "ann@example.com"}}, mail)
		})

		It("sends who, when and from where", func() {
			Expect(err).To(Succeed())
			Expect(mail.sent).To(HaveLen(1))
			Expect(mail.sent[0].To).To(Equal("ann@example.com"))
			Expect(mail.sent[0].Text).To(ContainSubstring("203.0.113.9"))
			Expect(mail.sent[0].Text).To(ContainSubstring("TestBrowser/1.0"))
			Expect(mail.sent[0].Text).To(ContainSubstring("22 Oct 2026 14:30 UTC"))
		})
	})

	Context("when sending fails", func() {
		var boom error

		BeforeEach(func() {
			boom = errors.New("connection refused")
			mail.err = boom
			err = alert(fakeWelcomeStore{user: models.User{ID: 3, Email: "a@example.com"}}, mail)
		})

		It("returns the error so the job is retried", func() {
			Expect(err).To(MatchError(boom))
			Expect(errors.As(err, &cancel)).To(BeFalse())
		})
	})

	Context("when the user is gone", func() {
		BeforeEach(func() {
			err = alert(fakeWelcomeStore{err: models.ErrNotFound}, mail)
		})

		It("cancels without sending", func() {
			Expect(errors.As(err, &cancel)).To(BeTrue())
			Expect(mail.sent).To(BeEmpty())
		})
	})
})

// SignInAlerts turns a login into one job carrying what the request knew.
var _ = Describe("SignInAlerts", func() {
	var (
		q   *fakeQueue
		at  time.Time
		err error
	)

	Context("when a user signs in", func() {
		BeforeEach(func() {
			q = &fakeQueue{}
			at = time.Date(2026, 10, 22, 14, 30, 0, 0, time.UTC)
			err = (jobs.SignInAlerts{Queue: q}).SignedIn(context.Background(), models.SignIn{UserID: 3, At: at, IP: "1.2.3.4", UserAgent: "UA"})
		})

		It("enqueues one job with the request's facts", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(q.args).To(HaveLen(1))

			got, ok := q.args[0].(jobs.SendSignInAlertArgs)
			Expect(ok).To(BeTrue())
			Expect(got.UserID).To(BeEquivalentTo(3))
			Expect(got.IP).To(Equal("1.2.3.4"))
			Expect(got.UserAgent).To(Equal("UA"))
			Expect(got.At).To(BeTemporally("==", at))
		})
	})
})
