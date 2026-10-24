package mailers_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMailers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mailers")
}
