package models

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
)

// Whatever the id looks like, a order_item that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownOrderItemsAreNotFound(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetOrderItem(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetOrderItem(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeleteOrderItem(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteOrderItem(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	// A valid-looking order_item (OrderID and ProductID need not name real
	// rows: zero rows match id 999999999, so neither foreign key is ever
	// actually checked), so Validate passes and the not-found check is
	// what actually gets exercised.
	missing := domain.OrderItem{ID: 999999999, OrderID: 1, ProductID: 1, Quantity: 1, UnitPriceCents: 100}
	if _, err := store.UpdateOrderItem(ctx, missing); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateOrderItem(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

func TestOrderItemRoundTrip(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	user, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: "digest", Role: domain.RoleCustomer})
	if err != nil {
		t.Fatal(err)
	}
	order, err := store.CreateOrder(ctx, domain.Order{UserID: user.ID, Status: "pending"})
	if err != nil {
		t.Fatal(err)
	}
	product, err := store.CreateProduct(ctx, domain.Product{
		Name: "Widget", Description: "a widget", PriceCents: 500, Stock: 10, ImageUrl: "/widget.webp",
	})
	if err != nil {
		t.Fatal(err)
	}

	created, err := store.CreateOrderItem(ctx, domain.OrderItem{
		OrderID: order.ID, ProductID: product.ID, Quantity: 2, UnitPriceCents: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Error("CreateOrderItem did not assign an id")
	}

	got, err := store.GetOrderItem(ctx, strconv.FormatInt(created.ID, 10))
	if err != nil {
		t.Fatal(err)
	}
	if got != created {
		t.Errorf("GetOrderItem = %+v, want %+v", got, created)
	}

	all, err := store.ListOrderItems(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != created.ID {
		t.Errorf("ListOrderItems = %+v, want just the one created above", all)
	}

	created.Quantity = 3
	updated, err := store.UpdateOrderItem(ctx, created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Quantity != 3 {
		t.Errorf("UpdateOrderItem: quantity = %d, want 3", updated.Quantity)
	}

	if err := store.DeleteOrderItem(ctx, strconv.FormatInt(created.ID, 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetOrderItem(ctx, strconv.FormatInt(created.ID, 10)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetOrderItem after delete: err = %v, want ErrNotFound", err)
	}
}
