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
	var (
		t   GinkgoTInterface
		cfg config.Config
		err error
	)

	BeforeEach(func() {
		t = GinkgoT()
	})

	JustBeforeEach(func() {
		cfg, err = config.Load()
	})

	Context("in the test environment", func() {
		BeforeEach(func() {
			t.Setenv(config.EnvVar, "test")
			// Defaults are what an unset environment gets. A developer on their own
			// Postgres exports the test URL to run the suite, so clear it here.
			t.Setenv(config.TestDatabaseURLVar, "")
		})

		Context("with nothing else set", func() {
			It("uses the defaults", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(cfg.Env).To(Equal(config.EnvTest))
				Expect(cfg.Server.Addr).To(Equal(":8080"))
				Expect(cfg.Server.LogLevel).To(Equal("info"))
				Expect(cfg.Server.ShutdownGrace.Seconds()).To(BeNumerically("==", 10))
				Expect(cfg.Database.URL).To(Equal("postgres://postgres:postgres@localhost:5440/ecommerce_test?sslmode=disable"),
					"the compose stack's ecommerce_test")
				Expect(cfg.Database.MigrateAtBoot).To(BeTrue())
			})

			// Development and test deliver mail to Mailpit with no setup.
			It("delivers mail to Mailpit by default", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Smtp.Address).To(Equal("localhost:1125"))
				Expect(cfg.Mail.From).NotTo(BeEmpty())
			})
		})

		// The suite truncates every table and bin/ci drops the database, so test must
		// never follow a DATABASE_URL exported for development (or by another
		// app in the same shell); it moves only with its own variable.
		Context("with the shared DATABASE_URL set", func() {
			BeforeEach(func() {
				t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5440/ecommerce_development?sslmode=disable")
			})

			It("ignores it", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Database.URL).To(Equal("postgres://postgres:postgres@localhost:5440/ecommerce_test?sslmode=disable"))
			})

			Context("and its own database variable set", func() {
				BeforeEach(func() {
					t.Setenv(config.TestDatabaseURLVar, "postgres://ci:ci@db:5432/ecommerce_test")
				})

				It("follows its own variable", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(cfg.Database.URL).To(Equal("postgres://ci:ci@db:5432/ecommerce_test"))
				})
			})
		})

		Context("with a credentials file right there", func() {
			BeforeEach(func() {
				t.Setenv(config.ConfigDirVar, writeCredentials("server:\n  addr: \":9999\"\n"))
			})

			It("never reads it", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Server.Addr).To(Equal(":8080"))
			})
		})

		Context("with a password in the database URL", func() {
			BeforeEach(func() {
				t.Setenv(config.TestDatabaseURLVar, "postgres://app:hunter2@db:5432/ecommerce")
			})

			It("redacts the password when printed", func() {
				Expect(err).NotTo(HaveOccurred())

				var out strings.Builder
				cfg.PrintTo(&out)
				Expect(out.String()).NotTo(ContainSubstring("hunter2"))
				Expect(out.String()).To(ContainSubstring("app:xxxxx@db"))
			})
		})

		Context("with a bad shutdown grace", func() {
			BeforeEach(func() {
				t.Setenv("SERVER_SHUTDOWN_GRACE", "soon")
			})

			It("refuses to load", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a bad migrate-at-boot flag", func() {
			BeforeEach(func() {
				t.Setenv("SERVER_SHUTDOWN_GRACE", "5s")
				t.Setenv("DATABASE_MIGRATE_AT_BOOT", "maybe")
			})

			It("refuses to load", func() {
				Expect(err).To(HaveOccurred())
			})
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
		var dir string

		BeforeEach(func() {
			t.Setenv(config.EnvVar, "development")
		})

		Context("naming some keys, with one also set in the environment", func() {
			BeforeEach(func() {
				dir = writeCredentials("server:\n  addr: \":9999\"\n  log_level: debug\n")
				t.Setenv(config.ConfigDirVar, dir)
				t.Setenv("SERVER_LOG_LEVEL", "warn")
			})

			It("reads the file, keeps defaults for absent keys, and lets the environment win", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Server.Addr).To(Equal(":9999"), "the file's addr")
				Expect(cfg.Server.LogLevel).To(Equal("warn"), "the environment's value over the file's debug")
				Expect(cfg.Server.ShutdownGrace.Seconds()).To(BeNumerically("==", 10))
				Expect(cfg.Database.MigrateAtBoot).To(BeTrue())
				Expect(cfg.Source).To(Equal(filepath.Join(dir, "credentials.yml.enc")))
			})
		})

		// A deployment hands the key over in the environment and ships no key file.
		Context("without a key file but with the master key in the environment", func() {
			BeforeEach(func() {
				dir = writeCredentials("server:\n  addr: \":7777\"\n")
				key, readErr := os.ReadFile(filepath.Join(dir, "master.key"))
				Expect(readErr).NotTo(HaveOccurred())
				Expect(os.Remove(filepath.Join(dir, "master.key"))).To(Succeed())
				t.Setenv(config.ConfigDirVar, dir)
				t.Setenv(config.MasterKeyVar, string(key))
				t.Setenv(credentials.EnvKeyVar, "")
			})

			It("takes the master key from the environment", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Server.Addr).To(Equal(":7777"))
				Expect(cfg.KeyFile).To(ContainSubstring(config.MasterKeyVar))
			})
		})
	})

	Context("in production", func() {
		BeforeEach(func() {
			setProduction()
		})

		Context("when everything is set", func() {
			It("boots", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		// Production carries a real password in its URL, so there is no default to
		// fall through to: the setting is required and its absence refuses to boot.
		Context("without a database URL", func() {
			BeforeEach(func() {
				t.Setenv("DATABASE_URL", "")
			})

			It("refuses to boot", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		// Mail must never silently go to localhost, or nowhere, in production.
		Context("with neither smtp.address nor mail.from", func() {
			BeforeEach(func() {
				t.Setenv("SMTP_ADDRESS", "")
				t.Setenv("MAIL_FROM", "")
			})

			It("refuses to boot", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		// Each one is refused on its own, whichever group is checked first.
		Context("without smtp.address", func() {
			BeforeEach(func() {
				t.Setenv("SMTP_ADDRESS", "")
			})

			It("requires smtp.address", func() {
				Expect(err).To(MatchError(ContainSubstring("SMTP_ADDRESS")))
			})
		})

		Context("without mail.from", func() {
			BeforeEach(func() {
				t.Setenv("MAIL_FROM", "")
			})

			It("requires mail.from", func() {
				Expect(err).To(MatchError(ContainSubstring("MAIL_FROM")))
			})
		})

		// A short token secret is refused: it signs every login.
		Context("with a short token secret", func() {
			BeforeEach(func() {
				t.Setenv("TOKEN_SECRET", "too-short")
			})

			It("refuses it", func() {
				Expect(err).To(MatchError(ContainSubstring("TOKEN_SECRET")))
			})
		})

		// Links in emails need the shop's address: refuse to boot without
		// site.url rather than mailing localhost links.
		Context("without the site URL", func() {
			BeforeEach(func() {
				t.Setenv("SITE_URL", "")
			})

			It("requires it", func() {
				Expect(err).To(MatchError(ContainSubstring("SITE_URL")))
			})
		})
	})
})
