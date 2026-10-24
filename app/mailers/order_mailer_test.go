package mailers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/mailers"
)

var (
	ann   = domain.User{ID: 3, Email: "ann@example.com"}
	lines = []domain.OrderLine{
		{ProductName: "Wireless Mouse", UnitPriceCents: 1999, Quantity: 2},
		{ProductName: "Tom & <Jerry> Mug", UnitPriceCents: 500, Quantity: 1},
	}
)

var _ = Describe("OrderConfirmation", func() {
	It("names the order and its totals", func() {
		order := domain.Order{ID: 7, SubtotalCents: 4498, DiscountCents: 898, TotalCents: 3600}
		m := mailers.OrderConfirmation(ann, order, lines)

		Expect(m.To).To(Equal("ann@example.com"))
		Expect(m.Subject).To(Equal("Your order #7"))
		for _, want := range []string{"order #7", "Wireless Mouse x 2    $39.98", "Subtotal   $44.98", "Discounts  -$8.98", "Total      $36.00"} {
			Expect(m.Text).To(ContainSubstring(want))
		}
		Expect(m.HTML).To(ContainSubstring("$36.00"))
		Expect(m.HTML).To(ContainSubstring("Wireless Mouse &times; 2"))
	})

	// A product name comes from the admin, so the HTML part must escape it.
	It("escapes HTML", func() {
		m := mailers.OrderConfirmation(ann, domain.Order{ID: 1, TotalCents: 500}, lines[1:])
		Expect(m.HTML).NotTo(ContainSubstring("<Jerry>"))
		Expect(m.HTML).To(ContainSubstring("Tom &amp; &lt;Jerry&gt; Mug"))
	})

	It("omits discounts that did not happen", func() {
		m := mailers.OrderConfirmation(ann, domain.Order{ID: 1, SubtotalCents: 500, TotalCents: 500}, lines[1:])
		Expect(m.Text).NotTo(ContainSubstring("Discounts"))
		Expect(m.HTML).NotTo(ContainSubstring("Discounts"))
	})
})

var _ = DescribeTable("money",
	func(cents int32, want string) {
		Expect(mailers.Money(cents)).To(Equal(want))
	},
	Entry(nil, int32(0), "$0.00"),
	Entry(nil, int32(5), "$0.05"),
	Entry(nil, int32(1999), "$19.99"),
	Entry(nil, int32(100000), "$1000.00"),
)
