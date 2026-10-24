package requests_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/views"
	"ecommerce/test/factories"
	"ecommerce/test/testapp"
)

// An unexpected failure (here the database going away) is a 500 that says
// "internal error" and nothing else: no SQL, no driver text, no stack.
var _ = Describe("An unexpected failure", func() {
	BeforeEach(func() {
		app.Store.Close()
	})

	DescribeTable("answers 500 without detail",
		func(request func() *testapp.Request) {
			res := request().AsGuest("someone").Do().Status(http.StatusInternalServerError)
			Expect(string(res.Body)).To(ContainSubstring("internal error"))
			Expect(string(res.Body)).NotTo(ContainSubstring("pool"))
		},
		Entry("the catalog", func() *testapp.Request { return app.Get("/products") }),
		Entry("the categories", func() *testapp.Request { return app.Get("/categories") }),
		Entry("the cart", func() *testapp.Request { return app.Get("/cart") }),
		Entry("adding to the cart", func() *testapp.Request {
			return app.Post("/cart_items", views.AddCartItemRequest{ProductID: 1, Quantity: 1})
		}),
	)

	It("answers a signup with 500 without detail", func() {
		res := app.Post("/users", views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusInternalServerError)
		Expect(string(res.Body)).To(ContainSubstring("internal error"))
	})

	It("answers a login with 500, not 401, when the database cannot be asked", func() {
		app.Post("/sessions", views.LoginRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusInternalServerError)
	})
})

var _ = Describe("Reading a request", func() {
	DescribeTable("answers 400 for a body that is not JSON",
		func(path string) {
			app.Post(path, "{not json").AsGuest("g").Do().Status(http.StatusBadRequest)
		},
		Entry("a signup", "/users"),
		Entry("a login", "/sessions"),
		Entry("a cart line", "/cart_items"),
	)
})

var _ = Describe("Sign-in alerts", func() {
	It("record where the login came from", func() {
		app.Post("/users", views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusCreated)

		app.Post("/sessions", views.LoginRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).
			WithHeader("User-Agent", "TestBrowser/1.0").Do().Status(http.StatusOK)

		args := app.JobArgs("send_sign_in_alert")
		Expect(args).To(HaveLen(1))
		Expect(args[0]).To(ContainSubstring("TestBrowser/1.0"))
	})
})

var _ = Describe("Signing up with a bad guest token", func() {
	It("still creates the account: a cart that cannot be merged is not a reason to refuse", func() {
		app.Post("/users", views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).
			WithAuthorization("Bearer not-a-real-token").Do().Status(http.StatusCreated)
	})

	It("also lets a login through", func() {
		factories.User(GinkgoT(), app.Store) // an unrelated account
		app.Post("/users", views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusCreated)
		app.Post("/sessions", views.LoginRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).
			WithAuthorization("Bearer not-a-real-token").Do().Status(http.StatusOK)
	})
})
