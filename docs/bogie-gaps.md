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

## No anonymous-but-trackable identity shape

`bogie g authentication` has three shapes — `secret`, `token`, `api_key` —
and all three are for identity that has already been established: a shared
secret, a user's bearer token, a machine client's key. None of them cover
the far more common case of a visitor nobody has identified yet, who still
needs to be the same visitor across requests — a guest cart, a shopping
session, an anonymous rate-limit bucket.

**Workaround in use:** `cartTokenHeader`, `currentToken` and `randomToken`
in `app/controllers/cart_controller.go` (task C2). The shape: the client
sends an opaque token if it has one; the server mints one with
`crypto/rand` if not; either way the token is echoed back as a response
header (`X-Cart-Token`), so the client's only job is "send what you have,
save what comes back." No cookie, deliberately — this is an API, and task
D's real auth will be token-based too, so this is a smaller version of the
same pattern rather than a second one.

**Real fix:** a fourth `g authentication` shape, maybe `anonymous`, that
writes the mint-or-reuse-and-echo middleware and the header name as a
config constant, the way `token`/`api_key` already write their own
middleware. Worth doing once a second app needs the same pattern — this is
the first time it has come up.
