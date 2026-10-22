package controllers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

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
func userWithPassword(t *testing.T, id int64, plain string) domain.User {
	t.Helper()
	digest, err := password.Hash(plain)
	if err != nil {
		t.Fatal(err)
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

func sessionsRequest(t *testing.T, store *fakeSessionsStore, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	return sessionsRequestWithAlerts(t, store, &fakeAlerts{}, body, authorization)
}

// sessionsRequestWithAlerts runs the real cart merge over the fake store, so
// the merge behaviors below are tested through the controller as before.
func sessionsRequestWithAlerts(t *testing.T, store *fakeSessionsStore, alerts *fakeAlerts, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	merger := cartmerge.Merger{Store: store, Tokens: testTokens, Log: log}
	NewSessionsController(store, testTokens, merger, alerts, log).SetupRoutes(&router.RouterGroup)

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

func TestSessionsRoutesAreMounted(t *testing.T) {
	rec := sessionsRequest(t, &fakeSessionsStore{}, `{"email":"a@example.com","password":"wrong"}`, "")
	if rec.Code == http.StatusNotFound {
		t.Error("POST /sessions is not mounted")
	}
}

func TestSessionsCreateMintsATokenOnSuccess(t *testing.T) {
	store := &fakeSessionsStore{user: userWithPassword(t, 7, "correct-password")}
	rec := sessionsRequest(t, store, `{"email":"a@example.com","password":"correct-password"}`, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"token":"`) {
		t.Errorf("body = %s, want a token", rec.Body)
	}
}

// A wrong password and an unknown email answer identically: a caller can't
// tell the two apart.
func TestSessionsCreateWrongPasswordIs401(t *testing.T) {
	store := &fakeSessionsStore{user: userWithPassword(t, 7, "correct-password")}
	rec := sessionsRequest(t, store, `{"email":"a@example.com","password":"wrong-password"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401: %s", rec.Code, rec.Body)
	}
}

func TestSessionsCreateUnknownEmailIs401(t *testing.T) {
	store := &fakeSessionsStore{userErr: domain.ErrNotFound}
	rec := sessionsRequest(t, store, `{"email":"nobody@example.com","password":"whatever1"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401: %s", rec.Code, rec.Body)
	}
}

func TestSessionsCreateMalformedJSONIs400(t *testing.T) {
	rec := sessionsRequest(t, &fakeSessionsStore{}, `{"nope":`, "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// No bearer token sent with the login: nothing to merge, and the store is
// never even asked about an old cart.
func TestSessionsCreateSkipsMergeWithNoExistingToken(t *testing.T) {
	store := &fakeSessionsStore{user: userWithPassword(t, 7, "correct-password")}
	rec := sessionsRequest(t, store, `{"email":"a@example.com","password":"correct-password"}`, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.gotCartByTokenCalled {
		t.Error("GetCartByToken was called with no existing token sent")
	}
}

// A garbage or expired bearer token just means nothing to merge, not a
// failed login.
func TestSessionsCreateSkipsMergeWithAnInvalidExistingToken(t *testing.T) {
	store := &fakeSessionsStore{user: userWithPassword(t, 7, "correct-password")}
	rec := sessionsRequest(t, store, `{"email":"a@example.com","password":"correct-password"}`, "Bearer garbage")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.gotCartByTokenCalled {
		t.Error("GetCartByToken was called even though the old token didn't verify")
	}
}

// A valid old token naming a cart that doesn't exist: still nothing to
// merge or delete.
func TestSessionsCreateSkipsMergeWhenNoGuestCartExists(t *testing.T) {
	store := &fakeSessionsStore{user: userWithPassword(t, 7, "correct-password"), oldCartErr: domain.ErrNotFound}
	auth := bearerFor(t, "guest-abc")
	rec := sessionsRequest(t, store, `{"email":"a@example.com","password":"correct-password"}`, auth)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.deleteCalled {
		t.Error("DeleteCartAndItems was called even though there was no cart to delete")
	}
}

// The real case: a guest cart with lines in it is folded into the newly
// logged-in user's cart, then removed.
func TestSessionsCreateMergesTheGuestCart(t *testing.T) {
	store := &fakeSessionsStore{
		user:    userWithPassword(t, 7, "correct-password"),
		oldCart: domain.Cart{ID: 5},
		lines: []domain.CartLine{
			{ProductID: 10, Quantity: 2},
			{ProductID: 11, Quantity: 1},
		},
		newCart: domain.Cart{ID: 99},
	}
	auth := bearerFor(t, "guest-abc")
	rec := sessionsRequest(t, store, `{"email":"a@example.com","password":"correct-password"}`, auth)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if store.gotOldCartToken != "guest-abc" {
		t.Errorf("store looked up cart by token %q, want guest-abc", store.gotOldCartToken)
	}
	if store.gotNewCartSubject != strconv.FormatInt(7, 10) {
		t.Errorf("store resolved the new cart by subject %q, want \"7\"", store.gotNewCartSubject)
	}
	if len(store.addedProductIDs) != 2 || store.addedProductIDs[0] != 10 || store.addedProductIDs[1] != 11 {
		t.Errorf("store got product ids %v, want [10 11]", store.addedProductIDs)
	}
	if len(store.addedQuantities) != 2 || store.addedQuantities[0] != 2 || store.addedQuantities[1] != 1 {
		t.Errorf("store got quantities %v, want [2 1]", store.addedQuantities)
	}
	if !store.deleteCalled || store.deletedCartID != 5 {
		t.Errorf("deleteCalled = %v, deletedCartID = %d, want true and 5", store.deleteCalled, store.deletedCartID)
	}
}

// One line failing to merge (out of stock, say) doesn't fail the login, and
// the guest cart is still cleaned up afterward.
func TestSessionsCreateContinuesMergeAfterALineFails(t *testing.T) {
	store := &fakeSessionsStore{
		user:    userWithPassword(t, 7, "correct-password"),
		oldCart: domain.Cart{ID: 5},
		lines:   []domain.CartLine{{ProductID: 10, Quantity: 999}},
		newCart: domain.Cart{ID: 99},
		addErr:  errors.Join(domain.ErrInvalid, errors.New("only 1 in stock")),
	}
	auth := bearerFor(t, "guest-abc")
	rec := sessionsRequest(t, store, `{"email":"a@example.com","password":"correct-password"}`, auth)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if !store.deleteCalled {
		t.Error("the guest cart should still be cleaned up even though its one line couldn't merge")
	}
}

// A real store error during the merge (not just "nothing to merge") is a
// 500 that says nothing about its cause.
func TestSessionsCreateUnexpectedMergeErrorIs500WithoutDetail(t *testing.T) {
	store := &fakeSessionsStore{
		user:       userWithPassword(t, 7, "correct-password"),
		oldCartErr: errors.New("connection refused to db-1"),
	}
	auth := bearerFor(t, "guest-abc")
	rec := sessionsRequest(t, store, `{"email":"a@example.com","password":"correct-password"}`, auth)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500: %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "db-1") {
		t.Errorf("body leaks the internal error: %s", rec.Body)
	}
}

// A correct login queues one alert, with who, when and from where, taken from
// the request itself.
func TestSessionsCreateQueuesASignInAlert(t *testing.T) {
	store := &fakeSessionsStore{user: userWithPassword(t, 7, "correct-password")}
	alerts := &fakeAlerts{}
	rec := sessionsRequestWithAlerts(t, store, alerts, `{"email":"a@example.com","password":"correct-password"}`, "")

	if rec.Code != http.StatusOK || len(alerts.sent) != 1 {
		t.Fatalf("status %d, %d alerts; want 200 and one alert", rec.Code, len(alerts.sent))
	}
	got := alerts.sent[0]
	if got.UserID != 7 || got.IP != "203.0.113.9" || got.UserAgent != "TestBrowser/1.0" || got.At.IsZero() {
		t.Errorf("alert = %+v, want user 7 from 203.0.113.9 with the request's browser and a time", got)
	}
}

// Only a login that worked is announced: a wrong password, an unknown email
// and a bad body say nothing to anyone.
func TestSessionsCreateAlertsOnlyOnSuccess(t *testing.T) {
	alerts := &fakeAlerts{}
	sessionsRequestWithAlerts(t, &fakeSessionsStore{user: userWithPassword(t, 7, "correct-password")}, alerts, `{"email":"a@example.com","password":"wrong"}`, "")
	sessionsRequestWithAlerts(t, &fakeSessionsStore{userErr: domain.ErrNotFound}, alerts, `{"email":"x@example.com","password":"whatever1"}`, "")
	sessionsRequestWithAlerts(t, &fakeSessionsStore{}, alerts, `{"nope":`, "")
	if len(alerts.sent) != 0 {
		t.Errorf("alerts = %+v, want none", alerts.sent)
	}
}

// A correct password is never refused because the queue is down.
func TestSessionsCreateStillLogsInWhenTheAlertCannotBeQueued(t *testing.T) {
	store := &fakeSessionsStore{user: userWithPassword(t, 7, "correct-password")}
	alerts := &fakeAlerts{err: errors.New("queue is down")}
	rec := sessionsRequestWithAlerts(t, store, alerts, `{"email":"a@example.com","password":"correct-password"}`, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"token"`) {
		t.Errorf("status %d: %s; want a login despite the failed alert", rec.Code, rec.Body)
	}
}
