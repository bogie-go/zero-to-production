package mailers

import (
	"strings"
	"testing"

	"ecommerce/app/domain"
)

var (
	ann   = domain.User{ID: 3, Email: "ann@example.com"}
	lines = []domain.OrderLine{
		{ProductName: "Wireless Mouse", UnitPriceCents: 1999, Quantity: 2},
		{ProductName: "Tom & <Jerry> Mug", UnitPriceCents: 500, Quantity: 1},
	}
)

func TestOrderConfirmationNamesTheOrderAndItsTotals(t *testing.T) {
	order := domain.Order{ID: 7, SubtotalCents: 4498, DiscountCents: 898, TotalCents: 3600}
	m := OrderConfirmation(ann, order, lines)

	if m.To != "ann@example.com" || m.Subject != "Your order #7" {
		t.Errorf("To = %q, Subject = %q", m.To, m.Subject)
	}
	for _, want := range []string{"order #7", "Wireless Mouse x 2    $39.98", "Subtotal   $44.98", "Discounts  -$8.98", "Total      $36.00"} {
		if !strings.Contains(m.Text, want) {
			t.Errorf("text lacks %q:\n%s", want, m.Text)
		}
	}
	if !strings.Contains(m.HTML, "$36.00") || !strings.Contains(m.HTML, "Wireless Mouse &times; 2") {
		t.Errorf("html = %s", m.HTML)
	}
}

// A product name comes from the admin, so the HTML part must escape it.
func TestOrderConfirmationEscapesHTML(t *testing.T) {
	m := OrderConfirmation(ann, domain.Order{ID: 1, TotalCents: 500}, lines[1:])
	if strings.Contains(m.HTML, "<Jerry>") || !strings.Contains(m.HTML, "Tom &amp; &lt;Jerry&gt; Mug") {
		t.Errorf("name not escaped: %s", m.HTML)
	}
}

func TestOrderConfirmationOmitsDiscountsThatDidNotHappen(t *testing.T) {
	m := OrderConfirmation(ann, domain.Order{ID: 1, SubtotalCents: 500, TotalCents: 500}, lines[1:])
	if strings.Contains(m.Text, "Discounts") || strings.Contains(m.HTML, "Discounts") {
		t.Errorf("a zero discount was listed:\n%s", m.Text)
	}
}

func TestMoney(t *testing.T) {
	for cents, want := range map[int32]string{0: "$0.00", 5: "$0.05", 1999: "$19.99", 100000: "$1000.00"} {
		if got := money(cents); got != want {
			t.Errorf("money(%d) = %q, want %q", cents, got, want)
		}
	}
}
