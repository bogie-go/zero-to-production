package factories

import (
	"context"

	"ecommerce/app/models"
	"ecommerce/db/adapter"
)

// User is a customer with a unique email. The password digest is a stand-in,
// not a real hash: use password.Hash in the test when logging in matters.
func User(t T, s *adapter.Store, with ...func(*models.User)) models.User {
	t.Helper()
	u := models.User{
		Email:          Sequence("user") + "@example.com",
		PasswordDigest: "digest",
		Role:           models.RoleCustomer,
	}
	for _, f := range with {
		f(&u)
	}
	created, err := s.CreateUser(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

// Staff is a User with the staff role.
func Staff(t T, s *adapter.Store, with ...func(*models.User)) models.User {
	t.Helper()
	return User(t, s, append([]func(*models.User){func(u *models.User) { u.Role = models.RoleStaff }}, with...)...)
}

// Category has a unique name.
func Category(t T, s *adapter.Store, with ...func(*models.Category)) models.Category {
	t.Helper()
	c := models.Category{Name: Sequence("Category ")}
	for _, f := range with {
		f(&c)
	}
	created, err := s.CreateCategory(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

// Product is in stock, uncategorized, and costs $10.00.
func Product(t T, s *adapter.Store, with ...func(*models.Product)) models.Product {
	t.Helper()
	p := models.Product{
		Name:        Sequence("Product "),
		Description: "a product",
		PriceCents:  1000,
		Stock:       10,
		ImageUrl:    "/x.webp",
	}
	for _, f := range with {
		f(&p)
	}
	created, err := s.CreateProduct(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

// Cart is an empty cart with a unique token.
func Cart(t T, s *adapter.Store, with ...func(*models.Cart)) models.Cart {
	t.Helper()
	c := models.Cart{Token: Sequence("cart-")}
	for _, f := range with {
		f(&c)
	}
	created, err := s.CreateCart(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

// PriceRule is an active $1.00 fixed discount.
func PriceRule(t T, s *adapter.Store, with ...func(*models.PriceRule)) models.PriceRule {
	t.Helper()
	r := models.PriceRule{
		Name:       Sequence("Rule "),
		Kind:       "fixed_discount",
		Properties: []byte(`{"amount_cents":100}`),
		Active:     true,
	}
	for _, f := range with {
		f(&r)
	}
	created, err := s.CreatePriceRule(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	return created
}
