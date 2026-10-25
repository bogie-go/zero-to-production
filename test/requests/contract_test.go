package requests_test

import (
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/test/testapp"
)

// docs/openapi.yaml is the API contract. Every request test already checks
// what it sends and receives against it; these two say the file and the
// router list the same operations, so neither can grow one the other lacks.
var _ = Describe("The API contract", func() {
	var spec, mounted []string

	BeforeEach(func() {
		var err error
		spec, err = testapp.ContractRoutes()
		Expect(err).NotTo(HaveOccurred())
		mounted = nil
		for _, r := range app.Server.Router.Routes() {
			mounted = append(mounted, r.Method+" "+r.Path)
		}
		sort.Strings(mounted)
	})

	It("describes every route the server mounts", func() {
		Expect(spec).To(ContainElements(mounted), "a route is mounted but missing from docs/openapi.yaml")
	})

	It("mounts every route it describes", func() {
		Expect(mounted).To(ContainElements(spec), "docs/openapi.yaml describes a route that is not mounted")
	})
})
