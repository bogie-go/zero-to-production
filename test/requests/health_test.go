package requests_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A request test sends HTTP through the real router to the real application
// and the test database, and checks what comes back.
var _ = Describe("The health probes", func() {
	Describe("GET /healthz (liveness)", func() {
		It("answers ok", func() {
			app.Get("/healthz").Do().Status(http.StatusOK).Contains("ok")
		})

		Context("when the database is gone", func() {
			BeforeEach(func() {
				app.Store.Close()
			})

			It("still answers ok: the process is alive, which is all it says", func() {
				app.Get("/healthz").Do().Status(http.StatusOK)
			})
		})
	})

	Describe("GET /readyz (readiness)", func() {
		Context("when the database answers", func() {
			It("says ready, naming the check", func() {
				app.Get("/readyz").Do().Status(http.StatusOK).Contains(`"status":"ok"`).Contains(`"database":"ok"`)
			})
		})

		Context("when the database is gone", func() {
			BeforeEach(func() {
				app.Store.Close()
			})

			It("answers 503 so a new release does not take traffic", func() {
				app.Get("/readyz").Do().Status(http.StatusServiceUnavailable).Contains(`"database":"error"`)
			})

			It("never says why: the cause is for the log, not for a public probe", func() {
				res := app.Get("/readyz").Do()
				Expect(string(res.Body)).NotTo(ContainSubstring("pool"))
				Expect(string(res.Body)).NotTo(ContainSubstring("closed"))
			})
		})
	})
})
