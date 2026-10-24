package requests_test

import (
	"net/http"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("Categories and price rules", func() {
	var staff domain.User

	BeforeEach(func() {
		staff = factories.Staff(GinkgoT(), app.Store)
	})

	Describe("GET /categories", func() {
		It("is an empty list, not null, with no categories", func() {
			app.Get("/categories").Do().Status(http.StatusOK).Contains(`"categories":[]`)
		})

		It("counts only live products and lists empty categories with 0", func() {
			audio, empty := factories.Category(GinkgoT(), app.Store), factories.Category(GinkgoT(), app.Store)
			factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.CategoryID = &audio.ID })
			gone := factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.CategoryID = &audio.ID })
			_, err := app.Store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(gone.ID, 10))
			Expect(err).NotTo(HaveOccurred())

			var list views.CategoriesResponse
			app.Get("/categories").Do().Status(http.StatusOK).JSON(&list)

			counts := map[int64]int64{}
			for _, c := range list.Categories {
				counts[c.ID] = c.ProductCount
			}
			Expect(counts).To(Equal(map[int64]int64{audio.ID: 1, empty.ID: 0}))
		})
	})

	Describe("/admin/categories", func() {
		It("lets staff create, rename and look up categories", func() {
			var created views.CategoryResponse
			app.Post("/admin/categories", views.CategoryRequest{Name: "Audio"}).AsUser(staff).Do().Status(http.StatusCreated).JSON(&created)
			path := "/admin/categories/" + strconv.FormatInt(created.ID, 10)

			app.Post("/admin/categories", views.CategoryRequest{Name: "Audio"}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
			app.Post("/admin/categories", views.CategoryRequest{Name: ""}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)

			var renamed views.CategoryResponse
			app.Patch(path, views.CategoryRequest{Name: "Sound"}).AsUser(staff).Do().Status(http.StatusOK).JSON(&renamed)
			Expect(renamed.Name).To(Equal("Sound"))
			app.Get("/admin/categories/999999").AsUser(staff).Do().Status(http.StatusNotFound)
		})

		It("is closed to customers, and to anyone without a token", func() {
			app.Post("/admin/categories", views.CategoryRequest{Name: "x"}).Do().Status(http.StatusUnauthorized)
			app.Post("/admin/categories", views.CategoryRequest{Name: "x"}).AsUser(factories.User(GinkgoT(), app.Store)).Do().Status(http.StatusForbidden)

			var list views.CategoriesResponse
			app.Get("/admin/categories").AsUser(staff).Do().JSON(&list)
			Expect(list.Categories).To(BeEmpty(), "nothing was created")
		})

		It("lists categories with their product counts", func() {
			category := factories.Category(GinkgoT(), app.Store)
			factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.CategoryID = &category.ID })
			var list views.CategoriesResponse
			app.Get("/admin/categories").AsUser(staff).Do().Status(http.StatusOK).JSON(&list)
			Expect(list.Categories).To(HaveLen(1))
			Expect(list.Categories[0].ProductCount).To(BeEquivalentTo(1))
		})

		It("keeps a deleted category's products, just without a category", func() {
			category := factories.Category(GinkgoT(), app.Store)
			product := factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.CategoryID = &category.ID })

			app.Delete("/admin/categories/" + strconv.FormatInt(category.ID, 10)).AsUser(staff).Do().Status(http.StatusNoContent)

			var shown views.ProductResponse
			app.Get("/products/" + strconv.FormatInt(product.ID, 10)).Do().Status(http.StatusOK).JSON(&shown)
			Expect(shown.CategoryID).To(BeNil())
			Expect(shown.CategoryName).To(BeEmpty())
		})
	})

	Describe("/admin/price_rules", func() {
		It("offers every kind of rule, with its fields", func() {
			var kinds views.KindsResponse
			app.Get("/admin/price_rules/kinds").AsUser(staff).Do().Status(http.StatusOK).JSON(&kinds)
			Expect(kinds.Kinds).NotTo(BeEmpty())
			for _, k := range kinds.Kinds {
				Expect(k.Label).NotTo(BeEmpty(), k.Kind)
				Expect(k.Fields).NotTo(BeEmpty(), k.Kind)
			}
		})

		It("says where each rule stands: active, off, scheduled or ended", func() {
			later, earlier := time.Now().Add(24*time.Hour), time.Now().Add(-24*time.Hour)
			factories.PriceRule(GinkgoT(), app.Store, func(r *domain.PriceRule) { r.Name = "live" })
			factories.PriceRule(GinkgoT(), app.Store, func(r *domain.PriceRule) { r.Name = "off"; r.Active = false })
			factories.PriceRule(GinkgoT(), app.Store, func(r *domain.PriceRule) { r.Name = "soon"; r.StartsAt = &later })
			factories.PriceRule(GinkgoT(), app.Store, func(r *domain.PriceRule) { r.Name = "over"; r.EndsAt = &earlier })

			var list views.PriceRulesResponse
			app.Get("/admin/price_rules").AsUser(staff).Do().Status(http.StatusOK).JSON(&list)

			status := map[string]string{}
			for _, r := range list.PriceRules {
				status[r.Name] = r.Status
				Expect(r.Summary).NotTo(BeEmpty(), r.Name)
			}
			Expect(status).To(Equal(map[string]string{"live": "active", "off": "off", "soon": "scheduled", "over": "ended"}))
		})

		It("shows and updates one", func() {
			rule := factories.PriceRule(GinkgoT(), app.Store)
			path := "/admin/price_rules/" + strconv.FormatInt(rule.ID, 10)

			var updated views.PriceRuleResponse
			app.Patch(path, `{"name":"Renamed","kind":"fixed_discount","properties":{"amount_cents":500},"active":true,"priority":7}`).
				AsUser(staff).Do().Status(http.StatusOK).JSON(&updated)
			Expect(updated.ID).To(Equal(rule.ID))
			Expect(updated.Name).To(Equal("Renamed"))
			Expect(updated.Priority).To(BeEquivalentTo(7))
			app.Get(path).AsUser(staff).Do().Status(http.StatusOK).Contains("Renamed")
		})

		It("is closed to customers and changes nothing for them", func() {
			customer := factories.User(GinkgoT(), app.Store)
			app.Get("/admin/price_rules").Do().Status(http.StatusUnauthorized)
			app.Post("/admin/price_rules", `{"name":"x","kind":"fixed_discount","properties":{"amount_cents":1}}`).AsUser(customer).Do().Status(http.StatusForbidden)
			var list views.PriceRulesResponse
			app.Get("/admin/price_rules").AsUser(staff).Do().JSON(&list)
			Expect(list.PriceRules).To(BeEmpty())
		})

		It("answers 400 for malformed JSON and 404 for an unknown id", func() {
			app.Post("/admin/price_rules", "{").AsUser(staff).Do().Status(http.StatusBadRequest)
			app.Get("/admin/price_rules/999999").AsUser(staff).Do().Status(http.StatusNotFound)
		})

		It("creates, switches and deletes a rule", func() {
			var rule views.PriceRuleResponse
			app.Post("/admin/price_rules", `{"name":"Spring sale","kind":"fixed_discount","properties":{"amount_cents":250,"junk":1},"active":false}`).
				AsUser(staff).Do().Status(http.StatusCreated).JSON(&rule)
			Expect(rule.Status).To(Equal("off"))
			Expect(rule.Summary).NotTo(BeEmpty())
			Expect(rule.Properties).To(MatchJSON(`{"amount_cents":250}`), "unknown keys are dropped")
			path := "/admin/price_rules/" + strconv.FormatInt(rule.ID, 10)

			app.Post(path+"/activate", nil).AsUser(staff).Do().Status(http.StatusOK).Contains(`"status":"active"`)
			app.Post(path+"/deactivate", nil).AsUser(staff).Do().Status(http.StatusOK).Contains(`"status":"off"`)

			app.Delete(path).AsUser(staff).Do().Status(http.StatusNoContent)
			app.Get(path).AsUser(staff).Do().Status(http.StatusNotFound)
		})

		DescribeTable("refuses a bad rule",
			func(body string) {
				app.Post("/admin/price_rules", body).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
			},
			Entry("a percentage over 100", `{"name":"x","kind":"percentage_discount","properties":{"percentage":500}}`),
			Entry("a buy-x-get-y without quantities", `{"name":"x","kind":"buy_x_get_y","properties":{}}`),
			Entry("an unknown kind", `{"name":"Bad","kind":"no_such_kind","properties":{}}`),
			Entry("a negative amount", `{"name":"Bad","kind":"fixed_discount","properties":{"amount_cents":-5}}`),
			Entry("properties that are not an object", `{"name":"Bad","kind":"fixed_discount","properties":[1]}`),
		)

		It("changes what the shop charges once a rule is switched on", func() {
			product := factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.PriceCents = 1000 })
			rule := factories.PriceRule(GinkgoT(), app.Store, func(r *domain.PriceRule) { r.Active = false })
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("g").Do().Status(http.StatusCreated)

			var cart views.CartResponse
			app.Get("/cart").AsGuest("g").Do().JSON(&cart)
			Expect(cart.DiscountCents).To(BeZero())

			app.Post("/admin/price_rules/"+strconv.FormatInt(rule.ID, 10)+"/activate", nil).AsUser(staff).Do().Status(http.StatusOK)
			app.Get("/cart").AsGuest("g").Do().JSON(&cart)
			Expect(cart.DiscountCents).To(BeEquivalentTo(100))
			Expect(cart.AppliedRules).To(HaveLen(1))
		})
	})
})
