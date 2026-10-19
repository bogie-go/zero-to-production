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
