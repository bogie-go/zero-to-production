package adapter_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("Archiving products", func() {

	archive := func(p models.Product) models.Product {
		archived, err := store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(p.ID, 10))
		Expect(err).NotTo(HaveOccurred())
		return archived
	}

	// An archived product leaves the shop (list and get) but the admin's own
	// listing still has it, and restoring puts it back.
	Context("with a product", func() {
		var p models.Product
		var id string

		BeforeEach(func() {
			p = factories.Product(GinkgoT(), store)
			id = strconv.FormatInt(p.ID, 10)
		})

		It("archives it", func() {
			Expect(archive(p).ArchivedAt).NotTo(BeNil())
		})

		Context("when it is archived", func() {
			var archived models.Product

			BeforeEach(func() {
				archived = archive(p)
			})

			It("hides it from the shop's get", func() {
				_, err := store.GetActiveProduct(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			})

			It("hides it from the shop's list", func() {
				Expect(store.ListActiveProducts(GinkgoT().Context(), nil, 10, 0)).To(BeEmpty())
			})

			It("keeps it in the admin's list", func() {
				all, err := store.ListProducts(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ArchivedAt).NotTo(BeNil())
			})

			It("keeps the first time when archived twice", func() {
				again, err := store.ArchiveProduct(GinkgoT().Context(), id)
				Expect(err).NotTo(HaveOccurred())
				Expect(*again.ArchivedAt).To(BeTemporally("==", *archived.ArchivedAt))
			})

			It("brings it back when restored", func() {
				restored, err := store.RestoreProduct(GinkgoT().Context(), id)
				Expect(err).NotTo(HaveOccurred())
				Expect(restored.ArchivedAt).To(BeNil())
				_, err = store.GetActiveProduct(GinkgoT().Context(), id)
				Expect(err).NotTo(HaveOccurred())
			})

			It("is not quietly brought back when it is edited", func() {
				archived.Stock = 3

				updated, err := store.UpdateProduct(GinkgoT().Context(), archived)

				Expect(err).NotTo(HaveOccurred())
				Expect(updated.ArchivedAt).NotTo(BeNil())
			})
		})
	})

	Describe("ArchiveProduct", func() {
		DescribeTable("returns ErrNotFound",
			func(id string) {
				_, err := store.ArchiveProduct(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			},
			Entry("when the id is not a number", "nope"),
			Entry("when the id is empty", ""),
			Entry("when no product has that id", "999999999"),
		)
	})

	Describe("a product in a cart or an order", func() {
		var user models.User
		var cart models.Cart
		var product models.Product

		Context("with a product, a user and a cart", func() {
			BeforeEach(func() {
				product = factories.Product(GinkgoT(), store, func(p *models.Product) { p.PriceCents = 500; p.Stock = 10 })
				user, cart = factories.User(GinkgoT(), store), factories.Cart(GinkgoT(), store)
			})

			// The database refuses, and the store answers ErrInvalid (a 422
			// upstream), not a raw error.
			Context("when it was ordered", func() {
				BeforeEach(func() {
					_, err := store.AddCartItem(GinkgoT().Context(), cart.ID, product.ID, 1)
					Expect(err).NotTo(HaveOccurred())
					_, _, err = store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)
					Expect(err).NotTo(HaveOccurred())
				})

				It("cannot be hard-deleted", func() {
					Expect(store.DeleteProduct(GinkgoT().Context(), strconv.FormatInt(product.ID, 10))).To(MatchError(models.ErrInvalid))
				})
			})

			// An archived product cannot be added to a cart, and one archived after
			// it was added blocks checkout without touching stock or the cart.
			Context("when it was added to the cart and then archived", func() {
				BeforeEach(func() {
					_, err := store.AddCartItem(GinkgoT().Context(), cart.ID, product.ID, 2)
					Expect(err).NotTo(HaveOccurred())
					archive(product)
				})

				It("cannot be added again", func() {
					_, err := store.AddCartItem(GinkgoT().Context(), cart.ID, product.ID, 1)
					Expect(err).To(MatchError(models.ErrInvalid))
				})

				It("blocks checkout", func() {
					_, _, err := store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)
					Expect(err).To(MatchError(models.ErrInvalid))
				})

				It("leaves stock and the cart untouched when checkout is refused", func() {
					ctx := GinkgoT().Context()
					_, _, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID)
					Expect(err).To(HaveOccurred())

					got, err := store.GetProduct(ctx, strconv.FormatInt(product.ID, 10))
					Expect(err).NotTo(HaveOccurred())
					Expect(got.Stock).To(BeEquivalentTo(10))
					Expect(store.CartContents(ctx, cart.ID)).To(HaveLen(1))
				})
			})
		})
	})
})

