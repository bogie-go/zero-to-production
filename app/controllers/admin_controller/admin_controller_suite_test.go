package admin_controller_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAdminController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AdminController")
}
