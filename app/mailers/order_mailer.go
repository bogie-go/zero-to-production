// Package mailers composes the emails the shop sends: pure functions from
// domain values to a mailer.Message, no I/O, so they are easy to test and to
// preview. Delivering them is a job's work (app/jobs), over SMTP
// (app/services/mailer). The same split as a Rails mailer class and
// deliver_later.
package mailers

import (
	"fmt"
	"html"
	"strings"

	"ecommerce/app/domain"
	"ecommerce/app/services/mailer"
)

// OrderConfirmation is the email a customer gets after placing an order: what
// they ordered, at the prices saved with the order (never today's), and the
// total.
func OrderConfirmation(user domain.User, order domain.Order, lines []domain.OrderLine) mailer.Message {
	var text, rows strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&text, "  %s x %d    %s\n", l.ProductName, l.Quantity, money(l.UnitPriceCents*l.Quantity))
		fmt.Fprintf(&rows, "<tr><td>%s &times; %d</td><td align=\"right\">%s</td></tr>\n",
			html.EscapeString(l.ProductName), l.Quantity, money(l.UnitPriceCents*l.Quantity))
	}

	var totals, htmlTotals strings.Builder
	fmt.Fprintf(&totals, "Subtotal   %s\n", money(order.SubtotalCents))
	fmt.Fprintf(&htmlTotals, "<tr><td>Subtotal</td><td align=\"right\">%s</td></tr>\n", money(order.SubtotalCents))
	if order.DiscountCents > 0 {
		fmt.Fprintf(&totals, "Discounts  -%s\n", money(order.DiscountCents))
		fmt.Fprintf(&htmlTotals, "<tr><td>Discounts</td><td align=\"right\">-%s</td></tr>\n", money(order.DiscountCents))
	}
	fmt.Fprintf(&totals, "Total      %s\n", money(order.TotalCents))
	fmt.Fprintf(&htmlTotals, "<tr><td><strong>Total</strong></td><td align=\"right\"><strong>%s</strong></td></tr>\n", money(order.TotalCents))

	return mailer.Message{
		To:      user.Email,
		Subject: fmt.Sprintf("Your order #%d", order.ID),
		Text: fmt.Sprintf("Thanks for your order #%d.\n\n%s\n%s\nWe have received it and will be in touch.\n",
			order.ID, text.String(), totals.String()),
		HTML: fmt.Sprintf(`<p>Thanks for your order #%d.</p>
<table cellpadding="4">
%s%s</table>
<p>We have received it and will be in touch.</p>
`, order.ID, rows.String(), htmlTotals.String()),
	}
}

// money formats cents as dollars, the way every page of the shop does.
func money(cents int32) string { return fmt.Sprintf("$%d.%02d", cents/100, cents%100) }