var _ = Describe("Paging products", func() {

	named := func(name string, categories ...models.Category) models.Product {
		p := factories.Product(GinkgoT(), store, func(p *models.Product) { p.Name = name })
		return factories.Categorize(GinkgoT(), store, p, categories...)
	}
	names := func(products []models.Product) []string {
		out := make([]string, len(products))
		for i, p := range products {
			out[i] = p.Name
		}
		return out
	}

	// Pages are slices of one total order: no product twice, none skipped,
	// newest first, and a page past the end is empty rather than an error. The
	// count follows the same filter, so the pager and the pages agree.
	Describe("the shop's pages", func() {
		var audio models.Category

		Context("with five Audio products, one other and one archived", func() {
			BeforeEach(func() {
				audio = factories.Category(GinkgoT(), store, func(c *models.Category) { c.Name = "Audio" })
				for _, n := range []string{"A", "B", "C", "D", "E"} {
					named(n, audio)
				}
				named("Other")
				gone := named("Gone")
				_, err := store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(gone.ID, 10))
				Expect(err).NotTo(HaveOccurred())
			})

			It("walks the whole list, newest first, leaving archived ones out", func() {
				var seen []string
				for offset := int32(0); offset < 6; offset += 2 {
					page, err := store.ListActiveProducts(GinkgoT().Context(), nil, 2, offset)
					Expect(err).NotTo(HaveOccurred())
					seen = append(seen, names(page)...)
				}
				Expect(seen).To(Equal([]string{"Other", "E", "D", "C", "B", "A"}))
			})

			It("gives an empty page past the end", func() {
				Expect(store.ListActiveProducts(GinkgoT().Context(), nil, 2, 100)).To(BeEmpty())
			})

			It("counts the same products the pages hold", func() {
				Expect(store.CountActiveProducts(GinkgoT().Context(), nil)).To(BeEquivalentTo(6))
				Expect(store.CountActiveProducts(GinkgoT().Context(), &audio.ID)).To(BeEquivalentTo(5))
			})
		})
	})

	// The admin's pages can hold the archived or leave them out, and the count
	// follows.
	Describe("the admin's pages", func() {
		Context("with a live and an archived product", func() {
			BeforeEach(func() {
				named("Live")
				gone := named("Gone")
				_, err := store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(gone.ID, 10))
				Expect(err).NotTo(HaveOccurred())
			})

			It("include archived products when asked, and count them", func() {
				all, err := store.ListProductsPage(GinkgoT().Context(), true, 10, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(2))
				Expect(store.CountProducts(GinkgoT().Context(), true)).To(BeEquivalentTo(2))

				second, err := store.ListProductsPage(GinkgoT().Context(), true, 1, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(names(second)).To(Equal([]string{"Live"}))
			})

			It("leave them out when asked, and count without them", func() {
				live, err := store.ListProductsPage(GinkgoT().Context(), false, 10, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(names(live)).To(Equal([]string{"Live"}))
				Expect(store.CountProducts(GinkgoT().Context(), false)).To(BeEquivalentTo(1))
			})
		})
	})
})
