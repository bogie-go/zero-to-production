package config_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/bogie-go/credentials/pkg/credentials"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/config"
)

// writeCredentials seals yaml into a fresh config dir and returns the dir.
func writeCredentials(yaml string) string {
	dir := GinkgoT().TempDir()
	key, err := credentials.GenerateMasterKey(filepath.Join(dir, "master.key"))
	Expect(err).NotTo(HaveOccurred())
	ed := credentials.NewConfigEditor(dir, "credentials.yml.enc", "master.key", "")
	Expect(ed.EncryptAndSave([]byte(yaml), hex.EncodeToString(key))).To(Succeed())
	return dir
}

// setProduction sets a complete, valid production environment; specs then
// blank or change the one setting under test.
func setProduction() {
	t := GinkgoT()
	t.Setenv(config.EnvVar, "production")
	t.Setenv(config.ConfigDirVar, t.TempDir()) // no credentials file anywhere
	t.Setenv("DATABASE_URL", "postgres://u:p@db:5432/ecommerce")
	t.Setenv("TOKEN_SECRET", strings.Repeat("s", 32))
	t.Setenv("SITE_URL", "https://shop.example.com")
	t.Setenv("SMTP_ADDRESS", "smtp.example.com:587")
	t.Setenv("MAIL_FROM", "shop@example.com")
}

