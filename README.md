# ecommerce

The Go service next to the Rails app. Generated with
[Bogie](https://github.com/bogie-go/bogie); the layout is Rails-shaped on
purpose and `AGENTS.md` says how to work in it.

    make up                   # postgres on :5440, via docker compose
    bogie db:prepare          # create, migrate, seed
    bogie s                   # server: migrates, serves, reloads on save; or: make server
    curl localhost:8080/healthz
    ECOMMERCE_ENV=test bogie db:prepare   # once: the test database
    bogie test                # or: make test
    bin/ci                    # every check, locally; green signs off the commit

`make help` lists the rest. `bogie` spells every command the way Rails does,
`db:migrate`, `db:rollback`, `credentials:edit`, and works from any
subdirectory; every command runs the app's own code.

In development `bogie server` and `bogie worker` rebuild and restart when a
watched file changes, which includes a new migration, since migrations are
embedded in the binary and applied at boot. `.air.toml` holds the settings;
delete it, or run `make server`, for a single boot with no watcher. Nothing
reloads outside development: production runs the compiled binary.

The worker runs beside it, in its own terminal and with the same reload:
`bogie worker`.

## Your own Postgres, without Docker

`make up` runs the bundled Postgres; any Postgres the app can reach will do.
Point development at yours (the role needs `CREATEDB`), or set `database.url`
with `bogie credentials:edit` instead of exporting it:

    export DATABASE_URL='postgres://USER:PASSWORD@localhost:5432/ecommerce_development?sslmode=disable'
    bogie db:prepare
    bogie s

Tests never read the credentials or DATABASE_URL; they take their
own URL:

    export ECOMMERCE_TEST_DATABASE_URL='postgres://USER:PASSWORD@localhost:5432/ecommerce_test?sslmode=disable'
    ECOMMERCE_ENV=test bogie db:prepare
    bogie test

`make up`, `make down`, `make psql` and the database step of `bin/ci` still use
Docker.

## Mail

`make up` also starts [Mailpit](https://mailpit.axllent.org), a fake SMTP
server: the app delivers mail to it exactly as it will to a real provider, and
nothing leaves your machine. Read what it caught at http://localhost:8125 (SMTP
on 1125; the usual 1025/8025 plus 100, since another project's Mailpit likely
holds those).

    go run . mail test you@example.com     # send one, then look in the inbox

Production has no default: set `smtp.address` (host:port of your provider) and
`mail.from`, plus `smtp.username` / `smtp.password` if it wants a login, with
`make credentials ENV=production` or the environment (`SMTP_ADDRESS`,
`MAIL_FROM`, `SMTP_USERNAME`, `SMTP_PASSWORD`). It refuses to boot without the
first two. Adding a setting like these is `bogie g config`.

## Background jobs

Jobs run on River, on the same Postgres, in their own process:

    make worker               # or: go run . worker; bogie worker adds live reload

Placing an order enqueues `send_order_confirmation` inside the order's own
transaction, so the email job exists if and only if the order does; the worker
loads the order and sends the email through the mailer. In development read it
at http://localhost:8125. The server alone never sends anything: without a
worker running, jobs just wait in `river_job`.

## Admin namespace

`db seed` creates one staff account, `staff@shop.test` /
`staff-password-please-change` — log in at `POST /sessions` like anyone
else, then use the token at `/admin/products`. Development-only: there is
no admin UI yet to promote a second person, or to change this password.
See `docs/bogie-gaps.md`.
