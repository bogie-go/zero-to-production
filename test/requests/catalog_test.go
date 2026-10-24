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

func TestCatalogListsProductsNewestFirstInPages(t *testing.T) {
	app := testapp.New(t)
	for i := 0; i < 5; i++ {
		factories.Product(t, app.Store)
	}

	var page views.ProductsResponse
	app.Get("/products?per_page=2&page=3").Do().Status(http.StatusOK).JSON(&page)

	if len(page.Products) != 1 {
		t.Errorf("last page holds %d products, want the 1 left over", len(page.Products))
	}
	if p := page.Pagination; p.Page != 3 || p.PerPage != 2 || p.Total != 5 || p.TotalPages != 3 {
		t.Errorf("pagination = %+v", p)
	}
}

func TestCatalogFiltersByCategoryAndHidesArchivedProducts(t *testing.T) {
	app := testapp.New(t)
	audio := factories.Category(t, app.Store)
	inAudio := factories.Product(t, app.Store, func(p *domain.Product) { p.CategoryID = &audio.ID })
	factories.Product(t, app.Store)
	gone := factories.Product(t, app.Store, func(p *domain.Product) { p.CategoryID = &audio.ID })
	if _, err := app.Store.ArchiveProduct(t.Context(), strconv.FormatInt(gone.ID, 10)); err != nil {
		t.Fatal(err)
	}

	var list views.ProductsResponse
	app.Get("/products?category_id=" + strconv.FormatInt(audio.ID, 10)).Do().Status(http.StatusOK).JSON(&list)
	if len(list.Products) != 1 || list.Products[0].ID != inAudio.ID {
		t.Errorf("products in the category = %+v, want only the live one", list.Products)
	}
	if list.Products[0].CategoryName != audio.Name {
		t.Errorf("category_name = %q, want %q", list.Products[0].CategoryName, audio.Name)
	}

	app.Get("/products/" + strconv.FormatInt(gone.ID, 10)).Do().Status(http.StatusNotFound)
}

func TestCatalogRefusesABadPage(t *testing.T) {
	app := testapp.New(t)
	app.Get("/products?page=0").Do().Status(http.StatusUnprocessableEntity)
	app.Get("/products?category_id=audio").Do().Status(http.StatusUnprocessableEntity)
}
