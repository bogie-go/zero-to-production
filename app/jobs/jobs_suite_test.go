package jobs_test

import (
	"io"
	"log/slog"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/jobs"
)

func TestJobs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jobs")
}

var _ = Describe("Workers", func() {
	It("builds a bundle with the email workers", func() {
		workers, registered := jobs.Workers(slog.New(slog.NewTextHandler(io.Discard, nil)), jobs.Deps{})
		Expect(workers).NotTo(BeNil())
		Expect(registered).To(BeTrue())
	})
})
