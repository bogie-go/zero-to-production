package requests_test

import (
	"net/http"
	"testing"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
	"ecommerce/test/testapp"
)

func TestSignUpThenLogInThenWhoAmI(t *testing.T) {
	app := testapp.New(t)
	creds := views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}

	var signup views.SessionResponse
	app.Post("/users", creds).Do().Status(http.StatusCreated).JSON(&signup)

	var login views.SessionResponse
	app.Post("/sessions", views.LoginRequest(creds)).Do().Status(http.StatusOK).JSON(&login)
	if login.Token == "" {
		t.Fatal("login returned no token")
	}
	if app.Jobs("send_sign_in_alert") != 1 {
		t.Errorf("sign-in alerts queued = %d, want 1", app.Jobs("send_sign_in_alert"))
	}

	var me views.MeResponse
	app.Get("/me").WithAuthorization("Bearer " + login.Token).Do().Status(http.StatusOK).JSON(&me)
	if me.Email != "ada@example.com" || me.Role != domain.RoleCustomer {
		t.Errorf("me = %+v", me)
	}
}

func TestSignUpRefusesBadInput(t *testing.T) {
	app := testapp.New(t)
	good := views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}
	app.Post("/users", good).Do().Status(http.StatusCreated)

	app.Post("/users", good).Do().Status(http.StatusUnprocessableEntity).Contains("already")
	app.Post("/users", views.SignupRequest{Email: "not-an-email", Password: "correct-horse-battery"}).Do().Status(http.StatusUnprocessableEntity)
	app.Post("/users", views.SignupRequest{Email: "grace@example.com", Password: "short"}).Do().Status(http.StatusUnprocessableEntity)
	app.Post("/users", "{not json").Do().Status(http.StatusBadRequest)
}

// Signing up can never make a staff account, whatever the body says.
func TestSignUpCannotChooseTheRole(t *testing.T) {
	app := testapp.New(t)
	var session views.SessionResponse
	app.Post("/users", `{"email":"mallory@example.com","password":"correct-horse-battery","role":"staff"}`).
		Do().Status(http.StatusCreated).JSON(&session)

	app.Get("/admin/products").WithAuthorization("Bearer " + session.Token).Do().Status(http.StatusForbidden)
}

// A wrong password and an unknown email answer the same, so a caller cannot
// tell which accounts exist.
func TestLoginDoesNotSayWhichPartWasWrong(t *testing.T) {
	app := testapp.New(t)
	app.Post("/users", views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusCreated)

	wrong := app.Post("/sessions", views.LoginRequest{Email: "ada@example.com", Password: "wrong-password-here"}).Do().Status(http.StatusUnauthorized)
	unknown := app.Post("/sessions", views.LoginRequest{Email: "nobody@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusUnauthorized)
	if string(wrong.Body) != string(unknown.Body) {
		t.Errorf("a wrong password says %s but an unknown email says %s", wrong.Body, unknown.Body)
	}
	if app.Jobs("send_sign_in_alert") != 0 {
		t.Error("a failed login queued a sign-in alert")
	}
}

func TestLoggingInKeepsTheGuestCart(t *testing.T) {
	app := testapp.New(t)
	app.Post("/users", views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusCreated)
	product := factories.Product(t, app.Store)

	var guest views.GuestSessionResponse
	app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 2}).
		WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated)

	var login views.SessionResponse
	app.Post("/sessions", views.LoginRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).
		WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusOK).JSON(&login)

	var cart views.CartResponse
	app.Get("/cart").WithAuthorization("Bearer " + login.Token).Do().JSON(&cart)
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 2 {
		t.Errorf("cart after login = %+v, want the guest's 2 items", cart)
	}
}

func TestProtectedEndpointsNeedAToken(t *testing.T) {
	app := testapp.New(t)
	for _, path := range []string{"/me", "/cart", "/orders"} {
		app.Get(path).Do().Status(http.StatusUnauthorized)
		app.Get(path).WithAuthorization("Bearer garbage").Do().Status(http.StatusUnauthorized)
	}
	app.Post("/orders", nil).Do().Status(http.StatusUnauthorized)
}
