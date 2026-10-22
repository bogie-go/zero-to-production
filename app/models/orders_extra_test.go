package models

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

func checkoutFixtures(t *testing.T, store *Store, stock int32) (user domain.User, cart domain.Cart, product domain.Product) {
	t.Helper()
	ctx := context.Background()

	user, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: "digest", Role: domain.RoleCustomer})
	if err != nil {
		t.Fatal(err)
	}
	cart, err = store.FindOrCreateCartByToken(ctx, "cart-token")
	if err != nil {
		t.Fatal(err)
	}
	product, err = store.CreateProduct(ctx, domain.Product{
		Name: "Widget", Description: "a widget", PriceCents: 500, Stock: stock, ImageUrl: "/widget.webp",
	})
	if err != nil {
		t.Fatal(err)
	}
	return user, cart, product
}

// A successful checkout decrements stock, snapshots the price into an
// order_item, and empties the cart — all three, since they are one
// transaction.
func TestCreateOrderFromCartSucceeds(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	user, cart, product := checkoutFixtures(t, store, 10)
	if _, err := store.AddCartItem(ctx, cart.ID, product.ID, 3); err != nil {
		t.Fatal(err)
	}

	order, lines, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID)
	if err != nil {
		t.Fatal(err)
	}
	if order.UserID != user.ID {
		t.Errorf("order.UserID = %d, want %d", order.UserID, user.ID)
	}
	if order.SubtotalCents != 1500 || order.TotalCents != 1500 {
		t.Errorf("order totals = %+v, want subtotal and total 1500 (no price rules seeded)", order)
	}
	if len(lines) != 1 || lines[0].ProductID != product.ID || lines[0].Quantity != 3 || lines[0].UnitPriceCents != 500 {
		t.Errorf("lines = %+v, want one line of 3x product %d at 500", lines, product.ID)
	}

	gotProduct, err := store.GetProduct(ctx, strconv.FormatInt(product.ID, 10))
	if err != nil {
		t.Fatal(err)
	}
	if gotProduct.Stock != 7 {
		t.Errorf("product stock after checkout = %d, want 7 (10 - 3)", gotProduct.Stock)
	}

	contents, err := store.CartContents(ctx, cart.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 0 {
		t.Errorf("cart contents after checkout = %v, want empty", contents)
	}
}

