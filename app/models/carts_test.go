package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
)

var _ = Describe("Carts", func() {

	// Whatever the id looks like, a cart that is not there is ErrNotFound, and
	// a caller cannot tell a malformed id from one that names nothing.
	DescribeTable("an unknown cart is not found",
		func(id string) {
			_, err := store.GetCart(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(store.DeleteCart(GinkgoT().Context(), id)).To(MatchError(domain.ErrNotFound))
		},
		Entry("a malformed id", "nope"),
		Entry("an empty id", ""),
		Entry("a well-formed id that names nothing", "999999999"),
	)

	It("cannot update one that is not there", func() {
		_, err := store.UpdateCart(GinkgoT().Context(), domain.Cart{ID: 999999999})
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	Describe("CreateCart", func() {
		It("creates a cart", func() {
			created, err := store.CreateCart(GinkgoT().Context(), domain.Cart{Token: "visitor-1"})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.ID).NotTo(BeZero())
		})
	})

	Context("with a created cart", func() {
		var created domain.Cart
		var id string

		BeforeEach(func() {
			var err error
			created, err = store.CreateCart(GinkgoT().Context(), domain.Cart{Token: "visitor-1"})
			Expect(err).NotTo(HaveOccurred())
			id = strconv.FormatInt(created.ID, 10)
		})

		It("reads it", func() {
			Expect(store.GetCart(GinkgoT().Context(), id)).To(Equal(created))
		})

		It("lists it", func() {
			all, err := store.ListCarts(GinkgoT().Context(), 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(all).To(HaveLen(1))
			Expect(all[0].ID).To(Equal(created.ID))
		})

		It("updates it", func() {
			created.Token = "visitor-2"
			updated, err := store.UpdateCart(GinkgoT().Context(), created)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Token).To(Equal("visitor-2"))
		})

		It("deletes it", func() {
			Expect(store.DeleteCart(GinkgoT().Context(), id)).To(Succeed())
			_, err := store.GetCart(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})
	})
})
