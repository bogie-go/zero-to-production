package views_test

import (
	"ecommerce/app/views"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewPageRequest", func() {
	DescribeTable("accepts",
		func(page, per string, want views.PageRequest) {
			got, err := views.NewPageRequest(page, per, 12, 50)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(want))
		},
		Entry("no parameters: the defaults", "", "", views.PageRequest{1, 12}),
		Entry("a page only", "3", "", views.PageRequest{3, 12}),
		Entry("a page and a size", "2", "5", views.PageRequest{2, 5}),
		Entry("too much, which is brought down, not refused", "", "500", views.PageRequest{1, 50}),
	)

	DescribeTable("refuses",
		func(page, per string) {
			_, err := views.NewPageRequest(page, per, 12, 50)
			Expect(err).To(HaveOccurred())
		},
		Entry("page 0", "0", ""),
		Entry("a negative page", "-1", ""),
		Entry("a page that is not a number", "abc", ""),
		Entry("a fractional page", "1.5", ""),
		Entry("a page past the maximum", "10001", ""),
		Entry("per_page 0", "", "0"),
		Entry("a per_page that is not a number", "", "many"),
		Entry("a page that overflows", "99999999999", ""),
	)
})

var _ = Describe("PageRequest", func() {
	r := views.PageRequest{Page: 3, PerPage: 12}

	It("computes the offset", func() {
		Expect(r.Offset()).To(BeEquivalentTo(24))
	})

	DescribeTable("Pagination",
		func(total int64, pages int32) {
			Expect(r.Pagination(total)).To(Equal(views.Pagination{Page: 3, PerPage: 12, Total: total, TotalPages: pages}))
		},
		Entry("no items", int64(0), int32(0)),
		Entry("one item", int64(1), int32(1)),
		Entry("exactly one page", int64(12), int32(1)),
		Entry("one over a page", int64(13), int32(2)),
		Entry("exactly three pages", int64(36), int32(3)),
		Entry("one over three pages", int64(37), int32(4)),
	)

	// 10000 * 50 must still fit the int32 the database call takes.
	It("keeps the largest allowed offset within range", func() {
		off := views.PageRequest{Page: views.MaxPage, PerPage: 50}.Offset()
		Expect(off).To(BeEquivalentTo((views.MaxPage - 1) * 50))
		Expect(off).To(BeNumerically(">=", 0))
	})
})
