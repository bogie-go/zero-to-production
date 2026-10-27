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
	product := func(name string, categoryID *int64) models.Product {
		return factories.Product(GinkgoT(), store, func(p *models.Product) { p.Name = name; p.CategoryID = categoryID })
	}

	// The catalog joins the category's name on, filters by category when
	// asked, and an uncategorized product still appears (a LEFT JOIN).
	Describe("the active catalog", func() {
		var audio models.Category

		Context("with Audio, Peripherals and an uncategorized product", func() {
			BeforeEach(func() {
				audio = category("Audio")
				peripherals := category("Peripherals")
				product("Headphones", &audio.ID)
				product("Mouse", &peripherals.ID)
				product("Mystery Box", nil)
			})

			It("joins the category name onto every product, uncategorized included", func() {
				all, err := store.ListActiveProducts(GinkgoT().Context(), nil, 10, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(3))

				names := map[string]string{}
				for _, p := range all {
					names[p.Name] = p.CategoryName
				}
				Expect(names).To(Equal(map[string]string{"Headphones": "Audio", "Mouse": "Peripherals", "Mystery Box": ""}))

				one, err := store.GetActiveProduct(GinkgoT().Context(), strconv.FormatInt(all[0].ID, 10))
				Expect(err).NotTo(HaveOccurred())
				Expect(one.CategoryName).To(Equal(all[0].CategoryName))
			})

			It("filters by category", func() {
				only, err := store.ListActiveProducts(GinkgoT().Context(), &audio.ID, 10, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(only).To(HaveLen(1))
				Expect(only[0].Name).To(Equal("Headphones"))
				Expect(*only[0].CategoryID).To(Equal(audio.ID))
			})

			It("gives an empty list, not an error, for a category with nothing", func() {
				none := int64(999999999)
				Expect(store.ListActiveProducts(GinkgoT().Context(), &none, 10, 0)).To(BeEmpty())
			})
		})
	})

	// The counts are of ACTIVE products: an archived one does not make a
	// category look stocked, and an empty category still appears, with 0.
	Describe("ListCategoriesWithCounts", func() {
		Context("with Audio holding an active and an archived product, and an empty category", func() {
			BeforeEach(func() {
				audio := category("Audio")
				category("Empty")
				product("A", &audio.ID)
				b := product("B", &audio.ID)
				_, err := store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(b.ID, 10))
				Expect(err).NotTo(HaveOccurred())
			})

			It("counts only active products", func() {
				got, err := store.ListCategoriesWithCounts(GinkgoT().Context())
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(HaveLen(2))
				Expect(got[0].Name).To(Equal("Audio"), "ordered by name")
				counts := map[string]int64{}
				for _, c := range got {
					counts[c.Name] = c.ProductCount
				}
				Expect(counts).To(Equal(map[string]int64{"Audio": 1, "Empty": 0}))
			})
		})
	})

	// Deleting a category un-categorizes its products rather than refusing or
	// taking them along: ON DELETE SET NULL.
	Describe("deleting a category", func() {
		Context("with a category holding a product", func() {
			var audio models.Category
			var p models.Product

			BeforeEach(func() {
				audio = category("Audio")
				p = product("Headphones", &audio.ID)
			})

			It("keeps its products, without a category", func() {
				Expect(store.DeleteCategory(GinkgoT().Context(), strconv.FormatInt(audio.ID, 10))).To(Succeed())

				got, err := store.GetProduct(GinkgoT().Context(), strconv.FormatInt(p.ID, 10))
				Expect(err).NotTo(HaveOccurred())
				Expect(got.CategoryID).To(BeNil())
			})
		})
	})

	// A product pointed at a category that is not there is the caller's 422,
	// not a raw foreign-key error.
	Describe("a product's category", func() {
		ghost := int64(999999999)

		It("must exist on create", func() {
			_, err := store.CreateProduct(GinkgoT().Context(), models.Product{Name: "X", ImageUrl: "/x", CategoryID: &ghost})
			Expect(err).To(MatchError(models.ErrInvalid))
		})

		Context("with a product", func() {
			var p models.Product

			BeforeEach(func() {
				p = product("Y", nil)
			})

			It("must exist on update", func() {
				p.CategoryID = &ghost
				_, err := store.UpdateProduct(GinkgoT().Context(), p)
				Expect(err).To(MatchError(models.ErrInvalid))
			})
		})
	})
})
