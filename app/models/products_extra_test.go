package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/test/factories"
)

var _ = Describe("Archiving products", func() {

	archive := func(p domain.Product) domain.Product {
		archived, err := store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(p.ID, 10))
		Expect(err).NotTo(HaveOccurred())
		return archived
	}

	// An archived product leaves the shop (list and get) but the admin's own
	// listing still has it, and restoring puts it back.
	It("hides a product from the shop and restoring brings it back", func() {
		ctx := GinkgoT().Context()
		p := factories.Product(GinkgoT(), store)
		id := strconv.FormatInt(p.ID, 10)

		archived := archive(p)
		Expect(archived.ArchivedAt).NotTo(BeNil())
		_, err := store.GetActiveProduct(ctx, id)
		Expect(err).To(MatchError(domain.ErrNotFound))
		Expect(store.ListActiveProducts(ctx, nil, 10, 0)).To(BeEmpty())
		all, err := store.ListProducts(ctx, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(all).To(HaveLen(1))
		Expect(all[0].ArchivedAt).NotTo(BeNil())

		By("archiving twice keeps the first time")
		again, err := store.ArchiveProduct(ctx, id)
		Expect(err).NotTo(HaveOccurred())
		Expect(*again.ArchivedAt).To(BeTemporally("==", *archived.ArchivedAt))

		restored, err := store.RestoreProduct(ctx, id)
		Expect(err).NotTo(HaveOccurred())
		Expect(restored.ArchivedAt).To(BeNil())
		_, err = store.GetActiveProduct(ctx, id)
		Expect(err).NotTo(HaveOccurred())
	})

	It("does not quietly bring a product back when it is edited", func() {
		archived := archive(factories.Product(GinkgoT(), store))
		archived.Stock = 3

		updated, err := store.UpdateProduct(GinkgoT().Context(), archived)

		Expect(err).NotTo(HaveOccurred())
		Expect(updated.ArchivedAt).NotTo(BeNil())
	})

	DescribeTable("an unknown product cannot be archived",
		func(id string) {
			_, err := store.ArchiveProduct(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
		},
		Entry("a malformed id", "nope"),
		Entry("an empty id", ""),
		Entry("a well-formed id that names nothing", "999999999"),
	)

	Describe("a product in a cart or an order", func() {
		var user domain.User
		var cart domain.Cart
		var product domain.Product

		BeforeEach(func() {
			product = factories.Product(GinkgoT(), store, func(p *domain.Product) { p.PriceCents = 500; p.Stock = 10 })
			user, cart = factories.User(GinkgoT(), store), factories.Cart(GinkgoT(), store)
		})

		// The database refuses, and the store answers ErrInvalid (a 422
		// upstream), not a raw error.
		It("cannot be hard-deleted once ordered", func() {
			_, err := store.AddCartItem(GinkgoT().Context(), cart.ID, product.ID, 1)
			Expect(err).NotTo(HaveOccurred())
			_, _, err = store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)
			Expect(err).NotTo(HaveOccurred())

			Expect(store.DeleteProduct(GinkgoT().Context(), strconv.FormatInt(product.ID, 10))).To(MatchError(domain.ErrInvalid))
		})

		// An archived product cannot be added to a cart, and one archived after
		// it was added blocks checkout without touching stock or the cart.
		It("cannot be bought once archived", func() {
			ctx := GinkgoT().Context()
			_, err := store.AddCartItem(ctx, cart.ID, product.ID, 2)
			Expect(err).NotTo(HaveOccurred())
			archive(product)

			_, err = store.AddCartItem(ctx, cart.ID, product.ID, 1)
			Expect(err).To(MatchError(domain.ErrInvalid))
			_, _, err = store.CreateOrderFromCart(ctx, user.ID, cart.ID)
			Expect(err).To(MatchError(domain.ErrInvalid))

			got, err := store.GetProduct(ctx, strconv.FormatInt(product.ID, 10))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Stock).To(BeEquivalentTo(10))
			Expect(store.CartContents(ctx, cart.ID)).To(HaveLen(1))
		})
	})
})

var _ = Describe("Paging products", func() {

	named := func(name string, categoryID *int64) domain.Product {
		return factories.Product(GinkgoT(), store, func(p *domain.Product) { p.Name = name; p.CategoryID = categoryID })
	}
	names := func(products []domain.Product) []string {
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
		var audio domain.Category

		BeforeEach(func() {
			audio = factories.Category(GinkgoT(), store, func(c *domain.Category) { c.Name = "Audio" })
			for _, n := range []string{"A", "B", "C", "D", "E"} {
				named(n, &audio.ID)
			}
			named("Other", nil)
			gone := named("Gone", nil)
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

	// The admin's pages can hold the archived or leave them out, and the count
	// follows.
	Describe("the admin's pages", func() {
		BeforeEach(func() {
			named("Live", nil)
			gone := named("Gone", nil)
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
