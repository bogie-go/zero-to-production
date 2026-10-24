package requests_test

import (
	"net/http"
	"strconv"
	"testing"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
	"ecommerce/test/testapp"
)

func TestCartLinesCanBeChangedAndRemoved(t *testing.T) {
	app := testapp.New(t)
	product := factories.Product(t, app.Store, func(p *domain.Product) { p.PriceCents = 300; p.Stock = 5 })
	var cart views.CartResponse
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("g1").Do().Status(http.StatusCreated).JSON(&cart)
	line := "/cart_items/" + strconv.FormatInt(cart.Items[0].ID, 10)

	app.Patch(line, views.UpdateCartItemRequest{Quantity: 4}).AsGuest("g1").Do().Status(http.StatusOK).JSON(&cart)
	if cart.TotalCents != 1200 {
		t.Errorf("total after raising to 4 = %d, want 1200", cart.TotalCents)
	}
	app.Patch(line, views.UpdateCartItemRequest{Quantity: 6}).AsGuest("g1").Do().Status(http.StatusUnprocessableEntity).Contains("in stock")
	app.Patch(line, views.UpdateCartItemRequest{Quantity: 0}).AsGuest("g1").Do().Status(http.StatusUnprocessableEntity)

	app.Delete(line).AsGuest("g1").Do().Status(http.StatusOK).JSON(&cart)
	if len(cart.Items) != 0 {
		t.Errorf("lines after delete = %d", len(cart.Items))
	}
}

// A line is reachable only through its owner's own cart: someone else's id in
// the URL is simply not found.
func TestCartLinesBelongToTheirOwner(t *testing.T) {
	app := testapp.New(t)
	product := factories.Product(t, app.Store)
	var cart views.CartResponse
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("owner").Do().JSON(&cart)
	line := "/cart_items/" + strconv.FormatInt(cart.Items[0].ID, 10)

	app.Patch(line, views.UpdateCartItemRequest{Quantity: 2}).AsGuest("thief").Do().Status(http.StatusNotFound)
	app.Delete(line).AsGuest("thief").Do().Status(http.StatusNotFound)
	app.Get("/cart").AsGuest("owner").Do().JSON(&cart)
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 1 {
		t.Errorf("the owner's cart was changed by someone else: %+v", cart)
	}
}

func TestAddingToTheCartRefusesBadLines(t *testing.T) {
	app := testapp.New(t)
	product := factories.Product(t, app.Store, func(p *domain.Product) { p.Stock = 2 })
	archived := factories.Product(t, app.Store)
	if _, err := app.Store.ArchiveProduct(t.Context(), strconv.FormatInt(archived.ID, 10)); err != nil {
		t.Fatal(err)
	}

	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 3}).AsGuest("g").Do().Status(http.StatusUnprocessableEntity)
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 0}).AsGuest("g").Do().Status(http.StatusUnprocessableEntity)
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: 999999, Quantity: 1}).AsGuest("g").Do().Status(http.StatusUnprocessableEntity)
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: archived.ID, Quantity: 1}).AsGuest("g").Do().Status(http.StatusUnprocessableEntity)
	app.Post("/cart_items", "{").AsGuest("g").Do().Status(http.StatusBadRequest)
}

func TestAddingTheSameProductTwiceMakesOneLine(t *testing.T) {
	app := testapp.New(t)
	product := factories.Product(t, app.Store)
	var cart views.CartResponse
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("g").Do().Status(http.StatusCreated)
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 2}).AsGuest("g").Do().JSON(&cart)
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 3 {
		t.Errorf("cart = %+v, want one line of 3", cart)
	}
}
