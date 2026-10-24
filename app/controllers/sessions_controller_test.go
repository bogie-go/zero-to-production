package controllers_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("POST /sessions", func() {
	creds := views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}

	BeforeEach(func() {
		app.Post("/users", creds).Do().Status(http.StatusCreated)
	})

	It("does not say which part was wrong", func() {
		wrong := app.Post("/sessions", views.LoginRequest{Email: "ada@example.com", Password: "wrong-password-here"}).Do().Status(http.StatusUnauthorized)
		unknown := app.Post("/sessions", views.LoginRequest{Email: "nobody@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusUnauthorized)

		Expect(wrong.Body).To(Equal(unknown.Body))
		Expect(app.Jobs("send_sign_in_alert")).To(BeZero())
	})

	It("keeps the cart a guest filled", func() {
		product := factories.Product(GinkgoT(), app.Store)
		var guest views.GuestSessionResponse
		app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
		app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 2}).
			WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated)

		var login views.SessionResponse
		app.Post("/sessions", views.LoginRequest(creds)).
			WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusOK).JSON(&login)

		var cart views.CartResponse
		app.Get("/cart").WithAuthorization("Bearer " + login.Token).Do().JSON(&cart)
		Expect(cart.Items).To(HaveLen(1))
		Expect(cart.Items[0].Quantity).To(BeEquivalentTo(2))

		By("removing the guest's own cart")
		app.Get("/cart").WithAuthorization("Bearer " + guest.Token).Do().JSON(&cart)
		Expect(cart.Items).To(BeEmpty())
	})

	It("keeps merging after one guest line cannot be merged", func() {
		keep, gone := factories.Product(GinkgoT(), app.Store), factories.Product(GinkgoT(), app.Store)
		var guest views.GuestSessionResponse
		app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
		for _, p := range []domain.Product{gone, keep} {
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: p.ID, Quantity: 1}).
				WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated)
		}
		_, err := app.Store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(gone.ID, 10))
		Expect(err).NotTo(HaveOccurred())

		var login views.SessionResponse
		app.Post("/sessions", views.LoginRequest(creds)).
			WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusOK).JSON(&login)

		var cart views.CartResponse
		app.Get("/cart").WithAuthorization("Bearer " + login.Token).Do().JSON(&cart)
		Expect(cart.Items).To(HaveLen(1))
		Expect(cart.Items[0].ProductID).To(Equal(keep.ID))
	})

	It("records where the sign-in alert's login came from", func() {
		app.Post("/sessions", views.LoginRequest(creds)).
			WithHeader("User-Agent", "TestBrowser/1.0").Do().Status(http.StatusOK)

		args := app.JobArgs("send_sign_in_alert")
		Expect(args).To(HaveLen(1))
		Expect(args[0]).To(ContainSubstring("TestBrowser/1.0"))
	})

	It("lets a login through with a bad guest token", func() {
		factories.User(GinkgoT(), app.Store) // an unrelated account
		app.Post("/sessions", views.LoginRequest(creds)).
			WithAuthorization("Bearer not-a-real-token").Do().Status(http.StatusOK)
	})
})

var _ = Describe("POST /guest_sessions", func() {
	It("hands out a token that works as a guest's cart", func() {
		var guest views.GuestSessionResponse
		app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
		Expect(guest.Token).NotTo(BeEmpty())

		var cart views.CartResponse
		app.Get("/cart").WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusOK).JSON(&cart)
		Expect(cart.Items).To(BeEmpty())
	})
})
