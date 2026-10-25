package config

// MailConfig is the mail: group of the credentials file. Each key is overridden by
// the environment variable named after it: mail.from is MAIL_FROM.
//
// Written by `bogie g config`, once: `bogie g config mail KEY` adds a key above
// the marker and never overwrites this file, so defaults is yours to edit.
type MailConfig struct {
	From string `mapstructure:"from"`
	// bogie:config:mail
}

// defaults is what this group starts from in env. The credentials file and
// the environment change only the keys they name, so a key absent from both
// keeps what is set here. Per-environment policy goes here, by hand: a fake
// service in development, nothing in production.
func (c *MailConfig) defaults(env string) {
	if env == EnvDevelopment || env == EnvTest {
		c.From = "Bogie Depot <shop@bogiedepot.test>"
	}
}

// validate refuses to boot in production without a required key.
func (c MailConfig) validate(env string) error {
	for _, err := range []error{
		requireKey(env, "MAIL_FROM", c.From != ""),
		// bogie:validate:mail
	} {
		if err != nil {
			return err
		}
	}
	return nil
}
