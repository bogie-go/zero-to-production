package requests_test

import (
	"net/http"
	"strconv"
	"testing"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
	"ecommerce/test/testapp"
	"ecommerce/test/testdb"
)

// The whole customer journey, through HTTP only: a guest fills a cart, signs
// up (the cart comes along), checks out, and finds the order, the stock and
// the queued emails exactly as the UI relies on them.
func TestAGuestSignsUpAndChecksOut(t *testing.T) {
	app := testapp.New(t)
	product := factories.Product(t, app.Store, func(p *domain.Product) { p.PriceCents = 500; p.Stock = 10 })

	var guest views.GuestSessionResponse
	app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 3}).
		WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated)

	// A guest can look but not buy.
	app.Post("/orders", nil).WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusForbidden)

	var session views.SessionResponse
	app.Post("/users", views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).
		WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated).JSON(&session)
	if app.Jobs("send_welcome_email") != 1 {
		t.Errorf("welcome emails queued = %d, want 1", app.Jobs("send_welcome_email"))
	}

	var cart views.CartResponse
	app.Get("/cart").WithAuthorization("Bearer " + session.Token).Do().Status(http.StatusOK).JSON(&cart)
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 3 || cart.TotalCents != 1500 {
		t.Fatalf("the guest's cart did not follow the signup: %+v", cart)
	}

	var order views.OrderResponse
	app.Post("/orders", nil).WithAuthorization("Bearer " + session.Token).Do().Status(http.StatusCreated).JSON(&order)
	if order.TotalCents != 1500 || len(order.Items) != 1 || order.Items[0].UnitPriceCents != 500 {
		t.Errorf("order = %+v, want 3 x $5.00 snapshotted", order)
	}
	if app.Jobs("send_order_confirmation") != 1 {
		t.Errorf("confirmation emails queued = %d, want 1", app.Jobs("send_order_confirmation"))
	}

	left, err := app.Store.GetProduct(t.Context(), strconv.FormatInt(product.ID, 10))
	if err != nil || left.Stock != 7 {
		t.Errorf("stock = %d, %v; want 7", left.Stock, err)
	}
	app.Get("/cart").WithAuthorization("Bearer " + session.Token).Do().Status(http.StatusOK).JSON(&cart)
	if len(cart.Items) != 0 {
		t.Errorf("the cart still has %d lines after checkout", len(cart.Items))
	}
}

func TestCheckoutRefusesMoreThanIsInStockAndChangesNothing(t *testing.T) {
	app := testapp.New(t)
	product := factories.Product(t, app.Store, func(p *domain.Product) { p.Stock = 5 })
	user := factories.User(t, app.Store)
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 5}).AsUser(user).Do().Status(http.StatusCreated)
	// The cart refuses more than is in stock when adding, so the stock has to
	// be taken by someone else between adding and paying.
	if _, err := testdb.Pool(t).Exec(t.Context(), "UPDATE products SET stock = 2 WHERE id = $1", product.ID); err != nil {
		t.Fatal(err)
	}

	app.Post("/orders", nil).AsUser(user).Do().Status(http.StatusUnprocessableEntity)

	left, _ := app.Store.GetProduct(t.Context(), strconv.FormatInt(product.ID, 10))
	if left.Stock != 2 {
		t.Errorf("stock = %d after a refused checkout, want 2", left.Stock)
	}
	var orders []views.OrderSummaryResponse
	app.Get("/orders").AsUser(user).Do().Status(http.StatusOK).JSON(&orders)
	if len(orders) != 0 || app.Jobs("send_order_confirmation") != 0 {
		t.Errorf("a refused checkout left %d orders and %d emails behind", len(orders), app.Jobs("send_order_confirmation"))
	}
}

func TestACustomerSeesOnlyTheirOwnOrders(t *testing.T) {
	app := testapp.New(t)
	product := factories.Product(t, app.Store)
	ada, grace := factories.User(t, app.Store), factories.User(t, app.Store)
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsUser(ada).Do().Status(http.StatusCreated)
	var order views.OrderResponse
	app.Post("/orders", nil).AsUser(ada).Do().Status(http.StatusCreated).JSON(&order)

	app.Get("/orders/" + strconv.FormatInt(order.ID, 10)).AsUser(ada).Do().Status(http.StatusOK)
	app.Get("/orders/" + strconv.FormatInt(order.ID, 10)).AsUser(grace).Do().Status(http.StatusNotFound)
	app.Get("/orders").Do().Status(http.StatusUnauthorized)
}
