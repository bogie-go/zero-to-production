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
