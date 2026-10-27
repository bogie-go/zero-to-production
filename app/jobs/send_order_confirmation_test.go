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

	"ecommerce/app/jobs"
	"ecommerce/app/models"
	"ecommerce/app/services/mailer"
)

type fakeConfirmationStore struct {
	user     models.User
	userErr  error
	order    models.Order
	lines    []models.OrderLine
	orderErr error

	gotUserID  int64
	gotOrderID string
}

func (f *fakeConfirmationStore) GetUser(_ context.Context, id string) (models.User, error) {
	return f.user, f.userErr
}

func (f *fakeConfirmationStore) GetOrderForUser(_ context.Context, userID int64, id string) (models.Order, []models.OrderLine, error) {
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
		user:  models.User{ID: 3, Email: "ann@example.com"},
		order: models.Order{ID: 7, SubtotalCents: 500, TotalCents: 500},
		lines: []models.OrderLine{{ProductName: "Widget", UnitPriceCents: 500, Quantity: 1}},
	}
}

var _ = Describe("SendOrderConfirmationWorker", func() {
	var (
		store  *fakeConfirmationStore
		mail   *fakeSender
		cancel *rivertype.JobCancelError
		err    error
	)

	BeforeEach(func() {
		store, mail = goodStore(), &fakeSender{}
		cancel = nil
	})

	Context("when the order and user exist", func() {
		BeforeEach(func() {
			err = work(store, mail)
		})

		It("sends the confirmation", func() {
			Expect(err).To(Succeed())
			Expect(mail.sent).To(HaveLen(1))
			Expect(mail.sent[0].To).To(Equal("ann@example.com"))
			Expect(mail.sent[0].Subject).To(ContainSubstring("#7"))
			Expect(store.gotUserID).To(BeEquivalentTo(3))
			Expect(store.gotOrderID).To(Equal("7"))
		})
	})

	// A mail server that is down is River's to retry: the error comes back as is.
	Context("when the mail server is down", func() {
		var boom error

		BeforeEach(func() {
			boom = errors.New("connection refused")
			mail.err = boom
			err = work(store, mail)
		})

		It("returns the send failure so River retries", func() {
			Expect(err).To(MatchError(boom))
			Expect(errors.As(err, &cancel)).To(BeFalse(), "a send failure cancelled the job; it should be retried")
		})
	})

	// An order that is gone can never be emailed: cancel, do not retry 25 times.
	Context("when the order is gone", func() {
		BeforeEach(func() {
			store.orderErr = models.ErrNotFound
			err = work(store, mail)
		})

		It("cancels without sending", func() {
			Expect(errors.As(err, &cancel)).To(BeTrue(), "err = %v, want a cancel", err)
			Expect(mail.sent).To(BeEmpty())
		})
	})

	Context("when the user is gone", func() {
		BeforeEach(func() {
			store.userErr = models.ErrNotFound
			err = work(store, mail)
		})

		It("cancels without sending", func() {
			Expect(errors.As(err, &cancel)).To(BeTrue(), "err = %v, want a cancel", err)
			Expect(mail.sent).To(BeEmpty())
		})
	})

	// Any other store error is transient: retried, not cancelled.
	Context("with a database error", func() {
		BeforeEach(func() {
			store.userErr = errors.New("database is restarting")
			err = work(store, mail)
		})

		It("retries instead of cancelling", func() {
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &cancel)).To(BeFalse())
		})
	})
})
