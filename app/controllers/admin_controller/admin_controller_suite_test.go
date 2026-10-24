package admin_controller_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/test/testapp"
)

// app is the whole application on a clean test database, rebuilt before
// every spec.
var app *testapp.App

var _ = BeforeEach(func() {
	app = testapp.New(GinkgoT())
})

func TestAdmin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Admin")
}
