package admin_controllers_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("/admin/products", func() {
	var staff models.User

	Context("with a staff account", func() {
		BeforeEach(func() {
			staff = factories.Staff(GinkgoT(), app.Store)
		})

		Describe("the collection", func() {
			It("refuses a product without a name", func() {
				app.Post("/admin/products", views.AdminProductRequest{Name: ""}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
			})

			Context("with a product to create", func() {
				var request views.AdminProductRequest

				BeforeEach(func() {
					request = views.AdminProductRequest{
						Name: "Desk Lamp", Description: "warm light", PriceCents: 2500, Stock: 4, ImageUrl: "/lamp.webp",
					}
				})

				It("creates a product", func() {
					app.Post("/admin/products", request).AsUser(staff).Do().Status(http.StatusCreated)
				})

				Context("with a product created", func() {
					var id string

					BeforeEach(func() {
						var created views.AdminProductResponse
						app.Post("/admin/products", request).AsUser(staff).Do().Status(http.StatusCreated).JSON(&created)
						id = strconv.FormatInt(created.ID, 10)
					})

					It("puts it on the shop", func() {
						app.Get("/products/" + id).Do().Status(http.StatusOK).Contains("Desk Lamp")
					})

					Context("when it is archived", func() {
						var all views.AdminProductsResponse

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
			var product models.Product
			var path string

			Context("with a product", func() {
				BeforeEach(func() {
					product = factories.Product(GinkgoT(), app.Store)
					path = "/admin/products/" + strconv.FormatInt(product.ID, 10)
				})

				It("shows it", func() {
					var shown views.AdminProductResponse
					app.Get(path).AsUser(staff).Do().Status(http.StatusOK).JSON(&shown)
					Expect(shown.ID).To(Equal(product.ID))
				})

				It("updates it", func() {
					var updated views.AdminProductResponse
					app.Patch(path, views.AdminProductRequest{Name: "Renamed", PriceCents: 1234, Stock: 3, ImageUrl: "/r.webp"}).
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
					var archived views.AdminProductResponse
					app.Post(path+"/archive", nil).AsUser(staff).Do().Status(http.StatusOK).JSON(&archived)
					Expect(archived.ArchivedAt).NotTo(BeNil())
				})

				Context("when it is archived", func() {
					BeforeEach(func() {
						app.Post(path+"/archive", nil).AsUser(staff).Do().Status(http.StatusOK)
					})

					It("restores it, with archived_at null again", func() {
						var restored views.AdminProductResponse
						app.Post(path+"/restore", nil).AsUser(staff).Do().Status(http.StatusOK).JSON(&restored)
						Expect(restored.ArchivedAt).To(BeNil())
					})
				})

				Describe("PUT /admin/products/:id/categories", func() {
					var gifts, drinkware models.Category

					BeforeEach(func() {
						gifts = factories.Category(GinkgoT(), app.Store, func(c *models.Category) { c.Name = "Gifts" })
						drinkware = factories.Category(GinkgoT(), app.Store, func(c *models.Category) { c.Name = "Drinkware" })
					})

					It("puts the product in the categories sent, once each", func() {
						var set views.ProductCategoriesResponse
						app.Put(path+"/categories", views.ProductCategoriesRequest{CategoryIDs: []int64{gifts.ID, drinkware.ID, gifts.ID}}).
							AsUser(staff).Do().Status(http.StatusOK).JSON(&set)
						Expect(set.Categories).To(HaveLen(2))
						Expect(set.Categories[0].Name).To(Equal("Drinkware"))
						Expect(set.Categories[1].Name).To(Equal("Gifts"))
					})

					It("shows them on the product", func() {
						app.Put(path+"/categories", views.ProductCategoriesRequest{CategoryIDs: []int64{gifts.ID}}).AsUser(staff).Do().Status(http.StatusOK)

						var shown views.AdminProductDetailResponse
						app.Get(path).AsUser(staff).Do().Status(http.StatusOK).JSON(&shown)
						Expect(shown.Categories).To(HaveLen(1))
						Expect(shown.Categories[0].ID).To(Equal(gifts.ID))
					})

					It("answers 404 for a product that is not there", func() {
						app.Put("/admin/products/999999999/categories", views.ProductCategoriesRequest{CategoryIDs: []int64{gifts.ID}}).
							AsUser(staff).Do().Status(http.StatusNotFound)
					})

					Context("when the product is in Gifts", func() {
						BeforeEach(func() {
							factories.Categorize(GinkgoT(), app.Store, product, gifts)
						})

						It("takes it out of the ones left out", func() {
							var set views.ProductCategoriesResponse
							app.Put(path+"/categories", views.ProductCategoriesRequest{CategoryIDs: []int64{drinkware.ID}}).
								AsUser(staff).Do().Status(http.StatusOK).JSON(&set)
							Expect(set.Categories).To(HaveLen(1))
							Expect(set.Categories[0].Name).To(Equal("Drinkware"))
						})

						It("takes it out of all of them when none are sent", func() {
							var set views.ProductCategoriesResponse
							app.Put(path+"/categories", views.ProductCategoriesRequest{CategoryIDs: []int64{}}).
								AsUser(staff).Do().Status(http.StatusOK).JSON(&set)
							Expect(set.Categories).To(BeEmpty())
						})

						It("refuses a category that does not exist, with 422", func() {
							res := app.Put(path+"/categories", views.ProductCategoriesRequest{CategoryIDs: []int64{drinkware.ID, 999999999}}).
								AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
							Expect(string(res.Body)).To(ContainSubstring("category 999999999 does not exist"))
						})

						It("changes nothing when it refuses", func() {
							app.Put(path+"/categories", views.ProductCategoriesRequest{CategoryIDs: []int64{drinkware.ID, 999999999}}).
								AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
							categories, err := app.Store.ListCategoriesForProduct(GinkgoT().Context(), product.ID)
							Expect(err).NotTo(HaveOccurred())
							Expect(categories).To(HaveLen(1))
							Expect(categories[0].ID).To(Equal(gifts.ID))
						})

						It("is closed to customers", func() {
							customer := factories.User(GinkgoT(), app.Store)
							app.Put(path+"/categories", views.ProductCategoriesRequest{CategoryIDs: []int64{}}).AsUser(customer).Do().Status(http.StatusForbidden)
							categories, err := app.Store.ListCategoriesForProduct(GinkgoT().Context(), product.ID)
							Expect(err).NotTo(HaveOccurred())
							Expect(categories).To(HaveLen(1))
						})
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
					var customer models.User

					BeforeEach(func() {
						customer = factories.User(GinkgoT(), app.Store)
					})

					expectUntouched := func() {
						var list views.AdminProductsResponse
						app.Get("/admin/products").AsUser(staff).Do().JSON(&list)
						Expect(list.Products).To(HaveLen(1))
						Expect(list.Products[0].ArchivedAt).To(BeNil())
					}

					It("cannot create a product", func() {
						app.Post("/admin/products", views.AdminProductRequest{Name: "Sneaky", PriceCents: 1, Stock: 1, ImageUrl: "/s"}).AsUser(customer).Do().Status(http.StatusForbidden)
						expectUntouched()
					})

					It("cannot archive a product", func() {
						app.Post(path+"/archive", nil).AsUser(customer).Do().Status(http.StatusForbidden)
						expectUntouched()
					})
				})

				Context("with twenty-one more products", func() {
					var list views.AdminProductsResponse

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
