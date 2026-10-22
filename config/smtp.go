package config

// SmtpConfig is the smtp: group of the credentials file. Each key is overridden by
// the environment variable named after it: smtp.address is SMTP_ADDRESS.
//
// Written by `bogie g config`, once: `bogie g config smtp KEY` adds a key above
// the marker and never overwrites this file, so defaults is yours to edit.
type SmtpConfig struct {
	Address  string `mapstructure:"address"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	// bogie:config:smtp
}

// defaults is what this group starts from in env. The credentials file and
// the environment change only the keys they name, so a key absent from both
// keeps what is set here. Per-environment policy goes here, by hand: a fake
// service in development, nothing in production.
func (c *SmtpConfig) defaults(env string) {
	// Mailpit from docker-compose (1125); delivering real mail is a deliberate
	// choice of provider, so production has no default.
	if env == EnvDevelopment || env == EnvTest {
		c.Address = "localhost:1125"
	}
}

// validate refuses to boot in production without a required key.
func (c SmtpConfig) validate(env string) error {
	for _, err := range []error{
		requireKey(env, "SMTP_ADDRESS", c.Address != ""),
		// bogie:validate:smtp
	} {
		if err != nil {
			return err
		}
	}
	return nil
}