// The transaction's whole point: when one line can't be filled, nothing
// happens — not the stock decrement of any other line, not the order, not
// the cart clear. A flash sale where two shoppers grab the last item is
// exactly this race: AddCartItem's own stock check (a plain read then
// compare) only looks at stock at add-to-cart time, so by checkout time it
// can be stale.
func TestCreateOrderFromCartRollsBackOnInsufficientStock(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	user, cart, inStock := checkoutFixtures(t, store, 10)
	outOfStock, err := store.CreateProduct(ctx, domain.Product{
		Name: "Scarce", Description: "almost gone", PriceCents: 200, Stock: 1, ImageUrl: "/scarce.webp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddCartItem(ctx, cart.ID, inStock.ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddCartItem(ctx, cart.ID, outOfStock.ID, 1); err != nil {
		t.Fatal(err)
	}

	// Someone else buys the last unit between adding to cart and checking
	// out — simulated by taking the stock to zero directly.
	if _, err := store.pool.Exec(ctx, "UPDATE products SET stock = 0 WHERE id = $1", outOfStock.ID); err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("CreateOrderFromCart: err = %v, want ErrInvalid", err)
	}

	// inStock's decrement must have been rolled back too, even though its
	// own stock was sufficient.
	gotInStock, err := store.GetProduct(ctx, strconv.FormatInt(inStock.ID, 10))
	if err != nil {
		t.Fatal(err)
	}
	if gotInStock.Stock != 10 {
		t.Errorf("in-stock product's stock = %d, want 10 (rolled back)", gotInStock.Stock)
	}

	contents, err := store.CartContents(ctx, cart.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 2 {
		t.Errorf("cart contents after a failed checkout = %v, want both lines still there", contents)
	}

	orders, err := store.ListOrdersForUser(ctx, user.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 0 {
		t.Errorf("orders after a failed checkout = %v, want none", orders)
	}
}

func TestCreateOrderFromCartRejectsAnEmptyCart(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	user, cart, _ := checkoutFixtures(t, store, 10)
	if _, _, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("CreateOrderFromCart(empty cart): err = %v, want ErrInvalid", err)
	}
}

func TestGetOrderForUserIsScopedToItsOwner(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	user, cart, product := checkoutFixtures(t, store, 10)
	otherUser, err := store.CreateUser(ctx, domain.User{Email: "b@example.com", PasswordDigest: "digest", Role: domain.RoleCustomer})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddCartItem(ctx, cart.ID, product.ID, 1); err != nil {
		t.Fatal(err)
	}
	order, _, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.GetOrderForUser(ctx, otherUser.ID, strconv.FormatInt(order.ID, 10)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetOrderForUser(other user's order): err = %v, want ErrNotFound", err)
	}
	if _, _, err := store.GetOrderForUser(ctx, user.ID, strconv.FormatInt(order.ID, 10)); err != nil {
		t.Errorf("GetOrderForUser(owner): err = %v, want nil", err)
	}
}

// The hook runs inside the checkout's transaction, after the order exists,
// with that transaction: what it writes there commits with the order.
func TestOnOrderPlacedRunsInsideTheTransaction(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	user, cart, product := checkoutFixtures(t, store, 10)
	if _, err := store.AddCartItem(ctx, cart.ID, product.ID, 1); err != nil {
		t.Fatal(err)
	}

	var seen domain.Order
	var sawTx bool
	store.OnOrderPlaced(func(ctx context.Context, tx pgx.Tx, order domain.Order) error {
		seen, sawTx = order, tx != nil
		// The order is already visible on this transaction, not yet outside it.
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM orders WHERE id = $1", order.ID).Scan(&n); err != nil || n != 1 {
			t.Errorf("inside the transaction the order is not visible: n=%d err=%v", n, err)
		}
		if err := store.pool.QueryRow(ctx, "SELECT count(*) FROM orders WHERE id = $1", order.ID).Scan(&n); err != nil || n != 0 {
			t.Errorf("outside the transaction the order is visible before commit: n=%d err=%v", n, err)
		}
		return nil
	})

	order, _, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !sawTx || seen.ID != order.ID || seen.UserID != user.ID {
		t.Errorf("hook saw %+v (tx %v), want order %d for user %d", seen, sawTx, order.ID, user.ID)
	}
}

// A hook that fails undoes the checkout: no order, stock back, cart intact.
// This is what makes "the job exists if and only if the order does" true.
func TestAFailingOnOrderPlacedRollsBackTheCheckout(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	user, cart, product := checkoutFixtures(t, store, 10)
	if _, err := store.AddCartItem(ctx, cart.ID, product.ID, 3); err != nil {
		t.Fatal(err)
	}
	store.OnOrderPlaced(func(context.Context, pgx.Tx, domain.Order) error { return errors.New("queue is down") })

	if _, _, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID); err == nil {
		t.Fatal("CreateOrderFromCart succeeded although the hook failed")
	}
	if orders, _ := store.ListOrdersForUser(ctx, user.ID, 10); len(orders) != 0 {
		t.Errorf("orders = %v, want none", orders)
	}
	if got, _ := store.GetProduct(ctx, strconv.FormatInt(product.ID, 10)); got.Stock != 10 {
		t.Errorf("stock = %d, want 10", got.Stock)
	}
	if lines, _ := store.CartContents(ctx, cart.ID); len(lines) != 1 {
		t.Errorf("cart = %v, want the line still there", lines)
	}
}

// No checkout, no job: the hook only runs for an order that was written.
func TestOnOrderPlacedIsNotCalledWhenCheckoutFails(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	user, cart, _ := checkoutFixtures(t, store, 10) // an empty cart
	called := false
	store.OnOrderPlaced(func(context.Context, pgx.Tx, domain.Order) error { called = true; return nil })

	if _, _, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID); err == nil {
		t.Fatal("an empty cart checked out")
	}
	if called {
		t.Error("the hook ran for a checkout that failed")
	}
}
