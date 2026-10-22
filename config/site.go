package config

// SiteConfig is the site: group of the credentials file. Each key is overridden by
// the environment variable named after it: site.url is SITE_URL.
//
// Written by `bogie g config`, once: `bogie g config site KEY` adds a key above
// the marker and never overwrites this file, so defaults is yours to edit.
type SiteConfig struct {
	Url string `mapstructure:"url"`
	// bogie:config:site
}

// defaults is what this group starts from in env. The credentials file and
// the environment change only the keys they name, so a key absent from both
// keeps what is set here. Per-environment policy goes here, by hand: a fake
// service in development, nothing in production.
func (c *SiteConfig) defaults(env string) {
	// The address links in emails point at. Development and test serve on
	// :8080; production has no default (it is required).
	if env == EnvDevelopment || env == EnvTest {
		c.Url = "http://localhost:8080"
	}
}

// validate refuses to boot in production without a required key.
func (c SiteConfig) validate(env string) error {
	for _, err := range []error{
		requireKey(env, "SITE_URL", c.Url != ""),
		// bogie:validate:site
	} {
		if err != nil {
			return err
		}
	}
	return nil
}
