package controllers_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers"
	"ecommerce/app/domain"
)

// fakeUsersStore is a fake of the store interface declared beside the
// controller: it records what the controller asked for and answers with
// what the test chose.
type fakeUsersStore struct {
	user domain.User
	err  error

	gotUser domain.User
}

func (f *fakeUsersStore) RegisterUser(_ context.Context, u domain.User) (domain.User, error) {
	f.gotUser = u
	if f.err != nil {
		return domain.User{}, f.err
	}
	return f.user, nil
}

// fakeMerger records the cart merge a signup asked for; the merge itself is
// tested in cartmerge's own behaviors, through the sessions controller.
type fakeMerger struct {
	err           error
	called        bool
	gotAuthHeader string
	gotNewSubject string
}

func (f *fakeMerger) FromAuthorization(_ context.Context, authorization, newSubject string) error {
	f.called, f.gotAuthHeader, f.gotNewSubject = true, authorization, newSubject
	return f.err
}

func usersRequest(store *fakeUsersStore, body string) *httptest.ResponseRecorder {
	return usersRequestWithMerger(store, &fakeMerger{}, body, "")
}

func usersRequestWithMerger(store *fakeUsersStore, merger *fakeMerger, body, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controllers.NewUsersController(store, testTokens, merger, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(&router.RouterGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("POST /users", func() {
	const validBody = `{"email":"a@example.com","password":"at-least-8"}`

	var store *fakeUsersStore

	BeforeEach(func() {
		store = &fakeUsersStore{}
	})

	It("is mounted", func() {
		Expect(usersRequest(store, validBody).Code).NotTo(Equal(http.StatusNotFound))
	})

	// A signup hashes the password (never stores it plain), creates the
	// account, and mints a bearer token immediately — no separate login needed
	// right after registering.
	It("hashes the password and mints a token", func() {
		store.user = domain.User{ID: 7, Email: "a@example.com"}

		rec := usersRequest(store, validBody)

		Expect(rec.Code).To(Equal(http.StatusCreated), rec.Body.String())
		Expect(rec.Body.String()).To(ContainSubstring(`"token":"`))
		Expect(store.gotUser.Email).To(Equal("a@example.com"))
		Expect(store.gotUser.PasswordDigest).NotTo(BeEmpty())
		Expect(store.gotUser.PasswordDigest).NotTo(Equal("at-least-8"), "a bcrypt digest, not the plain password")
	})

	// A password too short to hash never reaches the store at all.
	It("answers 422 for a short password, without calling the store", func() {
		rec := usersRequest(store, `{"email":"a@example.com","password":"short"}`)

		Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
		Expect(store.gotUser.Email).To(BeEmpty(), "the store was called even though the password was invalid")
	})

	// An email already taken (the store's :uniq violation, mapped to
	// ErrInvalid) is 422, same as any other validation failure.
	It("answers 422 for a duplicate email", func() {
		store.err = errors.Join(domain.ErrInvalid, errors.New("email is already taken"))

		rec := usersRequest(store, validBody)

		Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
	})

	It("answers 400 for malformed JSON", func() {
		Expect(usersRequest(store, `{"nope":`).Code).To(Equal(http.StatusBadRequest))
	})

	// An unexpected error is a 500 that says nothing about its cause.
	It("answers 500 without detail for an unexpected error", func() {
		store.err = errors.New("connection refused to db-1")

		rec := usersRequest(store, validBody)

		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		Expect(rec.Body.String()).NotTo(ContainSubstring("db-1"))
	})

	Describe("the guest cart", func() {
		BeforeEach(func() {
			store.user = domain.User{ID: 7, Email: "a@example.com"}
		})

		// Signing up as a guest with a cart keeps the cart: the request's own
		// Authorization header goes to the merge, for the new account.
		It("is merged into the new account", func() {
			merger := &fakeMerger{}

			rec := usersRequestWithMerger(store, merger, validBody, "Bearer guest-token")

			Expect(rec.Code).To(Equal(http.StatusCreated), rec.Body.String())
			Expect(merger.called).To(BeTrue())
			Expect(merger.gotAuthHeader).To(Equal("Bearer guest-token"))
			Expect(merger.gotNewSubject).To(Equal("7"))
		})

		// The account exists by then, so a merge that fails does not turn a
		// signup that worked into an error.
		It("does not fail the signup when the merge fails", func() {
			rec := usersRequestWithMerger(store, &fakeMerger{err: errors.New("db hiccup")}, validBody, "")

			Expect(rec.Code).To(Equal(http.StatusCreated))
			Expect(rec.Body.String()).To(ContainSubstring(`"token"`))
		})
	})
})
