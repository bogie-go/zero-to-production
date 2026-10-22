package mailers

import (
	"strings"
	"testing"

	"ecommerce/app/domain"
)

func TestWelcomeLinksToTheShop(t *testing.T) {
	m := Welcome(domain.User{Email: "ann@example.com"}, "https://shop.example.com/")
	if m.To != "ann@example.com" || m.Subject != "Welcome to Ecommerce" {
		t.Errorf("To = %q, Subject = %q", m.To, m.Subject)
	}
	// A trailing slash on the configured address does not double up.
	if !strings.Contains(m.Text, "Start shopping: https://shop.example.com\n") || !strings.Contains(m.HTML, `href="https://shop.example.com"`) {
		t.Errorf("link wrong:\n%s\n%s", m.Text, m.HTML)
	}
}

func TestWelcomeEscapesTheEmailInHTML(t *testing.T) {
	m := Welcome(domain.User{Email: `a"<b>@example.com`}, "http://x")
	if strings.Contains(m.HTML, "<b>") {
		t.Errorf("email not escaped: %s", m.HTML)
	}
}
