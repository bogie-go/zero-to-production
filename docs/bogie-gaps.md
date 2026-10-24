# Bogie gaps

Things the app needed that Bogie's generators don't do yet, per CLAUDE.md:
note the gap here rather than working around it silently. Fixing Bogie is
part of the work — these are candidates for that, not permanent limits.

## No singular resource (Rails' `resource :x`)

Bogie's controller actions are the fixed Rails five, and `show`/`update`/
`destroy` always mount at `/:id`:

```go
"index":   {"GET", ""},
"show":    {"GET", "/:id"},
"update":  {"PATCH", "/:id"},
"destroy": {"DELETE", "/:id"},
```

There's no way to generate a singleton resource (`GET /cart`, no id) with
semantically-named actions, the way Rails' `resource :cart` (singular, no
`s`) gives you `show` at `/cart`, `update` at `/cart`, `destroy` at `/cart`,
no id anywhere. `g controller cart index` works as a stand-in — the route
it mounts (`GET /cart`) is identical to what a singular `show` would
produce — but the method ends up named `Index` for something that isn't a
list, which reads oddly.

**Workaround in use:** `app/controllers/cart_controller.go`, the `Index`
action, for the current cart (task C, cart mechanics).

**Real fix:** a singular-resource mode in the controller generator (maybe
`g controller cart --singular`), dropping `/:id` from `show`/`update`/
`destroy` and skipping `index`/`create` entirely for that controller.

## ~~No anonymous-but-trackable identity shape~~ — not a gap after all

Task C2 first assumed `bogie g authentication`'s three shapes — `secret`,
`token`, `api_key` — only cover identity that's already been established,
and hand-rolled a fourth mechanism for a guest cart: an opaque token the
client sends if it has one, minted with `crypto/rand` and echoed back as a
response header (`X-Cart-Token`) if not.

That turned out to be unnecessary. `g authentication token` already
generates exactly what a guest identity needs: `Maker.Mint(subject, ttl)`
to hand out a signed bearer token, and `RequireUser`/`Subject(c)` to guard
a route and read back whoever it was minted for. The only thing specific
to "guest" is that the subject is a fresh `uuid.NewString()` with no user
row behind it, minted from its own unguarded endpoint (`POST
/guest_sessions`) instead of a login form — everything after that is the
same `Authorization: Bearer` flow a real signed-in user would get.

**Current implementation:** `app/controllers/guest_sessions_controller.go`
mints; `app/middlewares/require_user.go` (generated) guards `/cart` and
`/cart_items` via `s.WithUser(...)` in `routes.go`.

No real fix needed here — a guest and a signed-in user are "same shape,
different stakes," not two mechanisms.

## AGENTS.md's id scheme line doesn't update with `bogie.toml`

`AGENTS.md` says "This app's id scheme is `uuid`," but `bogie.toml` has said
`id = "bigint"` since the switch before Task C (`4ad24f6`). `bogie app:update`
ran three times since (v0.2.9, v0.2.10, v0.2.11) and touched `AGENTS.md`
every time, but never updated that line. Low stakes — a reader just sees a
stale sentence, and `bogie.toml` is the actual source of truth — but it
means `AGENTS.md`'s id-scheme paragraph isn't really regenerated from
current settings on `app:update`, only on `bogie new`. Found while checking
which column types `g model` supports before generating `price_rule`, not
investigated further.

## ~~`:uniq` duplicates surfaced as a raw 500~~ — fixed in Bogie v0.2.11

`Create`/`Update` on a model with a `:uniq` column (`api_key.prefix` was the
only example before this) never classified a Postgres unique-violation: it
just wrapped the raw `pgconn.PgError` into an unclassified 500, the same as
any other database error. Found while adding `users.email:uniq` for Task D,
where a duplicate signup is an everyday, expected case, not a rare one like
`api_key`'s random collision-proof prefix.

**Fixed upstream**, not worked around: `internal/generate/model.go` now has
`Create`/`Update` catch error code `23505` and return `domain.ErrInvalid`
with a readable message ("email is already taken") for any model with at
least one `:uniq` attr. Released as Bogie v0.2.11, with a new golden test
(`model_user`) covering the path no existing fixture exercised.

## ~~No generated transaction helper~~ — fixed upstream in Bogie v0.2.12

