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

func TestAdminIsForStaffOnly(t *testing.T) {
	app := testapp.New(t)
	customer, staff := factories.User(t, app.Store), factories.Staff(t, app.Store)

	app.Get("/admin/products").Do().Status(http.StatusUnauthorized)
	app.Get("/admin/products").AsUser(customer).Do().Status(http.StatusForbidden)
	app.Get("/admin/products").AsGuest("someone").Do().Status(http.StatusForbidden)
	app.Get("/admin/products").AsUser(staff).Do().Status(http.StatusOK)
}

func TestStaffManageProductsAndArchivingTakesThemOffTheShop(t *testing.T) {
	app := testapp.New(t)
	staff := factories.Staff(t, app.Store)
	category := factories.Category(t, app.Store)

	var created views.ProductResponse
	app.Post("/admin/products", views.ProductRequest{
		Name: "Desk Lamp", Description: "warm light", PriceCents: 2500, Stock: 4, ImageUrl: "/lamp.webp", CategoryID: &category.ID,
	}).AsUser(staff).Do().Status(http.StatusCreated).JSON(&created)
	id := strconv.FormatInt(created.ID, 10)

	app.Post("/admin/products", views.ProductRequest{Name: ""}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)

	app.Get("/products/" + id).Do().Status(http.StatusOK).Contains("Desk Lamp")
	app.Post("/admin/products/"+id+"/archive", nil).AsUser(staff).Do().Status(http.StatusOK)
	app.Get("/products/" + id).Do().Status(http.StatusNotFound)

	var all views.ProductsResponse
	app.Get("/admin/products").AsUser(staff).Do().JSON(&all)
	if len(all.Products) != 1 || all.Products[0].ArchivedAt == nil {
		t.Errorf("the admin list = %+v, want the archived product still there", all.Products)
	}
	app.Get("/admin/products?include_archived=false").AsUser(staff).Do().JSON(&all)
	if len(all.Products) != 0 {
		t.Errorf("include_archived=false still lists %d", len(all.Products))
	}
}

func TestADiscountRuleChangesTheCartTotal(t *testing.T) {
	app := testapp.New(t)
	product := factories.Product(t, app.Store, func(p *domain.Product) { p.PriceCents = 1000 })
	factories.PriceRule(t, app.Store) // active, $1.00 off
	user := factories.User(t, app.Store)
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 2}).AsUser(user).Do().Status(http.StatusCreated)

	var cart views.CartResponse
	app.Get("/cart").AsUser(user).Do().Status(http.StatusOK).JSON(&cart)
	if cart.SubtotalCents != 2000 || cart.DiscountCents <= 0 || cart.TotalCents != cart.SubtotalCents-cart.DiscountCents {
		t.Errorf("cart = %+v, want a discount taken off the subtotal", cart)
	}
}
