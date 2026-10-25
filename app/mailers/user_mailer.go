package mailers

import (
	"fmt"
	"html"
	"strings"

	"ecommerce/app/domain"
	"ecommerce/app/services/mailer"
)

// Welcome is the email a person gets right after signing up. shopURL is where
// the shop lives (config site.url), so the link works in every environment.
func Welcome(user domain.User, shopURL string) mailer.Message {
	shop := strings.TrimRight(shopURL, "/")
	return mailer.Message{
		To:      user.Email,
		Subject: "Welcome to Bogie Depot",
		Text: fmt.Sprintf("Welcome to Bogie Depot, and thanks for creating an account.\n\nYour account is %s.\nStart shopping: %s\n",
			user.Email, shop),
		HTML: fmt.Sprintf("<p>Welcome to Bogie Depot, and thanks for creating an account.</p>\n<p>Your account is %s.</p>\n<p><a href=%q>Start shopping</a></p>\n",
			html.EscapeString(user.Email), shop),
	}
}
