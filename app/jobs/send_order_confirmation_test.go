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

func work(store *fakeConfirmationStore, mail *fakeSender) error {
	w := &jobs.SendOrderConfirmationWorker{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Store: store, Mail: mail}
	job := &river.Job[jobs.SendOrderConfirmationArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   jobs.SendOrderConfirmationArgs{UserID: 3, OrderID: 7},
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

var _ = Describe("SendOrderConfirmationWorker", func() {
	var (
		store  *fakeConfirmationStore
		mail   *fakeSender
		cancel *rivertype.JobCancelError
	)

	BeforeEach(func() {
		store, mail = goodStore(), &fakeSender{}
		cancel = nil
	})

	It("sends the confirmation", func() {
		Expect(work(store, mail)).To(Succeed())
		Expect(mail.sent).To(HaveLen(1))
		Expect(mail.sent[0].To).To(Equal("ann@example.com"))
		Expect(mail.sent[0].Subject).To(ContainSubstring("#7"))
		Expect(store.gotUserID).To(BeEquivalentTo(3))
		Expect(store.gotOrderID).To(Equal("7"))
	})

	// A mail server that is down is River's to retry: the error comes back as is.
	It("returns a send failure so River retries", func() {
		boom := errors.New("connection refused")
		mail.err = boom
		err := work(store, mail)
		Expect(err).To(MatchError(boom))
		Expect(errors.As(err, &cancel)).To(BeFalse(), "a send failure cancelled the job; it should be retried")
	})

	// An order that is gone can never be emailed: cancel, do not retry 25 times.
	DescribeTable("cancels when the order or user is gone",
		func(break_ func(*fakeConfirmationStore)) {
			break_(store)
			err := work(store, mail)
			Expect(errors.As(err, &cancel)).To(BeTrue(), "err = %v, want a cancel", err)
			Expect(mail.sent).To(BeEmpty())
		},
		Entry("order gone", func(s *fakeConfirmationStore) { s.orderErr = domain.ErrNotFound }),
		Entry("user gone", func(s *fakeConfirmationStore) { s.userErr = domain.ErrNotFound }),
	)

	// Any other store error is transient: retried, not cancelled.
	It("retries on a database error", func() {
		store.userErr = errors.New("database is restarting")
		err := work(store, mail)
		Expect(err).To(HaveOccurred())
		Expect(errors.As(err, &cancel)).To(BeFalse())
	})
})
