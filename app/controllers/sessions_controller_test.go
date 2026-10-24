package controllers_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers"
	"ecommerce/app/domain"
	"ecommerce/app/services/cartmerge"
	"ecommerce/app/services/password"
)

// fakeSessionsStore is a fake of the store interface declared beside the
// controller: it records what the controller asked for and answers with
// what the test chose.
type fakeSessionsStore struct {
	user       domain.User
	userErr    error
	oldCart    domain.Cart
	oldCartErr error
	lines      []domain.CartLine
	newCart    domain.Cart
	addErr     error
	deleteErr  error

	gotOldCartToken      string
	gotCartByTokenCalled bool
	gotNewCartSubject    string
	addedProductIDs      []int64
	addedQuantities      []int32
	deleteCalled         bool
	deletedCartID        int64
}

func (f *fakeSessionsStore) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	return f.user, f.userErr
}

func (f *fakeSessionsStore) GetCartByToken(_ context.Context, token string) (domain.Cart, error) {
	f.gotCartByTokenCalled = true
	f.gotOldCartToken = token
	return f.oldCart, f.oldCartErr
}

func (f *fakeSessionsStore) FindOrCreateCartByToken(_ context.Context, token string) (domain.Cart, error) {
	f.gotNewCartSubject = token
	return f.newCart, nil
}

func (f *fakeSessionsStore) CartContents(_ context.Context, cartID int64) ([]domain.CartLine, error) {
	return f.lines, nil
}

func (f *fakeSessionsStore) AddCartItem(_ context.Context, cartID, productID int64, quantity int32) (domain.CartItem, error) {
	f.addedProductIDs = append(f.addedProductIDs, productID)
	f.addedQuantities = append(f.addedQuantities, quantity)
	return domain.CartItem{}, f.addErr
}

func (f *fakeSessionsStore) DeleteCartAndItems(_ context.Context, cartID int64) error {
	f.deleteCalled = true
	f.deletedCartID = cartID
	return f.deleteErr
}

// userWithPassword builds a user whose digest checks out against plain.
func userWithPassword(id int64, plain string) domain.User {
	digest, err := password.Hash(plain)
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
	return domain.User{ID: id, Email: "a@example.com", PasswordDigest: digest}
}

// fakeAlerts records the sign-in alerts a login asked for.
type fakeAlerts struct {
	err  error
	sent []domain.SignIn
}

func (f *fakeAlerts) SignedIn(_ context.Context, s domain.SignIn) error {
	f.sent = append(f.sent, s)
	return f.err
}

func sessionsRequest(store *fakeSessionsStore, body, authorization string) *httptest.ResponseRecorder {
	return sessionsRequestWithAlerts(store, &fakeAlerts{}, body, authorization)
}

