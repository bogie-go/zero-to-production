package adapter_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("Categories and products", func() {

	category := func(name string) models.Category {
		return factories.Category(GinkgoT(), store, func(c *models.Category) { c.Name = name })
	}
	product := func(name string, categories ...models.Category) models.Product {
		p := factories.Product(GinkgoT(), store, func(p *models.Product) { p.Name = name })
		return factories.Categorize(GinkgoT(), store, p, categories...)
	}
	idOf := func(p models.Product) string { return strconv.FormatInt(p.ID, 10) }

	// The catalog filters by category when asked, and a product in several
	// categories is listed once however it is filtered.
	Describe("the active catalog", func() {
		var audio models.Category

		Context("with Audio, Peripherals, a product in both and an uncategorized one", func() {
			BeforeEach(func() {
				audio = category("Audio")
				peripherals := category("Peripherals")
				product("Headphones", audio)
				product("Mouse", peripherals)
				product("Webcam Mic", audio, peripherals)
				product("Mystery Box")
			})

			It("lists every product, uncategorized included, once each", func() {
				all, err := store.ListActiveProducts(GinkgoT().Context(), nil, 10, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(4))
			})

			It("filters by category", func() {
				only, err := store.ListActiveProducts(GinkgoT().Context(), &audio.ID, 10, 0)
				Expect(err).NotTo(HaveOccurred())
				names := []string{}
				for _, p := range only {
					names = append(names, p.Name)
				}
				Expect(names).To(ConsistOf("Headphones", "Webcam Mic"))
			})

			It("counts what the filter lists", func() {
				Expect(store.CountActiveProducts(GinkgoT().Context(), &audio.ID)).To(BeEquivalentTo(2))
			})

			It("gives an empty list, not an error, for a category with nothing", func() {
				none := int64(999999999)
				Expect(store.ListActiveProducts(GinkgoT().Context(), &none, 10, 0)).To(BeEmpty())
			})
		})
	})

	// A product's categories, by name: the "through" of the many-to-many.
	Describe("ListCategoriesForProduct", func() {
		Context("with a product in Stickers and Desk, and another in Desk", func() {
			var sticker models.Product

			BeforeEach(func() {
				stickers, desk := category("Stickers"), category("Desk")
				sticker = product("Logo Sticker", stickers, desk)
				product("Desk Mat", desk)
			})

			It("lists the product's categories by name", func() {
				got, err := store.ListCategoriesForProduct(GinkgoT().Context(), sticker.ID)
				Expect(err).NotTo(HaveOccurred())
				names := []string{}
				for _, c := range got {
					names = append(names, c.Name)
				}
				Expect(names).To(Equal([]string{"Desk", "Stickers"}))
			})
		})

		Context("with an uncategorized product", func() {
			It("is an empty list", func() {
				Expect(store.ListCategoriesForProduct(GinkgoT().Context(), product("Mystery Box").ID)).To(BeEmpty())
			})
		})
	})

	// The counts are of ACTIVE products: an archived one does not make a
	// category look stocked, and an empty category still appears, with 0.
	Describe("ListCategoriesWithCounts", func() {
		Context("with Audio holding an active and an archived product, Desk holding only an archived one, and an empty category", func() {
			BeforeEach(func() {
				audio, desk := category("Audio"), category("Desk")
				category("Empty")
				product("A", audio)
				b := product("B", audio)
				c := product("C", desk)
				for _, p := range []models.Product{b, c} {
					_, err := store.ArchiveProduct(GinkgoT().Context(), idOf(p))
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("counts only active products, and still lists every category", func() {
				got, err := store.ListCategoriesWithCounts(GinkgoT().Context())
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(HaveLen(3))
				Expect(got[0].Name).To(Equal("Audio"), "ordered by name")
				counts := map[string]int64{}
				for _, c := range got {
					counts[c.Name] = c.ProductCount
				}
				Expect(counts).To(Equal(map[string]int64{"Audio": 1, "Desk": 0, "Empty": 0}))
			})
		})

		Context("with a product in two categories", func() {
			BeforeEach(func() {
				product("Mat", category("Desk"), category("Gifts"))
			})

			It("counts it in each", func() {
				got, err := store.ListCategoriesWithCounts(GinkgoT().Context())
				Expect(err).NotTo(HaveOccurred())
				Expect(got[0].ProductCount).To(BeEquivalentTo(1))
				Expect(got[1].ProductCount).To(BeEquivalentTo(1))
			})
		})
	})

	// Deleting a category takes its memberships with it (ON DELETE CASCADE)
	// and leaves the products.
	Describe("deleting a category", func() {
		Context("with a category holding a product", func() {
			var audio models.Category
			var p models.Product

			BeforeEach(func() {
				audio = category("Audio")
				p = product("Headphones", audio)
			})

			It("keeps its products, without the category", func() {
				Expect(store.DeleteCategory(GinkgoT().Context(), strconv.FormatInt(audio.ID, 10))).To(Succeed())

				Expect(store.GetProduct(GinkgoT().Context(), idOf(p))).To(HaveField("ID", p.ID))
				Expect(store.ListCategoriesForProduct(GinkgoT().Context(), p.ID)).To(BeEmpty())
			})
		})
	})

	// A product's categories are set as a whole, in one transaction.
	Describe("SetProductCategories", func() {
		var gifts, drinkware models.Category
		var mug models.Product

		BeforeEach(func() {
			gifts, drinkware = category("Gifts"), category("Drinkware")
			mug = product("Mug")
		})

		names := func(p models.Product) []string {
			got, err := store.ListCategoriesForProduct(GinkgoT().Context(), p.ID)
			Expect(err).NotTo(HaveOccurred())
			out := []string{}
			for _, c := range got {
				out = append(out, c.Name)
			}
			return out
		}

		It("puts the product in each category once, however often it is named", func() {
			Expect(store.SetProductCategories(GinkgoT().Context(), idOf(mug), []int64{gifts.ID, drinkware.ID, gifts.ID})).To(Succeed())
			Expect(names(mug)).To(Equal([]string{"Drinkware", "Gifts"}))
		})

		It("leaves the caller's slice as it was", func() {
			ids := []int64{drinkware.ID, gifts.ID, drinkware.ID}
			Expect(store.SetProductCategories(GinkgoT().Context(), idOf(mug), ids)).To(Succeed())
			Expect(ids).To(Equal([]int64{drinkware.ID, gifts.ID, drinkware.ID}))
		})

		It("is not found for a product that is not there", func() {
			Expect(store.SetProductCategories(GinkgoT().Context(), "999999999", []int64{gifts.ID})).To(MatchError(models.ErrNotFound))
		})

		It("is not found for an id that is not a number", func() {
			Expect(store.SetProductCategories(GinkgoT().Context(), "nope", nil)).To(MatchError(models.ErrNotFound))
		})

		Context("when the product is in Gifts", func() {
			BeforeEach(func() {
				Expect(store.SetProductCategories(GinkgoT().Context(), idOf(mug), []int64{gifts.ID})).To(Succeed())
			})

			It("replaces them with the set it is given", func() {
				Expect(store.SetProductCategories(GinkgoT().Context(), idOf(mug), []int64{drinkware.ID})).To(Succeed())
				Expect(names(mug)).To(Equal([]string{"Drinkware"}))
			})

			It("leaves it in none for an empty set", func() {
				Expect(store.SetProductCategories(GinkgoT().Context(), idOf(mug), nil)).To(Succeed())
				Expect(names(mug)).To(BeEmpty())
			})

			It("is invalid for a category that is not there", func() {
				err := store.SetProductCategories(GinkgoT().Context(), idOf(mug), []int64{drinkware.ID, 999999999})
				Expect(err).To(MatchError(models.ErrInvalid))
				Expect(err).To(MatchError(ContainSubstring("category 999999999 does not exist")))
			})

			It("changes nothing when it is invalid", func() {
				Expect(store.SetProductCategories(GinkgoT().Context(), idOf(mug), []int64{drinkware.ID, 999999999})).NotTo(Succeed())
				Expect(names(mug)).To(Equal([]string{"Gifts"}))
			})
		})
	})
})
