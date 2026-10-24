package admin_controller_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("/admin/products", func() {
	var staff domain.User

	Context("with a staff account", func() {
		BeforeEach(func() {
			staff = factories.Staff(GinkgoT(), app.Store)
		})

		Describe("the collection", func() {
			It("refuses a product without a name", func() {
				app.Post("/admin/products", views.ProductRequest{Name: ""}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
			})

			Context("with a category", func() {
				var category domain.Category
				var request views.ProductRequest

				BeforeEach(func() {
					category = factories.Category(GinkgoT(), app.Store)
					request = views.ProductRequest{
						Name: "Desk Lamp", Description: "warm light", PriceCents: 2500, Stock: 4, ImageUrl: "/lamp.webp", CategoryID: &category.ID,
					}
				})

				It("creates a product", func() {
					app.Post("/admin/products", request).AsUser(staff).Do().Status(http.StatusCreated)
				})

				Context("with a product created", func() {
					var id string

					BeforeEach(func() {
						var created views.ProductResponse
						app.Post("/admin/products", request).AsUser(staff).Do().Status(http.StatusCreated).JSON(&created)
						id = strconv.FormatInt(created.ID, 10)
					})

					It("puts it on the shop", func() {
						app.Get("/products/" + id).Do().Status(http.StatusOK).Contains("Desk Lamp")
					})

					Context("when it is archived", func() {
						var all views.ProductsResponse

						BeforeEach(func() {
							app.Post("/admin/products/"+id+"/archive", nil).AsUser(staff).Do().Status(http.StatusOK)
						})

						It("takes it off the shop", func() {
							app.Get("/products/" + id).Do().Status(http.StatusNotFound)
						})

						It("still lists it for staff", func() {
							app.Get("/admin/products").AsUser(staff).Do().JSON(&all)
							Expect(all.Products).To(HaveLen(1))
							Expect(all.Products[0].ArchivedAt).NotTo(BeNil())
						})

						It("leaves it out when include_archived is false", func() {
							app.Get("/admin/products?include_archived=false").AsUser(staff).Do().JSON(&all)
							Expect(all.Products).To(BeEmpty())
						})
					})
				})
			})
		})

		Describe("one at a time", func() {
			var product domain.Product
			var path string

			Context("with a product", func() {
				BeforeEach(func() {
					product = factories.Product(GinkgoT(), app.Store)
					path = "/admin/products/" + strconv.FormatInt(product.ID, 10)
				})

				It("shows it", func() {
					var shown views.ProductResponse
					app.Get(path).AsUser(staff).Do().Status(http.StatusOK).JSON(&shown)
					Expect(shown.ID).To(Equal(product.ID))
				})

				It("updates it", func() {
					var updated views.ProductResponse
					app.Patch(path, views.ProductRequest{Name: "Renamed", PriceCents: 1234, Stock: 3, ImageUrl: "/r.webp"}).
						AsUser(staff).Do().Status(http.StatusOK).JSON(&updated)
					Expect(updated.ID).To(Equal(product.ID), "the id is carried over")
					Expect(updated.Name).To(Equal("Renamed"))
					Expect(updated.PriceCents).To(BeEquivalentTo(1234))
				})

				It("deletes it", func() {
					app.Delete(path).AsUser(staff).Do().Status(http.StatusNoContent)
					app.Get(path).AsUser(staff).Do().Status(http.StatusNotFound)
				})

				It("archives it", func() {
					var archived views.ProductResponse
					app.Post(path+"/archive", nil).AsUser(staff).Do().Status(http.StatusOK).JSON(&archived)
					Expect(archived.ArchivedAt).NotTo(BeNil())
				})

				Context("when it is archived", func() {
					BeforeEach(func() {
						app.Post(path+"/archive", nil).AsUser(staff).Do().Status(http.StatusOK)
					})

					It("restores it, with archived_at null again", func() {
						var restored views.ProductResponse
						app.Post(path+"/restore", nil).AsUser(staff).Do().Status(http.StatusOK).JSON(&restored)
						Expect(restored.ArchivedAt).To(BeNil())
					})
				})

				Context("when it was ordered", func() {
					BeforeEach(func() {
						user := factories.User(GinkgoT(), app.Store)
						app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsUser(user).Do().Status(http.StatusCreated)
						app.Post("/orders", nil).AsUser(user).Do().Status(http.StatusCreated)
					})

					It("refuses to delete it, with 422", func() {
						app.Delete(path).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
					})
				})

				Context("with a customer", func() {
					var customer domain.User

					BeforeEach(func() {
						customer = factories.User(GinkgoT(), app.Store)
					})

					expectUntouched := func() {
						var list views.ProductsResponse
						app.Get("/admin/products").AsUser(staff).Do().JSON(&list)
						Expect(list.Products).To(HaveLen(1))
						Expect(list.Products[0].ArchivedAt).To(BeNil())
					}

					It("cannot create a product", func() {
						app.Post("/admin/products", views.ProductRequest{Name: "Sneaky", PriceCents: 1, Stock: 1, ImageUrl: "/s"}).AsUser(customer).Do().Status(http.StatusForbidden)
						expectUntouched()
					})

					It("cannot archive a product", func() {
						app.Post(path+"/archive", nil).AsUser(customer).Do().Status(http.StatusForbidden)
						expectUntouched()
					})
				})

				Context("with twenty-one more products", func() {
					var list views.ProductsResponse

					BeforeEach(func() {
						for i := 0; i < 21; i++ {
							factories.Product(GinkgoT(), app.Store)
						}
					})

					It("pages the list by 20", func() {
						app.Get("/admin/products").AsUser(staff).Do().Status(http.StatusOK).JSON(&list)
						Expect(list.Products).To(HaveLen(20))
						Expect(list.Pagination.TotalPages).To(BeEquivalentTo(2))
					})

					It("takes page and per_page", func() {
						app.Get("/admin/products?page=2&per_page=10").AsUser(staff).Do().JSON(&list)
						Expect(list.Pagination.Page).To(BeEquivalentTo(2))
					})

					It("refuses page 0 with 422", func() {
						app.Get("/admin/products?page=0").AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
					})
				})
			})
		})
	})
})
