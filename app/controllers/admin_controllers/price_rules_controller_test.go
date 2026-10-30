package admin_controllers_test

import (
	"net/http"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("/admin/price_rules", func() {
	var staff models.User

	Context("with a staff account", func() {
		BeforeEach(func() {
			staff = factories.Staff(GinkgoT(), app.Store)
		})

		Describe("GET /admin/price_rules/kinds", func() {
			It("offers every kind of rule, with its fields", func() {
				var kinds views.KindsResponse
				app.Get("/admin/price_rules/kinds").AsUser(staff).Do().Status(http.StatusOK).JSON(&kinds)
				Expect(kinds.Kinds).NotTo(BeEmpty())
				for _, k := range kinds.Kinds {
					Expect(k.Label).NotTo(BeEmpty(), k.Kind)
					Expect(k.Fields).NotTo(BeEmpty(), k.Kind)
				}
			})
		})

		Describe("GET /admin/price_rules", func() {
			Context("with a live, an off, a scheduled and an ended rule", func() {
				BeforeEach(func() {
					later, earlier := time.Now().Add(24*time.Hour), time.Now().Add(-24*time.Hour)
					factories.PriceRule(GinkgoT(), app.Store, func(r *models.PriceRule) { r.Name = "live" })
					factories.PriceRule(GinkgoT(), app.Store, func(r *models.PriceRule) { r.Name = "off"; r.Active = false })
					factories.PriceRule(GinkgoT(), app.Store, func(r *models.PriceRule) { r.Name = "soon"; r.StartsAt = &later })
					factories.PriceRule(GinkgoT(), app.Store, func(r *models.PriceRule) { r.Name = "over"; r.EndsAt = &earlier })
				})

				It("says where each rule stands: active, off, scheduled or ended", func() {
					var list views.AdminPriceRulesResponse
					app.Get("/admin/price_rules").AsUser(staff).Do().Status(http.StatusOK).JSON(&list)

					status := map[string]string{}
					for _, r := range list.PriceRules {
						status[r.Name] = r.Status
						Expect(r.Summary).NotTo(BeEmpty(), r.Name)
					}
					Expect(status).To(Equal(map[string]string{"live": "active", "off": "off", "soon": "scheduled", "over": "ended"}))
				})
			})
		})

		Context("with a rule", func() {
			var rule models.PriceRule
			var path string

			BeforeEach(func() {
				rule = factories.PriceRule(GinkgoT(), app.Store)
				path = "/admin/price_rules/" + strconv.FormatInt(rule.ID, 10)
			})

			It("shows it", func() {
				app.Get(path).AsUser(staff).Do().Status(http.StatusOK)
			})

			It("updates it", func() {
				var updated views.AdminPriceRuleResponse
				app.Patch(path, `{"name":"Renamed","kind":"fixed_discount","properties":{"amount_cents":500},"active":true,"priority":7}`).
					AsUser(staff).Do().Status(http.StatusOK).JSON(&updated)
				Expect(updated.ID).To(Equal(rule.ID))
				Expect(updated.Name).To(Equal("Renamed"))
				Expect(updated.Priority).To(BeEquivalentTo(7))
			})

			Context("when it was updated", func() {
				BeforeEach(func() {
					app.Patch(path, `{"name":"Renamed","kind":"fixed_discount","properties":{"amount_cents":500},"active":true,"priority":7}`).
						AsUser(staff).Do().Status(http.StatusOK)
				})

				It("shows the new name", func() {
					app.Get(path).AsUser(staff).Do().Status(http.StatusOK).Contains("Renamed")
				})
			})
		})

		Context("with a customer", func() {
			var customer models.User

			BeforeEach(func() {
				customer = factories.User(GinkgoT(), app.Store)
			})

			It("answers 401 without a token", func() {
				app.Get("/admin/price_rules").Do().Status(http.StatusUnauthorized)
			})

			It("is closed to the customer, who changes nothing", func() {
				app.Post("/admin/price_rules", `{"name":"x","kind":"fixed_discount","properties":{"amount_cents":1}}`).AsUser(customer).Do().Status(http.StatusForbidden)

				var list views.AdminPriceRulesResponse
				app.Get("/admin/price_rules").AsUser(staff).Do().JSON(&list)
				Expect(list.PriceRules).To(BeEmpty())
			})
		})

		It("answers 400 for malformed JSON", func() {
			app.Post("/admin/price_rules", "{").AsUser(staff).Do().Status(http.StatusBadRequest)
		})

		It("answers 404 for an unknown id", func() {
			app.Get("/admin/price_rules/999999").AsUser(staff).Do().Status(http.StatusNotFound)
		})

		Describe("POST /admin/price_rules", func() {
			const spring = `{"name":"Spring sale","kind":"fixed_discount","properties":{"amount_cents":250,"junk":1},"active":false}`

			It("creates a rule that is off, with a summary, dropping unknown keys", func() {
				var rule views.AdminPriceRuleResponse
				app.Post("/admin/price_rules", spring).AsUser(staff).Do().Status(http.StatusCreated).JSON(&rule)
				Expect(rule.Status).To(Equal("off"))
				Expect(rule.Summary).NotTo(BeEmpty())
				Expect(rule.Properties).To(MatchJSON(`{"amount_cents":250}`), "unknown keys are dropped")
			})

			Context("with a rule created off", func() {
				var path string

				BeforeEach(func() {
					var rule views.AdminPriceRuleResponse
					app.Post("/admin/price_rules", spring).AsUser(staff).Do().Status(http.StatusCreated).JSON(&rule)
					path = "/admin/price_rules/" + strconv.FormatInt(rule.ID, 10)
				})

				It("switches it on", func() {
					app.Post(path+"/activate", nil).AsUser(staff).Do().Status(http.StatusOK).Contains(`"status":"active"`)
				})

				It("deletes it", func() {
					app.Delete(path).AsUser(staff).Do().Status(http.StatusNoContent)
					app.Get(path).AsUser(staff).Do().Status(http.StatusNotFound)
				})

				Context("when it is switched on", func() {
					BeforeEach(func() {
						app.Post(path+"/activate", nil).AsUser(staff).Do().Status(http.StatusOK)
					})

					It("switches it off", func() {
						app.Post(path+"/deactivate", nil).AsUser(staff).Do().Status(http.StatusOK).Contains(`"status":"off"`)
					})
				})
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
	})
})