`app/models/store.go` is hand-written infrastructure (one per app, not
per model), and until Task E it only wrapped the pool: `Open`, `Close`,
`Ping`. sqlc's own generated `Queries.WithTx(tx pgx.Tx) *Queries` has
existed since `bogie new` — that part is free — but nothing above it ever
began, committed, or rolled back a transaction. Checkout needed a real
one: decrement stock, write the order and its lines, and clear the cart
as one atomic unit, or undo all of it if any line can't be filled.

A hand-written, unexported `(s *Store) withTx(ctx, fn func(*Queries)
error) error` was added directly to `store.go` first, as a workaround
(`48133b7`'s era comment on `DeleteCartAndItems` — "real transactions
arrive in Task E" — is this).

**Fixed upstream**, not left as a permanent workaround: by the time the
transaction had a second caller (`DeleteCartAndItems`, Task D's
guest-cart merge, rewritten to use it too), the need was no longer
speculative, so `templates/app/app/models/store.go.tmpl` gained the same
`withTx` — unused on a fresh app the same way `parseID` is (the identical
`var _ = (*Store).withTx` quieting trick id.go already used), until a
hand-written composite-write method calls it. Released as Bogie v0.2.12,
documented in `AGENTS.md`'s new "Run a transaction" section, verified by
scaffolding a fresh app (plain and `--jobs`) and running its own
`bin/ci` clean with zero models. `bogie app:update` here merged it in,
one conflict (the hand-written version's comment vs. the template's;
kept the template's, since it's slightly more general).

## Generated `Create`/`Update` do not classify a foreign-key violation

The generated `Delete` maps a foreign-key violation to `domain.ErrInvalid`
since v0.2.13, but `Create` and `Update` on a model with a `:references`
column do not: a product pointed at a category that does not exist surfaced
as a raw 500. Found building categories (2026-10-22) and fixed by hand in
`app/models/products.go` (SQLSTATE 23503 becomes `ErrInvalid`, "category does
not exist"). **Real fix:** the same mapping in `g model`'s `Create`/`Update`
for any model with a `:references` attribute, named for the column.

## `bogie app:update` does not regenerate existing models

A generator fix to the code `g model` writes (the `:uniq` mapping in v0.2.11,
the foreign-key `Delete` in v0.2.13) reaches new models only. `app:update`
merges the layout's template files and leaves `app/models/<table>.go` alone,
so an app that wants the fix applies it by hand (done here for `products.go`;
the other models' `Delete` still has the old behavior). Re-running `g model`
would overwrite hand edits, which is the point of the `_extra` convention but
not an answer for edits to the generated file itself, like a nullable column.

## `g migration add_x_to_y` writes columns that cannot apply to a table with rows

`x:datetime` and `x:references` come out `NOT NULL` with no default, which
fails on any table that already has data (archived_at, category_id, both
edited by hand), and a nullable column or a `ON DELETE SET NULL` needs the
same edit. **Real fix:** a nullable form (`x:datetime?`), or a safe default
for an added column.

## A generated model test breaks as soon as `Validate` is real

Every model's generated "unknown ones are not found" test calls `Update` with
a zero-value struct. That passes while `Validate` is an empty stub and fails
the moment it has a rule (users, orders, order items, products, categories,
price rules and cart items have all hit it, each fixed by hand with a
valid-looking value). **Real fix:** generate a valid-looking value, or skip
the call and test the not-found path through a different method.

## `g job` registers a worker with only a logger

`jobs.Workers(log)` and the generated line `&XWorker{Log: log}` leave a real
worker's dependencies (a store, a mailer, a setting) to be widened by hand,
and the second job made it a `Deps` struct (`app/jobs/jobs.go`). **Real fix:**
generate the `Deps` struct, and the registration with the fields the worker
declares.

## No mailer shape

Bogie has no `g mailer`. The SMTP service (`app/services/mailer`), the
composing functions (`app/mailers`) and the Mailpit container in
`docker-compose.yml` are by hand, as is the config (now a `bogie g config`
group). Worth deciding whether Bogie should ship the shape.

## `go test ./...` runs packages in parallel against one test database

Once a second package (`test/requests`) used the test database, packages
truncating each other's tables made tests fail at random under `bin/ci`. Fixed
here with `-p 1` in `bin/ci` and `make test`; a package-per-database scheme
would be faster but heavier. **Real fix:** Bogie's `bin/ci` and Makefile
template use `-p 1`, or the scaffold gives each package its own database.
