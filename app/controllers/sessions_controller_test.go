package controllers_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
	"ecommerce/test/testapp"
)

var _ = Describe("POST /sessions", func() {
	creds := views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}
	var guest views.GuestSessionResponse
	var login views.SessionResponse
	var cart views.CartResponse

	Context("with a registered account", func() {
		BeforeEach(func() {
			app.Post("/users", creds).Do().Status(http.StatusCreated)
		})

		It("does not say which part was wrong", func() {
			wrong := app.Post("/sessions", views.LoginRequest{Email: "ada@example.com", Password: "wrong-password-here"}).Do().Status(http.StatusUnauthorized)
			unknown := app.Post("/sessions", views.LoginRequest{Email: "nobody@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusUnauthorized)

			Expect(wrong.Body).To(Equal(unknown.Body))
			Expect(app.Jobs("send_sign_in_alert")).To(BeZero())
		})

		It("records where the sign-in alert's login came from", func() {
			app.Post("/sessions", views.LoginRequest(creds)).
				WithHeader("User-Agent", "TestBrowser/1.0").Do().Status(http.StatusOK)

			args := app.JobArgs("send_sign_in_alert")
			Expect(args).To(HaveLen(1))
			Expect(args[0]).To(ContainSubstring("TestBrowser/1.0"))
		})

		Describe("the address in the sign-in alert", func() {
			login := func() *testapp.Request {
				return app.Post("/sessions", views.LoginRequest(creds)).WithHeader("X-Forwarded-For", "203.0.113.9")
			}

			Context("when the request comes from a proxy on a private network", func() {
				It("is the client the proxy forwarded", func() {
					login().WithRemoteAddr("172.18.0.5:4000").Do().Status(http.StatusOK)

					Expect(app.JobArgs("send_sign_in_alert")[0]).To(MatchRegexp(`"ip":\s*"203\.0\.113\.9"`))
				})
			})

			Context("when the request comes from anywhere else", func() {
				It("is the socket's address: a caller cannot choose what the alert says", func() {
					login().WithRemoteAddr("198.51.100.4:4000").Do().Status(http.StatusOK)

					Expect(app.JobArgs("send_sign_in_alert")[0]).To(MatchRegexp(`"ip":\s*"198\.51\.100\.4"`))
				})
			})
		})

		Context("with an unrelated account", func() {
			BeforeEach(func() {
				factories.User(GinkgoT(), app.Store)
			})

			It("lets a login through with a bad guest token", func() {
				app.Post("/sessions", views.LoginRequest(creds)).
					WithAuthorization("Bearer not-a-real-token").Do().Status(http.StatusOK)
			})
		})

		Context("with a guest who filled a cart with two of a product", func() {
			BeforeEach(func() {
				product := factories.Product(GinkgoT(), app.Store)
				app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
				app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 2}).
					WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated)

				app.Post("/sessions", views.LoginRequest(creds)).
					WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusOK).JSON(&login)
			})

			It("keeps the cart the guest filled", func() {
				app.Get("/cart").WithAuthorization("Bearer " + login.Token).Do().JSON(&cart)
				Expect(cart.Items).To(HaveLen(1))
				Expect(cart.Items[0].Quantity).To(BeEquivalentTo(2))
			})

			It("removes the guest's own cart", func() {
				app.Get("/cart").WithAuthorization("Bearer " + guest.Token).Do().JSON(&cart)
				Expect(cart.Items).To(BeEmpty())
			})
		})

		Context("with a guest cart where one line's product was archived", func() {
			var keep domain.Product

			BeforeEach(func() {
				var gone domain.Product
				keep, gone = factories.Product(GinkgoT(), app.Store), factories.Product(GinkgoT(), app.Store)
				app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
				for _, p := range []domain.Product{gone, keep} {
					app.Post("/cart_items", views.AddCartItemRequest{ProductID: p.ID, Quantity: 1}).
						WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated)
				}
				_, err := app.Store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(gone.ID, 10))
				Expect(err).NotTo(HaveOccurred())
			})

			It("keeps merging after one guest line cannot be merged", func() {
				app.Post("/sessions", views.LoginRequest(creds)).
					WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusOK).JSON(&login)

				app.Get("/cart").WithAuthorization("Bearer " + login.Token).Do().JSON(&cart)
				Expect(cart.Items).To(HaveLen(1))
				Expect(cart.Items[0].ProductID).To(Equal(keep.ID))
			})
		})
	})
})

var _ = Describe("POST /guest_sessions", func() {
	var guest views.GuestSessionResponse

	It("hands out a token", func() {
		app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
		Expect(guest.Token).NotTo(BeEmpty())
	})

	Context("with a guest token handed out", func() {
		BeforeEach(func() {
			app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
		})

		It("works as a guest's cart", func() {
			var cart views.CartResponse
			app.Get("/cart").WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusOK).JSON(&cart)
			Expect(cart.Items).To(BeEmpty())
		})
	})
})
