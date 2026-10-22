// Package config loads and validates settings.
//
// Two sources, in order: an encrypted credentials file if one exists, then
// environment variables, which always win. The file is the Rails pattern,
// committed ciphertext and a key that is never committed, so this service is
// configured the same way the Rails app next to it is.
//
// No globals and no init(): Load returns a struct, main passes it where it is
// needed. Every field is validated once, at startup, so a missing setting is a
// refusal to boot rather than a nil dereference an hour later.
package config

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/bogie-go/credentials/pkg/credentials"
)

// Env is a CLOSED set, mirroring RAILS_ENV. There is deliberately no staging:
// a staging deployment is production behaviour reading staging settings.
// Closing the set matters because every behavioural switch reads Env, and
// ECOMMERCE_ENV=staging would otherwise fall through to production
// behaviour silently. So would a typo.
const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"
)

// Two variables, deliberately separate:
//
//	ECOMMERCE_ENV            exactly development, test or production; selects BEHAVIOUR
//	ECOMMERCE_CREDENTIALS    any name; selects WHICH credentials file to read
//
// A staging deployment is ECOMMERCE_ENV=production ECOMMERCE_CREDENTIALS=staging:
// production strictness, staging secrets. CREDENTIALS defaults to ENV.
const (
	EnvVar         = "ECOMMERCE_ENV"
	CredentialsVar = "ECOMMERCE_CREDENTIALS"

	// MasterKeyVar carries the decryption key from the deployment, so no key
	// file ever has to exist in the image. It is RAILS_MASTER_KEY's
	// counterpart and must hold the key for whichever credentials file is
	// selected. The reader takes it directly: no file to write, no chmod to
	// get wrong.
	MasterKeyVar = "ECOMMERCE_MASTER_KEY"

	// ConfigDirVar points at the config directory, for tests and for running
	// the binary from somewhere other than the app root.
	ConfigDirVar = "ECOMMERCE_CONFIG_DIR"

	// TestDatabaseURLVar is the only variable that moves the test database.
	// Test never reads ECOMMERCE_DATABASE_URL: the suite truncates every
	// table and bin/ci drops the database, so a URL exported for development
	// would turn `bogie test` into a wipe of development data. Rails keeps
	// test in its own database.yml entry for the same reason.
	TestDatabaseURLVar = "ECOMMERCE_TEST_DATABASE_URL"
)

var envs = []string{EnvDevelopment, EnvTest, EnvProduction}
var logLevels = []string{"debug", "info", "warn", "error"}

// Config is everything ecommerce needs to run.
//
// Env, Source and KeyFile never come from the file: Env selects which file to
// read, so it cannot live inside one, and the other two describe the file
// that was read.
type Config struct {
	Env     string `mapstructure:"-"`
	Source  string `mapstructure:"-"`
	KeyFile string `mapstructure:"-"`

	// App is the ecommerce: section of the credentials file. The section name
	// gives the environment-variable prefix: ecommerce.addr is overridden by
	// ECOMMERCE_ADDR, so a value can move between the file and the
	// environment without renaming anything.
	App AppConfig `mapstructure:"ecommerce"`
}

