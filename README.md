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

## Your own Postgres, without Docker

`make up` runs the bundled Postgres; any Postgres the app can reach will do.
Point development at yours (the role needs `CREATEDB`), or set `database_url`
with `bogie credentials:edit` instead of exporting it:

    export ECOMMERCE_DATABASE_URL='postgres://USER:PASSWORD@localhost:5432/ecommerce_development?sslmode=disable'
    bogie db:prepare
    bogie s

Tests never read the credentials or ECOMMERCE_DATABASE_URL; they take their
own URL:

    export ECOMMERCE_TEST_DATABASE_URL='postgres://USER:PASSWORD@localhost:5432/ecommerce_test?sslmode=disable'
    ECOMMERCE_ENV=test bogie db:prepare
    bogie test

`make up`, `make down`, `make psql` and the database step of `bin/ci` still use
Docker.
