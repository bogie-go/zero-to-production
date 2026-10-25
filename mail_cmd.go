package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"ecommerce/app"
	"ecommerce/app/services/mailer"
	"ecommerce/config"
)

const mailUsage = `Usage: ecommerce mail test ADDRESS

Sends one test email through the configured SMTP server, to see that mail
works before anything depends on it. In development that is Mailpit from
docker-compose (make up): read it at http://localhost:8125.
`

// runMail handles `ecommerce mail ...`.
func runMail(ctx context.Context, args []string) error {
	if len(args) != 2 || args[0] != "test" {
		fmt.Fprint(os.Stderr, mailUsage)
		return fmt.Errorf("mail: expected: mail test ADDRESS")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	m := mailer.New(app.MailerConfig(cfg))
	err = m.Send(ctx, mailer.Message{
		To:      args[1],
		Subject: "Bogie Depot test email",
		Text:    "If you can read this, mail works.\n\nSent " + time.Now().Format(time.RFC1123) + " via " + cfg.Smtp.Address + "\n",
	})
	if err != nil {
		return err
	}
	fmt.Printf("sent a test email to %s via %s\n", args[1], cfg.Smtp.Address)
	return nil
}
