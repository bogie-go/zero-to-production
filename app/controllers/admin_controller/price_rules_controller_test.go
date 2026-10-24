package admin_controller_test

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

var _ = Describe("/admin/price_rules", func() {
	var staff domain.User

	BeforeEach(func() {
		staff = factories.Staff(GinkgoT(), app.Store)
	})

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
})
