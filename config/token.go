package config

// TokenConfig is the token: group of the credentials file. Each key is overridden by
// the environment variable named after it: token.secret is TOKEN_SECRET.
//
// Written by `bogie g config`, once: `bogie g config token KEY` adds a key above
// the marker and never overwrites this file, so defaults is yours to edit.
type TokenConfig struct {
	Secret string `mapstructure:"secret"`
	// bogie:config:token
}

// defaults is what this group starts from in env. The credentials file and
// the environment change only the keys they name, so a key absent from both
// keeps what is set here. Per-environment policy goes here, by hand: a fake
// service in development, nothing in production.
func (c *TokenConfig) defaults(env string) {}

// validate refuses to boot in production without a required key.
func (c TokenConfig) validate(env string) error {
	for _, err := range []error{
		requireSecret(env, "TOKEN_SECRET", c.Secret, 32),
		// bogie:validate:token
	} {
		if err != nil {
			return err
		}
	}
	return nil
}
