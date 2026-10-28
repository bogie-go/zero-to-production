package admin_controllers_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("GET /admin/ops", func() {
	Context("without a token", func() {
		It("answers 401", func() {
			app.Get("/admin/ops").Do().Status(http.StatusUnauthorized)
		})
	})

	Context("as a customer", func() {
		It("answers 403", func() {
			customer := factories.User(GinkgoT(), app.Store)
			app.Get("/admin/ops").AsUser(customer).Do().Status(http.StatusForbidden)
		})
	})

	Context("as staff", func() {
		var staff models.User
		var ops views.OpsResponse

		BeforeEach(func() {
			staff = factories.Staff(GinkgoT(), app.Store)
		})

		Context("with nothing queued", func() {
			JustBeforeEach(func() {
				app.Get("/admin/ops").AsUser(staff).Do().Status(http.StatusOK).JSON(&ops)
			})

			It("names the release and the environment", func() {
				Expect(ops.Version).To(Equal("dev"))
				Expect(ops.Env).To(Equal("test"))
			})

			It("gives the schema version that is applied", func() {
				Expect(ops.SchemaVersion).To(BeNumerically(">", 0))
			})

			It("has an empty queue with no lag", func() {
				Expect(ops.Jobs).To(BeEmpty())
				Expect(ops.OldestWaitingSeconds).To(BeZero())
			})
		})

		Context("with a welcome email waiting in the queue", func() {
			BeforeEach(func() {
				app.Post("/users", views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusCreated)
			})

			It("counts it as available, with a lag", func() {
				app.Get("/admin/ops").AsUser(staff).Do().Status(http.StatusOK).JSON(&ops)

				Expect(ops.Jobs).To(HaveKeyWithValue("available", BeEquivalentTo(1)))
				Expect(ops.OldestWaitingSeconds).To(BeNumerically(">=", 0))
			})
		})
	})
})