// sessionsRequestWithAlerts runs the real cart merge over the fake store, so
// the merge behaviors below are tested through the controller as before.
func sessionsRequestWithAlerts(store *fakeSessionsStore, alerts *fakeAlerts, body, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	merger := cartmerge.Merger{Store: store, Tokens: testTokens, Log: log}
	controllers.NewSessionsController(store, testTokens, merger, alerts, log).SetupRoutes(&router.RouterGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	req.RemoteAddr = "203.0.113.9:4321"
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("POST /sessions", func() {
	const loginBody = `{"email":"a@example.com","password":"correct-password"}`

	var store *fakeSessionsStore

	BeforeEach(func() {
		store = &fakeSessionsStore{}
	})

	It("is mounted", func() {
		rec := sessionsRequest(store, `{"email":"a@example.com","password":"wrong"}`, "")
		Expect(rec.Code).NotTo(Equal(http.StatusNotFound))
	})

	It("answers 400 for malformed JSON", func() {
		Expect(sessionsRequest(store, `{"nope":`, "").Code).To(Equal(http.StatusBadRequest))
	})

	Context("with a known user", func() {
		BeforeEach(func() {
			store.user = userWithPassword(7, "correct-password")
		})

		It("mints a token on success", func() {
			rec := sessionsRequest(store, loginBody, "")

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(rec.Body.String()).To(ContainSubstring(`"token":"`))
		})

		// A wrong password and an unknown email answer identically: a caller can't
		// tell the two apart.
		It("answers 401 for a wrong password", func() {
			rec := sessionsRequest(store, `{"email":"a@example.com","password":"wrong-password"}`, "")
			Expect(rec.Code).To(Equal(http.StatusUnauthorized), rec.Body.String())
		})

		Describe("merging a guest cart", func() {
			// No bearer token sent with the login: nothing to merge, and the store is
			// never even asked about an old cart.
			It("is skipped with no existing token", func() {
				rec := sessionsRequest(store, loginBody, "")

				Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
				Expect(store.gotCartByTokenCalled).To(BeFalse(), "GetCartByToken was called with no existing token sent")
			})

			// A garbage or expired bearer token just means nothing to merge, not a
			// failed login.
			It("is skipped with an invalid existing token", func() {
				rec := sessionsRequest(store, loginBody, "Bearer garbage")

				Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
				Expect(store.gotCartByTokenCalled).To(BeFalse(), "the old token didn't verify")
			})

			// A valid old token naming a cart that doesn't exist: still nothing to
			// merge or delete.
			It("is skipped when no guest cart exists", func() {
				store.oldCartErr = domain.ErrNotFound

				rec := sessionsRequest(store, loginBody, bearerFor("guest-abc"))

				Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
				Expect(store.deleteCalled).To(BeFalse(), "there was no cart to delete")
			})

			Context("with a guest cart that has lines", func() {
				BeforeEach(func() {
					store.oldCart = domain.Cart{ID: 5}
					store.newCart = domain.Cart{ID: 99}
				})

				// The real case: a guest cart with lines in it is folded into the newly
				// logged-in user's cart, then removed.
				It("folds the lines into the user's cart, then removes the guest cart", func() {
					store.lines = []domain.CartLine{
						{ProductID: 10, Quantity: 2},
						{ProductID: 11, Quantity: 1},
					}

					rec := sessionsRequest(store, loginBody, bearerFor("guest-abc"))

					Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
					Expect(store.gotOldCartToken).To(Equal("guest-abc"))
					Expect(store.gotNewCartSubject).To(Equal(strconv.FormatInt(7, 10)))
					Expect(store.addedProductIDs).To(Equal([]int64{10, 11}))
					Expect(store.addedQuantities).To(Equal([]int32{2, 1}))
					Expect(store.deleteCalled).To(BeTrue())
					Expect(store.deletedCartID).To(Equal(int64(5)))
				})

				// One line failing to merge (out of stock, say) doesn't fail the login, and
				// the guest cart is still cleaned up afterward.
				It("continues after a line fails to merge", func() {
					store.lines = []domain.CartLine{{ProductID: 10, Quantity: 999}}
					store.addErr = errors.Join(domain.ErrInvalid, errors.New("only 1 in stock"))

					rec := sessionsRequest(store, loginBody, bearerFor("guest-abc"))

					Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
					Expect(store.deleteCalled).To(BeTrue(), "the guest cart is still cleaned up even though its one line couldn't merge")
				})
			})

			// A real store error during the merge (not just "nothing to merge") is a
			// 500 that says nothing about its cause.
			It("answers 500 without detail for an unexpected merge error", func() {
				store.oldCartErr = errors.New("connection refused to db-1")

				rec := sessionsRequest(store, loginBody, bearerFor("guest-abc"))

				Expect(rec.Code).To(Equal(http.StatusInternalServerError), rec.Body.String())
				Expect(rec.Body.String()).NotTo(ContainSubstring("db-1"))
			})
		})

		Describe("sign-in alerts", func() {
			var alerts *fakeAlerts

			BeforeEach(func() {
				alerts = &fakeAlerts{}
			})

			// A correct login queues one alert, with who, when and from where, taken from
			// the request itself.
			It("queues one alert for a correct login, taken from the request", func() {
				rec := sessionsRequestWithAlerts(store, alerts, loginBody, "")

				Expect(rec.Code).To(Equal(http.StatusOK))
				Expect(alerts.sent).To(HaveLen(1))
				got := alerts.sent[0]
				Expect(got.UserID).To(Equal(int64(7)))
				Expect(got.IP).To(Equal("203.0.113.9"))
				Expect(got.UserAgent).To(Equal("TestBrowser/1.0"))
				Expect(got.At).NotTo(BeZero())
			})

			// A correct password is never refused because the queue is down.
			It("still logs in when the alert cannot be queued", func() {
				alerts.err = errors.New("queue is down")

				rec := sessionsRequestWithAlerts(store, alerts, loginBody, "")

				Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
				Expect(rec.Body.String()).To(ContainSubstring(`"token"`))
			})
		})
	})

	Context("with an unknown email", func() {
		BeforeEach(func() {
			store.userErr = domain.ErrNotFound
		})

		It("answers 401", func() {
			rec := sessionsRequest(store, `{"email":"nobody@example.com","password":"whatever1"}`, "")
			Expect(rec.Code).To(Equal(http.StatusUnauthorized), rec.Body.String())
		})
	})

	// Only a login that worked is announced: a wrong password, an unknown email
	// and a bad body say nothing to anyone.
	It("alerts only on success", func() {
		alerts := &fakeAlerts{}
		sessionsRequestWithAlerts(&fakeSessionsStore{user: userWithPassword(7, "correct-password")}, alerts, `{"email":"a@example.com","password":"wrong"}`, "")
		sessionsRequestWithAlerts(&fakeSessionsStore{userErr: domain.ErrNotFound}, alerts, `{"email":"x@example.com","password":"whatever1"}`, "")
		sessionsRequestWithAlerts(&fakeSessionsStore{}, alerts, `{"nope":`, "")
		Expect(alerts.sent).To(BeEmpty())
	})
})