// AppConfig is the service's own settings.
type AppConfig struct {
	// Addr is the listen address, default :8080.
	Addr string `mapstructure:"addr"`
	// LogLevel is debug, info, warn or error.
	LogLevel string `mapstructure:"log_level"`
	// ShutdownGrace is how long Start waits for in-flight requests on
	// SIGTERM, default 10s.
	ShutdownGrace time.Duration `mapstructure:"shutdown_grace"`
	// DatabaseURL points at the database. Development and test default to the
	// compose stack on 5440 and a database named after the environment, as
	// database.yml would; production has no default.
	DatabaseURL string `mapstructure:"database_url"`
	// MigrateAtBoot applies pending migrations before the server binds,
	// default true. Set it to false only for a deployment that migrates from
	// one place on purpose.
	MigrateAtBoot bool `mapstructure:"migrate_at_boot"`

	// Settings a generator added (bogie g authentication) go above the
	// marker, one field each, read from the same section and overridden by
	// the same ECOMMERCE_<KEY> rule.
	TokenSecret string `mapstructure:"token_secret"`

	// Mail. SMTPAddr is host:port of the server mail is delivered to:
	// development and test default to Mailpit from docker-compose (1125),
	// production has no default. SMTPUser and SMTPPassword are for a provider
	// that wants a login; Mailpit does not. MailFrom is the From address.
	SMTPAddr     string `mapstructure:"smtp_addr"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	MailFrom     string `mapstructure:"mail_from"`
	// bogie:config
}

// Load reads configuration and validates it.
//
// Where it reads from, in order:
//
//   - test: the environment only. Tests must not depend on anyone's secrets,
//     and the test database must never be the development one, so its URL
//     comes from ECOMMERCE_TEST_DATABASE_URL, never ECOMMERCE_DATABASE_URL.
//   - config/credentials/<ECOMMERCE_CREDENTIALS>.yml.enc with its own key
//     beside it, if that file exists. Staging and production live here, each
//     with a key that opens nothing else.
//   - config/credentials.yml.enc with config/master.key: the shared file,
//     which holds development.
//   - nothing: the environment only, for a container configured entirely
//     through its environment.
//
// Whichever applies, environment variables override the file.
func Load() (Config, error) {
	env := envOr(EnvVar, EnvDevelopment)
	credentialsEnv := envOr(CredentialsVar, env)

	// Hand the deployment's key to the credentials reader under the name it
	// looks for, without making every deployment learn that name.
	if key := os.Getenv(MasterKeyVar); key != "" && os.Getenv(credentials.EnvKeyVar) == "" {
		if err := os.Setenv(credentials.EnvKeyVar, key); err != nil {
			return Config{}, fmt.Errorf("config: %w", err)
		}
	}

	cfg := Config{Env: env, App: defaults(env)}

	dir := envOr(ConfigDirVar, "config")
	scoped, scopedKey := credentials.EnvironmentPaths(dir, credentialsEnv)
	shared, sharedKey := credentials.EnvironmentPaths(dir, "")

	switch {
	case env == EnvTest:
		cfg.Source = "environment only (test never reads a credentials file)"
		if err := applyEnv(&cfg.App); err != nil {
			return Config{}, err
		}
		cfg.App.DatabaseURL = envOr(TestDatabaseURLVar, defaultDatabaseURL(EnvTest))

	case exists(scoped), exists(shared):
		cfg.Source, cfg.KeyFile = shared, sharedKey
		if exists(scoped) {
			cfg.Source, cfg.KeyFile = scoped, scopedKey
		}
		// Flat everywhere: the shared file holds one environment's settings
		// exactly as a per-environment file does, so a key sits at the same
		// depth wherever it is written. The reader applies environment
		// overrides itself, so anything set in the environment still wins.
		reader := credentials.NewConfigReader(dir)
		reader.FlatShared = true
		if err := reader.Read(credentialsEnv, &cfg); err != nil {
			return Config{}, fmt.Errorf("config: %w", err)
		}

	default:
		cfg.Source = "environment only (no credentials file)"
		if err := applyEnv(&cfg.App); err != nil {
			return Config{}, err
		}
	}

	if os.Getenv(credentials.EnvKeyVar) != "" {
		cfg.KeyFile = "from " + MasterKeyVar
	}

	return cfg, cfg.validate()
}

// defaults is what every environment starts from. The credentials file and the
// environment change only what they name, so a key absent from both keeps its
// default rather than becoming a zero value.
func defaults(env string) AppConfig {
	return AppConfig{
		Addr:          ":8080",
		LogLevel:      "info",
		ShutdownGrace: 10 * time.Second,
		DatabaseURL:   defaultDatabaseURL(env),
		MigrateAtBoot: true,
		SMTPAddr:      defaultSMTPAddr(env),
		MailFrom:      defaultMailFrom(env),
	}
}

// defaultSMTPAddr is the compose stack's Mailpit outside production, and
// nothing in production: delivering real mail is a deliberate choice of
// provider, never a default.
func defaultSMTPAddr(env string) string {
	if env == EnvDevelopment || env == EnvTest {
		return "localhost:1125"
	}
	return ""
}

func defaultMailFrom(env string) string {
	if env == EnvDevelopment || env == EnvTest {
		return "Ecommerce <shop@ecommerce.test>"
	}
	return ""
}

// defaultDatabaseURL is the compose stack's database for the environment, and
// nothing for production, where the URL carries a password and must be given.
func defaultDatabaseURL(env string) string {
	switch env {
	case EnvDevelopment, EnvTest:
		return "postgres://postgres:postgres@localhost:5440/ecommerce_" + env + "?sslmode=disable"
	default:
		return ""
	}
}

// applyEnv is the no-credentials-file path: every setting straight from the
// environment, under the same names the file's section would produce. A value
// that does not parse is an error, never a silent fallback: the process would
// boot and behave as though you never set it.
func applyEnv(a *AppConfig) error {
	a.Addr = envOr("ECOMMERCE_ADDR", a.Addr)
	a.LogLevel = envOr("ECOMMERCE_LOG_LEVEL", a.LogLevel)
	a.DatabaseURL = envOr("ECOMMERCE_DATABASE_URL", a.DatabaseURL)
	if v := os.Getenv("ECOMMERCE_SHUTDOWN_GRACE"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("config: ECOMMERCE_SHUTDOWN_GRACE=%q is not a duration", v)
		}
		a.ShutdownGrace = d
	}
	if v := os.Getenv("ECOMMERCE_MIGRATE_AT_BOOT"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("config: ECOMMERCE_MIGRATE_AT_BOOT=%q is not a boolean", v)
		}
		a.MigrateAtBoot = b
	}
	a.TokenSecret = envOr("ECOMMERCE_TOKEN_SECRET", a.TokenSecret)
	a.SMTPAddr = envOr("ECOMMERCE_SMTP_ADDR", a.SMTPAddr)
	a.SMTPUser = envOr("ECOMMERCE_SMTP_USER", a.SMTPUser)
	a.SMTPPassword = envOr("ECOMMERCE_SMTP_PASSWORD", a.SMTPPassword)
	a.MailFrom = envOr("ECOMMERCE_MAIL_FROM", a.MailFrom)
	// bogie:env
	return nil
}

func (c Config) validate() error {
	if !slices.Contains(envs, c.Env) {
		return fmt.Errorf("config: %s=%q; must be one of %v", EnvVar, c.Env, envs)
	}
	a := c.App
	if !slices.Contains(logLevels, a.LogLevel) {
		return fmt.Errorf("config: ECOMMERCE_LOG_LEVEL=%q; must be one of %v", a.LogLevel, logLevels)
	}
	if a.Addr == "" {
		return fmt.Errorf("config: ECOMMERCE_ADDR is empty")
	}
	if a.ShutdownGrace <= 0 {
		return fmt.Errorf("config: ECOMMERCE_SHUTDOWN_GRACE must be a positive duration")
	}
	if a.DatabaseURL == "" {
		return fmt.Errorf("config: ECOMMERCE_DATABASE_URL is required in %s", c.Env)
	}
	if _, err := url.Parse(a.DatabaseURL); err != nil {
		return fmt.Errorf("config: ECOMMERCE_DATABASE_URL: %w", err)
	}
	if c.Env == EnvProduction {
		if a.SMTPAddr == "" {
			return fmt.Errorf("config: ECOMMERCE_SMTP_ADDR is required in production (host:port of your mail provider)")
		}
		if a.MailFrom == "" {
			return fmt.Errorf("config: ECOMMERCE_MAIL_FROM is required in production")
		}
	}
	// One check per generated setting; a missing secret refuses to boot
	// rather than serving open.
	for _, err := range []error{
		requireSecret(c.Env, "ECOMMERCE_TOKEN_SECRET", a.TokenSecret, 32),
		// bogie:validate
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

// requireSecret is the check for a secret a generator added: in production
// it must be set and at least min bytes long; elsewhere the generated
// development value or the test's own serves, so nothing is enforced.
func requireSecret(env, name, value string, min int) error {
	if env == EnvProduction && len(value) < min {
		return fmt.Errorf("config: %s must be set in production, at least %d characters", name, min)
	}
	return nil
}

// Production reports whether this process runs with production behaviour.
func (c Config) Production() bool { return c.Env == EnvProduction }

// PrintTo writes the effective settings, for the boot log. Straight to the
// writer, because slog would escape the newlines into one unreadable line. A
// failed write to stderr at boot has nowhere to be reported, so it is dropped.
func (c Config) PrintTo(w io.Writer) {
	_, _ = fmt.Fprintf(w, "ecommerce: env=%s settings=%q key=%q\n", c.Env, c.Source, c.KeyFile)
	_, _ = fmt.Fprintf(w, "ecommerce: addr=%s log_level=%s shutdown_grace=%s database=%s migrate_at_boot=%t\n",
		c.App.Addr, c.App.LogLevel, c.App.ShutdownGrace, redacted(c.App.DatabaseURL), c.App.MigrateAtBoot)
	_, _ = fmt.Fprintf(w, "ecommerce: smtp=%s mail_from=%q\n", c.App.SMTPAddr, c.App.MailFrom)
}

// redacted hides the password in a URL; the boot log is not where it belongs.
func redacted(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(unparseable)"
	}
	return u.Redacted()
}

func exists(path string) bool {
	_, err := os.Stat(filepath.Clean(path))
	return err == nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
