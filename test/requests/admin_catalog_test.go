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

func TestPublicCategoriesCountOnlyLiveProducts(t *testing.T) {
	app := testapp.New(t)
	audio, empty := factories.Category(t, app.Store), factories.Category(t, app.Store)
	factories.Product(t, app.Store, func(p *domain.Product) { p.CategoryID = &audio.ID })
	gone := factories.Product(t, app.Store, func(p *domain.Product) { p.CategoryID = &audio.ID })
	if _, err := app.Store.ArchiveProduct(t.Context(), strconv.FormatInt(gone.ID, 10)); err != nil {
		t.Fatal(err)
	}

	var list views.CategoriesResponse
	app.Get("/categories").Do().Status(http.StatusOK).JSON(&list)
	counts := map[int64]int64{}
	for _, c := range list.Categories {
		counts[c.ID] = c.ProductCount
	}
	if counts[audio.ID] != 1 || counts[empty.ID] != 0 || len(counts) != 2 {
		t.Errorf("counts = %v, want audio 1 and the empty category listed with 0", counts)
	}
}

func TestStaffManageCategories(t *testing.T) {
	app := testapp.New(t)
	staff := factories.Staff(t, app.Store)

	var created views.CategoryResponse
	app.Post("/admin/categories", views.CategoryRequest{Name: "Audio"}).AsUser(staff).Do().Status(http.StatusCreated).JSON(&created)
	id := "/admin/categories/" + strconv.FormatInt(created.ID, 10)

	app.Post("/admin/categories", views.CategoryRequest{Name: "Audio"}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
	app.Post("/admin/categories", views.CategoryRequest{Name: ""}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)

	var renamed views.CategoryResponse
	app.Patch(id, views.CategoryRequest{Name: "Sound"}).AsUser(staff).Do().Status(http.StatusOK).JSON(&renamed)
	if renamed.Name != "Sound" {
		t.Errorf("name = %q after rename", renamed.Name)
	}
	app.Get("/admin/categories/999999").AsUser(staff).Do().Status(http.StatusNotFound)
	app.Post("/admin/categories", views.CategoryRequest{Name: "x"}).AsUser(factories.User(t, app.Store)).Do().Status(http.StatusForbidden)
}

// Deleting a category does not delete or hide its products: they simply have
// no category any more.
func TestDeletingACategoryKeepsItsProducts(t *testing.T) {
	app := testapp.New(t)
	staff := factories.Staff(t, app.Store)
	category := factories.Category(t, app.Store)
	product := factories.Product(t, app.Store, func(p *domain.Product) { p.CategoryID = &category.ID })

	app.Delete("/admin/categories/" + strconv.FormatInt(category.ID, 10)).AsUser(staff).Do().Status(http.StatusNoContent)

	var shown views.ProductResponse
	app.Get("/products/" + strconv.FormatInt(product.ID, 10)).Do().Status(http.StatusOK).JSON(&shown)
	if shown.CategoryID != nil || shown.CategoryName != "" {
		t.Errorf("product still shows category %v %q", shown.CategoryID, shown.CategoryName)
	}
}

func TestStaffManagePriceRules(t *testing.T) {
	app := testapp.New(t)
	staff := factories.Staff(t, app.Store)

	var kinds views.KindsResponse
	app.Get("/admin/price_rules/kinds").AsUser(staff).Do().Status(http.StatusOK).JSON(&kinds)
	if len(kinds.Kinds) == 0 {
		t.Fatal("no rule kinds offered")
	}

	var rule views.PriceRuleResponse
	app.Post("/admin/price_rules", `{"name":"Spring sale","kind":"fixed_discount","properties":{"amount_cents":250,"junk":1},"active":false}`).
		AsUser(staff).Do().Status(http.StatusCreated).JSON(&rule)
	if rule.Status != "off" || rule.Summary == "" {
		t.Errorf("new rule = %+v, want it off with a summary", rule)
	}
	if string(rule.Properties) != `{"amount_cents":250}` {
		t.Errorf("properties = %s, want unknown keys dropped", rule.Properties)
	}
	id := "/admin/price_rules/" + strconv.FormatInt(rule.ID, 10)

	app.Post(id+"/activate", nil).AsUser(staff).Do().Status(http.StatusOK).Contains(`"status":"active"`)
	app.Post(id+"/deactivate", nil).AsUser(staff).Do().Status(http.StatusOK).Contains(`"status":"off"`)

	app.Post("/admin/price_rules", `{"name":"Bad","kind":"no_such_kind","properties":{}}`).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
	app.Post("/admin/price_rules", `{"name":"Bad","kind":"fixed_discount","properties":{"amount_cents":-5}}`).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
	app.Post("/admin/price_rules", `{"name":"Bad","kind":"fixed_discount","properties":[1]}`).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)

	app.Delete(id).AsUser(staff).Do().Status(http.StatusNoContent)
	app.Get(id).AsUser(staff).Do().Status(http.StatusNotFound)
}

// The shop applies what the admin switches on, and only that.
func TestSwitchingARuleOnChangesWhatTheShopCharges(t *testing.T) {
	app := testapp.New(t)
	staff := factories.Staff(t, app.Store)
	product := factories.Product(t, app.Store, func(p *domain.Product) { p.PriceCents = 1000 })
	rule := factories.PriceRule(t, app.Store, func(r *domain.PriceRule) { r.Active = false })
	app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("g").Do().Status(http.StatusCreated)

	var cart views.CartResponse
	app.Get("/cart").AsGuest("g").Do().JSON(&cart)
	if cart.DiscountCents != 0 {
		t.Fatalf("discount = %d with the rule off", cart.DiscountCents)
	}
	app.Post("/admin/price_rules/"+strconv.FormatInt(rule.ID, 10)+"/activate", nil).AsUser(staff).Do().Status(http.StatusOK)
	app.Get("/cart").AsGuest("g").Do().JSON(&cart)
	if cart.DiscountCents != 100 || len(cart.AppliedRules) != 1 {
		t.Errorf("cart = %+v, want the $1.00 rule applied", cart)
	}
}
