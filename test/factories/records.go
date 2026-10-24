package factories

import (
	"context"
	"testing"

	"ecommerce/app/domain"
	"ecommerce/app/models"
)

// User is a customer with a unique email. The password digest is a stand-in,
// not a real hash: use password.Hash in the test when logging in matters.
func User(t *testing.T, s *models.Store, with ...func(*domain.User)) domain.User {
	t.Helper()
	u := domain.User{
		Email:          Sequence("user") + "@example.com",
		PasswordDigest: "digest",
		Role:           domain.RoleCustomer,
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
func Staff(t *testing.T, s *models.Store, with ...func(*domain.User)) domain.User {
	t.Helper()
	return User(t, s, append([]func(*domain.User){func(u *domain.User) { u.Role = domain.RoleStaff }}, with...)...)
}

// Category has a unique name.
func Category(t *testing.T, s *models.Store, with ...func(*domain.Category)) domain.Category {
	t.Helper()
	c := domain.Category{Name: Sequence("Category ")}
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
func Product(t *testing.T, s *models.Store, with ...func(*domain.Product)) domain.Product {
	t.Helper()
	p := domain.Product{
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
func Cart(t *testing.T, s *models.Store, with ...func(*domain.Cart)) domain.Cart {
	t.Helper()
	c := domain.Cart{Token: Sequence("cart-")}
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
func PriceRule(t *testing.T, s *models.Store, with ...func(*domain.PriceRule)) domain.PriceRule {
	t.Helper()
	r := domain.PriceRule{
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
