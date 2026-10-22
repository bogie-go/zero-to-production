package mailers

import (
	"strings"
	"testing"
	"time"

	"ecommerce/app/domain"
)

var signIn = domain.SignIn{UserID: 3, At: time.Date(2026, 10, 22, 14, 30, 0, 0, time.UTC), IP: "203.0.113.9", UserAgent: "Mozilla/5.0 (Mac)"}

func TestSignInAlertSaysWhenFromWhereAndWith(t *testing.T) {
	m := SignInAlert(domain.User{Email: "ann@example.com"}, signIn)
	if m.To != "ann@example.com" || m.Subject != "New sign-in to your account" {
		t.Errorf("To = %q, Subject = %q", m.To, m.Subject)
	}
	for _, want := range []string{"Thu, 22 Oct 2026 14:30 UTC", "203.0.113.9", "Mozilla/5.0 (Mac)"} {
		if !strings.Contains(m.Text, want) || !strings.Contains(m.HTML, want) {
			t.Errorf("missing %q in:\n%s\n%s", want, m.Text, m.HTML)
		}
	}
}

// The address and browser string come from the request, so the HTML part
// escapes them, and an absurd one cannot fill the email.
func TestSignInAlertDistrustsTheRequest(t *testing.T) {
	s := signIn
	s.UserAgent = `<script>alert(1)</script>` + strings.Repeat("x", 500)
	m := SignInAlert(domain.User{Email: "a@example.com"}, s)
	if strings.Contains(m.HTML, "<script>") || !strings.Contains(m.HTML, "&lt;script&gt;") {
		t.Errorf("user agent not escaped: %s", m.HTML)
	}
	if strings.Count(m.Text, "x") > maxUserAgent+20 {
		t.Errorf("user agent not truncated (%d x's)", strings.Count(m.Text, "x"))
	}
}

func TestSignInAlertToleratesAMissingAddress(t *testing.T) {
	s := signIn
	s.IP, s.UserAgent = "", ""
	if m := SignInAlert(domain.User{Email: "a@example.com"}, s); !strings.Contains(m.Text, "unknown") {
		t.Errorf("missing facts not shown as unknown:\n%s", m.Text)
	}
}
