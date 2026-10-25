package mailers

import (
	"fmt"
	"html"

	"ecommerce/app/domain"
	"ecommerce/app/services/mailer"
)

// maxUserAgent keeps a hostile or absurd User-Agent from filling the email.
const maxUserAgent = 200

// SignInAlert is the "new sign-in" email: when, from where (the address the
// request came from) and with what (the browser's own description), so the
// owner can tell their own login from someone else's. Both come from the
// request and are untrusted, so the HTML part escapes them.
func SignInAlert(user domain.User, s domain.SignIn) mailer.Message {
	when := s.At.UTC().Format("Mon, 2 Jan 2006 15:04 MST")
	ip := orUnknown(s.IP)
	agent := orUnknown(truncate(s.UserAgent, maxUserAgent))
	return mailer.Message{
		To:      user.Email,
		Subject: "New sign-in to your Bogie Depot account",
		Text: fmt.Sprintf(`Someone just signed in to your Bogie Depot account.

  When:     %s
  From:     %s
  Browser:  %s

If this was you, nothing more to do. If it was not, reply to this email
and we will help you secure your account.
`, when, ip, agent),
		HTML: fmt.Sprintf(`<p>Someone just signed in to your Bogie Depot account.</p>
<table cellpadding="4">
<tr><td>When</td><td>%s</td></tr>
<tr><td>From</td><td>%s</td></tr>
<tr><td>Browser</td><td>%s</td></tr>
</table>
<p>If this was you, nothing more to do. If it was not, reply to this email and we will help you secure your account.</p>
`, html.EscapeString(when), html.EscapeString(ip), html.EscapeString(agent)),
	}
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