var _ = Describe("Load", func() {
	var t GinkgoTInterface

	BeforeEach(func() {
		t = GinkgoT()
	})

	Context("in the test environment", func() {
		BeforeEach(func() {
			t.Setenv(config.EnvVar, "test")
			// Defaults are what an unset environment gets. A developer on their own
			// Postgres exports the test URL to run the suite, so clear it here.
			t.Setenv(config.TestDatabaseURLVar, "")
		})

		It("uses the defaults", func() {
			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())

			Expect(cfg.Env).To(Equal(config.EnvTest))
			Expect(cfg.Server.Addr).To(Equal(":8080"))
			Expect(cfg.Server.LogLevel).To(Equal("info"))
			Expect(cfg.Server.ShutdownGrace.Seconds()).To(BeNumerically("==", 10))
			Expect(cfg.Database.URL).To(Equal("postgres://postgres:postgres@localhost:5440/ecommerce_test?sslmode=disable"),
				"the compose stack's ecommerce_test")
			Expect(cfg.Database.MigrateAtBoot).To(BeTrue())
		})

		// The suite truncates every table and bin/ci drops the database, so test must
		// never follow a DATABASE_URL exported for development (or by another
		// app in the same shell); it moves only with its own variable.
		It("ignores the shared DATABASE_URL", func() {
			t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5440/ecommerce_development?sslmode=disable")

			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Database.URL).To(Equal("postgres://postgres:postgres@localhost:5440/ecommerce_test?sslmode=disable"))
		})

		It("follows its own database variable", func() {
			t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5440/ecommerce_development?sslmode=disable")
			t.Setenv(config.TestDatabaseURLVar, "postgres://ci:ci@db:5432/ecommerce_test")

			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Database.URL).To(Equal("postgres://ci:ci@db:5432/ecommerce_test"))
		})

		It("never reads a credentials file, even when one is right there", func() {
			t.Setenv(config.ConfigDirVar, writeCredentials("server:\n  addr: \":9999\"\n"))

			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Server.Addr).To(Equal(":8080"))
		})

		// Development and test deliver mail to Mailpit with no setup.
		It("delivers mail to Mailpit by default", func() {
			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Smtp.Address).To(Equal("localhost:1125"))
			Expect(cfg.Mail.From).NotTo(BeEmpty())
		})

		It("redacts the database password when printed", func() {
			t.Setenv(config.TestDatabaseURLVar, "postgres://app:hunter2@db:5432/ecommerce")
			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())

			var out strings.Builder
			cfg.PrintTo(&out)
			Expect(out.String()).NotTo(ContainSubstring("hunter2"))
			Expect(out.String()).To(ContainSubstring("app:xxxxx@db"))
		})

		It("refuses a bad shutdown grace", func() {
			t.Setenv("SERVER_SHUTDOWN_GRACE", "soon")
			_, err := config.Load()
			Expect(err).To(HaveOccurred())
		})

		It("refuses a bad migrate-at-boot flag", func() {
			t.Setenv("SERVER_SHUTDOWN_GRACE", "5s")
			t.Setenv("DATABASE_MIGRATE_AT_BOOT", "maybe")
			_, err := config.Load()
			Expect(err).To(HaveOccurred())
		})
	})

	// The set is closed on purpose: staging reads perfectly naturally and would
	// otherwise fall through to production behaviour silently.
	DescribeTable("refuses an unknown environment",
		func(env string) {
			t.Setenv(config.EnvVar, env)
			_, err := config.Load()
			Expect(err).To(HaveOccurred())
		},
		Entry("staging", "staging"),
		Entry("prod", "prod"),
		Entry("Production", "Production"),
	)

	// Development reads the shared file. A key the file names is taken from it, a
	// key it omits keeps its default, and the environment beats both.
	Context("in development with a shared credentials file", func() {
		It("reads the file, keeps defaults for absent keys, and lets the environment win", func() {
			dir := writeCredentials("server:\n  addr: \":9999\"\n  log_level: debug\n")
			t.Setenv(config.EnvVar, "development")
			t.Setenv(config.ConfigDirVar, dir)
			t.Setenv("SERVER_LOG_LEVEL", "warn")

			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Server.Addr).To(Equal(":9999"), "the file's addr")
			Expect(cfg.Server.LogLevel).To(Equal("warn"), "the environment's value over the file's debug")
			Expect(cfg.Server.ShutdownGrace.Seconds()).To(BeNumerically("==", 10))
			Expect(cfg.Database.MigrateAtBoot).To(BeTrue())
			Expect(cfg.Source).To(Equal(filepath.Join(dir, "credentials.yml.enc")))
		})

		// A deployment hands the key over in the environment and ships no key file.
		It("takes the master key from the environment when there is no key file", func() {
			dir := writeCredentials("server:\n  addr: \":7777\"\n")
			key, err := os.ReadFile(filepath.Join(dir, "master.key"))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Remove(filepath.Join(dir, "master.key"))).To(Succeed())
			t.Setenv(config.EnvVar, "development")
			t.Setenv(config.ConfigDirVar, dir)
			t.Setenv(config.MasterKeyVar, string(key))
			t.Setenv(credentials.EnvKeyVar, "")

			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Server.Addr).To(Equal(":7777"))
			Expect(cfg.KeyFile).To(ContainSubstring(config.MasterKeyVar))
		})
	})

	Context("in production", func() {
		BeforeEach(func() {
			setProduction()
		})

		It("boots when everything is set", func() {
			_, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
		})

		// Production carries a real password in its URL, so there is no default to
		// fall through to: the setting is required and its absence refuses to boot.
		It("requires a database", func() {
			t.Setenv("DATABASE_URL", "")
			_, err := config.Load()
			Expect(err).To(HaveOccurred())
		})

		// Mail must never silently go to localhost, or nowhere, in production.
		It("refuses to boot with neither smtp.address nor mail.from", func() {
			t.Setenv("SMTP_ADDRESS", "")
			t.Setenv("MAIL_FROM", "")
			_, err := config.Load()
			Expect(err).To(HaveOccurred())
		})

		// Each one is refused on its own, whichever group is checked first.
		It("requires smtp.address", func() {
			t.Setenv("SMTP_ADDRESS", "")
			_, err := config.Load()
			Expect(err).To(MatchError(ContainSubstring("SMTP_ADDRESS")))
		})

		It("requires mail.from", func() {
			t.Setenv("MAIL_FROM", "")
			_, err := config.Load()
			Expect(err).To(MatchError(ContainSubstring("MAIL_FROM")))
		})

		// A short token secret is refused: it signs every login.
		It("refuses a short token secret", func() {
			t.Setenv("TOKEN_SECRET", "too-short")
			_, err := config.Load()
			Expect(err).To(MatchError(ContainSubstring("TOKEN_SECRET")))
		})

		// Links in emails need the shop's address: refuse to boot without
		// site.url rather than mailing localhost links.
		It("requires the site URL", func() {
			t.Setenv("SITE_URL", "")
			_, err := config.Load()
			Expect(err).To(MatchError(ContainSubstring("SITE_URL")))
		})
	})
})
