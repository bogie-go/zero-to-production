# Build notes

Raw material for the book "Bogie, How to Go from zero to production": what was actually done while building the e-commerce API, which Bogie commands were run, what was generated and what was written by hand, and where Bogie fell short. These are notes, not chapters.

Last updated: 2026-10-20 (covers through Task D, accounts). Times are Bangkok time (UTC+7). "(inferred)" marks things read from files, timestamps or the Bogie repo, not confirmed by Mac.

---

## Pass 1 › Task A: Show our products

### What was done

1. **2026-10-16 19:30, `4e262e8` initialize application.** New app from `bogie new`, layout v0.2.3. 44 files: Gin router, sqlc store, goose migrations, encrypted credentials, Kamal deploy config, Dockerfile, `bin/ci`, Makefile, AGENTS.md. No tables yet. `db/migrate`, `db/queries` and `db/seeds` hold only `.keep` files.
2. **2026-10-16 19:37, `087780d` add plan docs.** Added `docs/book-plan.md`. CLAUDE.md went from a one-line pointer to AGENTS.md to a full description of the companion repo. At this point the UI was planned as Svelte in `ui/`.
3. **2026-10-16 20:51, `191464e` generate Product model.** One migration (`20261016135032_create_products.sql`, timestamp 20:50:32), `db/queries/products.sql` (get, list, create, update, delete), `app/domain/product.go` with an empty `Validate`, `app/models/products.go` (store methods), the sqlc output (`models.go`, `products.sql.go`), and `products_test.go`. Columns: name (varchar), description (text), price_cents, stock (integer), image_url (varchar).
4. **2026-10-16 21:13, `8544fa7` add products controller.** `products_controller.go` with `Index` and `Show` stubs that return 501, a test that checks the routes are mounted, and three one-line registrations above the `bogie:controllers`, `bogie:wire` and `bogie:routes` markers.
5. **2026-10-16 21:14, `ef180da` update bogie v0.2.5.** `bogie.toml` and AGENTS.md now say v0.2.5. `main.go` gained a `routes` subcommand, the app's side of `bogie routes`.
6. **2026-10-16 21:25, `0db61c9` implement index and show.** The controller now takes a `productsStore` interface (GetProduct, ListProducts). New `app/views/products.go` with `ProductResponse` and `ProductsResponse` (`{"products": [...]}`). Added a `fail` helper: ErrNotFound becomes 404, anything else is logged and answered 500 "internal error". `NewProductsController(store, log)` on the wire line. Four new controller tests use a fake store.
7. **2026-10-17 ~11:18 (work), committed 19:36 in `42dca45` seeding product: upgrade to Bogie v0.2.6.** `bogie.toml` now says v0.2.6. Seeds changed from goose SQL files with their own `seed_db_version` table to plain Go functions. Changed files: `migrate.go` (seed source, `Seed`, `SeedDown` removed), `db/db.go` (`Seeds` embed removed), `cli.go` (`db seed` and `db prepare` call `seeds.Run` on a store; `db seed-down` removed), Makefile (`db-seed-down` and `g_seed_%` removed), `store_test.go` (no longer skips `seed_db_version`), and AGENTS.md (new "Add a seed" recipe). All these files share the mtime 11:18:10, so they were changed in one run (inferred).
8. **2026-10-17 ~11:18–11:31 (work), committed 19:36 in `42dca45`: product seeds.** `db/seeds/seeds.go` (the runner, with `SeedProduct` registered above `// bogie:seeds`) and `db/seeds/product.go`, filled by hand with 10 tech products (mouse, keyboard, monitor and so on, priced in cents, picsum.photos images). To keep the seed safe to re-run, a hand-written query `GetProductByName` was added to `products.sql` and recompiled with sqlc, plus two hand-written store methods in `app/models/products.go`: `GetProductByName` and `FindOrCreateProductByName` (Rails' `find_or_create_by`).
9. **2026-10-17 19:10 (work), committed 19:36 in `42dca45`: CLAUDE.md.** The UI plan changed from Svelte in `ui/` to plain HTML/CSS/JS with Alpine.js in `public/`, served by Bogie via `embed`, with no build step.
10. **2026-10-17 19:36 (work), committed 19:38 in `19c18a7` update bogie to v0.2.7 (static files).** Run 8 seconds after `42dca45` (inferred: `bogie app:update`). New: `public/404.html`, `public/500.html`, `public/public.go` (embeds `*.html` only), and `app/controllers/public.go`. In `routes.go`, `s.mountWelcome()` becomes `r.NoRoute(s.staticFile, s.welcome, s.notFound)`. Also changed: `welcome.go`, `routes_test.go`, `middlewares.go` (Recover serves `500.html`), AGENTS.md, and `bogie.toml` (v0.2.7). `public/public.go` still embedded only `*.html`.
11. **2026-10-17 19:59, `f6cd452` serve public/index.html at / and restyle the 404/500 pages.** Fixed a bug in the generated `staticFile`: it gave up on `GET /` before checking for `index.html`, so `/` always fell through to the welcome page. Added `public/index.html`, mascot art (`public/images/bogie-404.webp`, `bogie-500.webp`), and widened `go:embed` to `*.html images`. Bogie shipped the same fix upstream 38 seconds later (`a9d6cbf`, 20:00), so the app and Bogie were fixed side by side (inferred).
12. **2026-10-17 20:19, `4e4281f` product catalog front page.** `index.html` is an Alpine.js component: it fetches `GET /products` and renders a card grid with loading, error and empty states. Seeds now point at 10 locally generated WebP photos in `public/images/products/` instead of random picsum.photos images ("a wireless charger showed ocean waves").
13. **2026-10-17 20:26, `8067900` update bogie to v0.2.8.** `app:update` merged cleanly except `routes_test.go`, where the app's own image test already covered the case the template added. Kept the app's version.
14. **2026-10-17 20:44, `7fbdd93` product detail page.** `public/product.html` reads `?id=`, fetches `GET /products/:id`, and handles 404. Cards on the front page link to it. Static page only, no HTML views in the API.
15. **2026-10-17 20:52, `f7979a4` update build-note.** Mac recorded the "scaffold first, then explain" decision (below).

### Bogie commands used

- `bogie new ecommerce`, with `jobs = false`. Options are inferred from `bogie.toml`.
- `bogie g model product name:string description:text price_cents:integer stock:integer image_url:string` (inferred from the migration). Bogie v0.2.4, tagged 20:01 that evening, allows bare `name` to default to string, so the exact spelling may differ. `g model` runs sqlc itself.
- `bogie g controller products index show` (inferred: only those two routes exist, and the stubs match the generator's 501 text).
- `bogie app:update` to v0.2.5, then v0.2.6, then v0.2.7 (inferred), then v0.2.8 (the `8067900` message says so). The commit touched exactly the files the templates own, and `bogie.toml` says "Do not edit by hand".
- `bogie g seed product` (inferred). The doc comment "SeedProduct seeds product. db seed can run this any number of times" comes straight from Bogie's `internal/generate/seed.go` template.
- `make sqlc` (or `go tool sqlc generate`) after adding `GetProductByName`. `products.sql.go` was rewritten 3 seconds after `products.sql` was saved (inferred).
- Probably `bogie routes`, `bogie server`, `bogie db:prepare` / `db:seed` and `bogie test` to check the work. No direct evidence for these.
- No Bogie command for the UI pages: `public/index.html` and `product.html` are hand-written (with Claude Code, per the commit trailers).
- Not used: `bogie g scaffold product ...`. It would have written the model, views and the full five-action controller in one step. Model plus controller was chosen on purpose, it seems, so the reader sees each piece.

### Generated vs hand-written

Generated and untouched:
- Everything in `4e262e8` (`bogie new`).
- The migration, domain type, store methods, sqlc output and model test in `191464e`.
- The controller skeleton, route test and three marker registrations in `8544fa7`.
- The `routes` command in `main.go` and the v0.2.6 seed refactor (from `app:update`).
- `db/seeds/seeds.go`.

Hand-written, or hand-edited after generation:
- The `Index` and `Show` bodies, `productsStore`, `fail`, `productView` / `productsView`, `app/views/products.go` and the four controller tests in `0db61c9`. This code is almost word for word what `bogie g scaffold` produces: phrases like "Paging is yours to add when a client needs more" and the `productsListLimit = 100` constant appear in Bogie's `internal/generate/scaffold.go` and `scaffold_post.golden`. So it was most likely copied or adapted from scaffold output, or written by an AI agent following AGENTS.md (inferred).
- The wire line edited from `NewProductsController(log)` to `(store, log)`.
- The product list and loop in `db/seeds/product.go`. The stub came from the generator.
- `GetProductByName` in `db/queries/products.sql`, and `GetProductByName` / `FindOrCreateProductByName` added inside the generated `app/models/products.go`.
- `CLAUDE.md` and `docs/book-plan.md`.
- Everything in `public/` except the generated `404.html`, `500.html` and `public.go`: `index.html`, `product.html`, the mascot and product images. The 404/500 pages were restyled by hand, and the one-line fix in `app/controllers/public.go` (serve `index.html` at `/`) was made in the app at the same time as Bogie's own fix (inferred).

### Bogie gaps/bugs found

- **Seeds were redesigned during the build.** The v0.2.3 seeds were goose SQL files with a version table and `seed-down`. Rails has neither, and SQL seeds can't call app code such as `FindOrCreate`. Bogie commit `e27c65d` (2026-10-17 09:33, released as v0.2.6 at 10:28) made seeds plain Go, run every time. Going from v0.2.5 to v0.2.6 means `app:update` rewrites `migrate.go`, `cli.go`, `db/db.go`, the Makefile and the test helper. That is a big upgrade for a few-days-old app.
- **v0.2.5's `main.go` change is a feature, not a hand fix.** Bogie added `bogie routes` (`3f1c39e`, 21:10 on 10-16), a thin proxy to `go run . routes`. That needs a `runRoutes` function in the app's own `main.go`, so `app:update` merged one in. The upgrade was committed 4 minutes after that Bogie release, which suggests the gap was found and fixed while building Task A (inferred).
- **`string` used to become `text`.** Bogie v0.2.4 (`946f810`, 20:01 on 10-16, an hour before `g model`) fixed this: `string` now becomes `varchar` and a bare field name defaults to string. That is why `name` and `image_url` are varchar and `description` is text (inferred: found while preparing this model).
- **Hand-written store methods went into a generated file.** `GetProductByName` and `FindOrCreateProductByName` were committed in `42dca45` inside `app/models/products.go`, which `g model` rewrites whole. Running `g model` again would report a conflict. Bogie v0.2.7 (`ec8b3ea`, tagged today at 19:34) makes the convention a sibling file, `app/models/products_extra.go`. `app:update` doesn't move them. Done by hand in `4ad24f6` (2026-10-18): `products_extra.go` plus `db/queries/products_extra.sql`.
- **The model test's TODO points to a file that doesn't exist.** `products_test.go` says "see posts_test.go for the shape". That text is hard-coded in Bogie's model template (see `model_comment.golden`), but a real app has no `posts_test.go`. The comment in `app/application.go` also uses `posts.New(store)`, a service-based example, while AGENTS.md says scaffolds have no service layer. Both are small template bugs.
- **Leftover generated comments after hand-editing.** `Index` and `Show` still say "calls one service function", but they call the store directly.
- **`ImageUrl`, not `ImageURL`.** sqlc's default naming breaks Go's initialism convention, and the name spreads into the domain and view structs. The JSON tag (`image_url`) is fine. An sqlc `rename` setting or a Bogie default could fix it.
- **`GetProductByName` is `:one` on a column that isn't unique.** With duplicate names it returns whichever row Postgres finds first. The code comment admits that find-then-create can race. A `name:uniq` index, or `ON CONFLICT`, would make this sound.
- **No pagination yet.** The list stops at 100 (`productsListLimit`) and the response gives no count or cursor. `ListProducts` takes only a `limit`.
- **No static file support until v0.2.7.** v0.2.7 (`7e45718`, committed in the app as `19c18a7`) adds a `public/` folder embedded in the binary, served by a three-handler NoRoute chain, plus 404 and 500 pages. That commit also found a Gin bug pattern: a NoRoute handler must call `c.Abort()`, or the next handler appends its output too. This app needs v0.2.7 for the Alpine.js UI. v0.2.7's `staticFile` also skipped `public/index.html` for `GET /`; fixed in `f6cd452` and in Bogie `a9d6cbf` (v0.2.8), which also widened `go:embed` to `*.html images`. CSS and JS files are still not embedded. The pages dodge this with inline `<style>`/`<script>` and Alpine.js from the jsdelivr CDN, so the "no build step" promise holds, but the UI now needs the internet to load (inferred: a deliberate shortcut, not discussed). Small note: `mustRead` panics at package init, which bends rule 4 ("Only main exits").
- **Possible doc nit.** `make test` says "then ECOMMERCE_ENV=test bogie db:prepare once", but test databases are migrated by `testStore` anyway. Worth checking whether that step is still needed.

### Rails comparisons

- `bogie new` is `rails new --api`, plus Kamal, encrypted credentials and Rails 8.1-style `bin/ci` from day one. As with `rails new`, there are no tables at the start.
- `bogie g model` is `rails g model`, but it writes more pieces: a migration (goose SQL, not a Ruby DSL), named SQL queries (no ActiveRecord, no `where` builder), a domain struct with `Validate`, and a store that maps rows to domain values. The reader sees the layers ActiveRecord hides.
- Every column is NOT NULL with a default, so sqlc gives plain Go types instead of pointers. Rails columns allow NULL unless you say otherwise.
- `bogie g controller products index show` is `rails g controller`. Go has no autoload, so the generator adds three explicit registration lines (field, construction, mount) where Rails has `resources :products` plus autoloading.
- `app/views/products.go` plays the role of a Jbuilder or serializer file. Rails' `render json: @product` exposes the model directly. Bogie forbids that on purpose: domain structs are never wire structs, and the controller converts between them.
- `rescue_from ActiveRecord::RecordNotFound` becomes the `fail` helper mapping `domain.ErrNotFound` to 404.
- `bogie routes` is `rails routes`. `make db-prepare`, `db-migrate` and `db-rollback` use the Rails names. The app migrates at boot, unlike Rails.
- `db/seeds/*.go` plus `seeds.Run` is `db/seeds.rb`, and `FindOrCreateProductByName` is `find_or_create_by`. As in Rails, seeds run every time, so making them safe to repeat is the author's job.
- `bogie app:update` is `rails app:update`, but it does a real three-way merge against the version in `bogie.toml`.
- Since v0.2.7: `public/`, `404.html` and `500.html` like Rails' `public/` and `PublicExceptions`, but embedded in the binary.

### Chapter-worthy observations

- **The upgrade story is a chapter moment.** Twice in one day the framework changed under the app (v0.2.5, then v0.2.6), and `app:update` carried the app across. Readers will meet this in real life. Showing a `bogie.toml` version bump plus a merge is honest and useful.
- **Seeds show "Borrow the idea from Rails, write it the Go way" well.** The first version followed the Go habit of reusing the migration tool. The redesign went back to Rails' idea: run every time and stay safe to repeat. The DESIGN.md reasoning ("no mature Go seeding tool", "a version table has no path to a seed that calls app code") is good sidebar material.
- **Generate, run, read, hand-edit worked as the plan says.** `g controller` gave a 501 stub. Filling it in taught interfaces declared where they are used (rule 2), fakes in tests, wire structs and error mapping. A natural iteration split: A1 model, A2 controller stub plus `bogie routes`, A3 real index/show, A4 seeds.
- **Decided (2026-10-17): scaffold first, then explain.** In the book, readers run `bogie g scaffold product ...` to generate index/show (and the rest), then the chapter walks through the generated code piece by piece. This replaces the model + controller path used in pass 1.
- **Money as `price_cents integer`** is a good early design note for the price rules and orders tasks. Price at the time of order will matter in Task E.
- **The 500 test** ("connection refused to db-1" must not leak into the body) is a short, clear security lesson.
- **The UI decision moved from Svelte to Alpine.js in `public/`** to keep the "no Node" promise. That probably set up the static file work in Bogie v0.2.7.
- **The `c.Abort()` gotcha** from Bogie's v0.2.7 commit could be a "peek under the hood" box when static files arrive.

### Still to do

- [x] Commit the v0.2.6 upgrade and the product seeds (`42dca45`).
- [x] Move `GetProductByName` / `FindOrCreateProductByName` into `app/models/products_extra.go` (done in `4ad24f6`, with the query in `db/queries/products_extra.sql`).
- [x] Commit the v0.2.7 upgrade (`19c18a7`).
- [x] A basic HTML page listing products: `public/index.html` with Alpine.js (`4e4281f`), plus a detail page (`7fbdd93`).
- [x] Decide where `index.html` lives versus the welcome page: `public/index.html` wins at `/`, welcome is the fallback (`f6cd452`).
- [x] Start `docs/bogie-gaps.md` (`a21c9da`). The Task A gaps above are not in it yet.
- [ ] Create, update and delete actions (POST, PATCH, DELETE `/products`) with `Validate` rules (name required, price ≥ 0, stock ≥ 0) and 422 responses.
- [ ] Fill in the model round-trip test. The TODO is still in `products_test.go`, and now in `carts_test.go` and `cart_items_test.go` too.
- [ ] Pagination for `GET /products`: decide offset or cursor, and the response shape (count, next).
- [ ] Serving CSS and JS from `public/` (widen `go:embed`), or decide the CDN plus inline approach is fine for the book.
- [ ] Decide whether the Task A gaps above (seeds redesign, `posts_test.go` TODO, `ImageUrl`, non-unique `GetProductByName`) move into `docs/bogie-gaps.md`.
- [ ] Tag Task A checkpoints (`task-a1`, ...). Only `task-c1` and `task-c2` exist so far.

---

## Pass 1 › Task B: Group products into categories

Not started. The build went from Task A straight to Task C on 2026-10-18 (inferred: no category model, migration or commit exists, and the tags jump to `task-c1`).

---

## Pass 1 › Before Task C: v0.2.9, bigint ids, v0.2.10

### What was done

1. **2026-10-18 19:24, `f59ac32` update bogie to v0.2.9.** Clean merge. AGENTS.md gained a paragraph on the app's id scheme (this app: uuid). `app/models/uuid.go` gained a trailing blank line.
2. **2026-10-18 19:26, `4ad24f6` switch to bigint ids, and generate carts/cart_items.** `bogie.toml` gained `id = "bigint"`. The products migration was edited in place (`uuid ... DEFAULT gen_random_uuid()` became `bigserial`), and the products domain, store, view and tests were regenerated: ids are `int64` now. `uuid.go` was replaced by `id.go` (`parseID`). The products extras moved to `products_extra.go` / `products_extra.sql`. Also generated the cart models (see Task C1). Checked with a full schema reset and `bin/ci`. The UI needed no change: the JSON just carries a number.
3. **2026-10-18 19:44, `c26e861` update bogie to v0.2.10.** Regenerated products, carts and cart_items with `--force` to pick up the new "Written by `bogie g model`: overwritten whole on the next run" header on `db/queries/<table>.sql` and `app/models/<table>.go`. AGENTS.md now says hand-written queries go in `<table>_extra.sql` too.

### Bogie commands used

- `bogie app:update` to v0.2.9, then v0.2.10.
- `bogie g model product name:string description:text price_cents:integer stock:integer image_url:string --force` after setting `id = "bigint"` (inferred from "regenerated under it"). Again with `--force` for all three models in `c26e861`.
- `make sqlc` for the moved `products_extra.sql` (inferred).

### Generated vs hand-written

- Generated: the regenerated products files, `id.go`, the v0.2.10 header comments.
- Hand-written: the `id = "bigint"` line in `bogie.toml` (inferred: the file says "Do not edit by hand", and Bogie has no command to switch an existing app's scheme), the in-place edit of the products migration, and moving the extras into sibling files.

### Bogie gaps/bugs found

- **Bigint became the default in Bogie v0.2.9** (`22c0682`, 19:21 on 10-18, five minutes before `4ad24f6`). Before that every id was uuid, hard-coded, with no stated reason. Building it also exposed a scaffold bug: `Update` built `domain.Post{ID: c.Param("id")}` from the URL string, which only compiled because ID was a string.
- **No way to switch an existing app's id scheme.** An app without an `id` line reads as uuid. Switching meant hand-editing `bogie.toml`, editing an applied migration, regenerating with `--force` and resetting the database. Fine before production, but not a path the book should teach.
- **A fresh v0.2.9 app fails `bin/ci`.** `id.go`/`uuid.go` end in a blank line (gofmt) and `parseID` is unused until the first model (lint). This app shows the blank line in `f59ac32`. Fixed in Bogie `4d10633` (2026-10-19 09:13), after v0.2.10, so not released yet.
- **The v0.2.10 header comment leaks into sqlc output (inferred bug, not in `docs/bogie-gaps.md`).** sqlc attaches the file-header comment in `db/queries/<table>.sql` to the first query, so the Go doc comment on `GetProduct`, `GetCart` and `GetCartItem` in `*.sql.go` now reads "Written by `bogie g model`: overwritten whole... A hand-written query goes in ..._extra.sql". That is wrong on a sqlc function, and the query string also gains a blank line. Bogie's template should separate the header from the first `-- name:` (or put it somewhere sqlc ignores). Not fixed upstream as of `6be3be1`.

### Rails comparisons

- Bigint `id` is Rails' default. `--id=uuid` is like `config.generators { |g| g.orm :active_record, primary_key_type: :uuid }`, and like Rails it is one choice for the whole app.
- Editing an old migration and resetting is what Rails developers do before a first deploy too (`db:migrate:reset`). After that, a new migration is the only way.
- `_extra.go` siblings play the role Rails gets for free from reopening a class or a concern. Go has no reopening, so the convention is a separate file in the same package.

### Chapter-worthy observations

- **Bigint vs uuid is a real design note for later.** Integers read better in a teaching book, but sequential ids are guessable over a public API. Task C2 shows why that matters (see below).
- **`--force` plus `_extra` files is the "generated code you own" story.** Generated files can be regenerated whole only because hand-written code lives beside them.

### Still to do

- [ ] Report the sqlc doc-comment leak to Bogie (or add it to `docs/bogie-gaps.md`).
- [ ] Pick up Bogie's `4d10633` fix in the next release.

---

## Pass 1 › Task C1: Cart mechanics

### What was done

1. **2026-10-18 19:26, in `4ad24f6`: cart models.** Migrations `20261018122446_create_carts.sql` (id, token varchar) and `20261018122447_create_cart_items.sql` (cart_id and product_id as `bigint REFERENCES`, quantity integer, an index on each foreign key). Domain types, store methods, sqlc output and model tests for both. No logic yet.
2. **2026-10-18 19:53, `a21c9da` start docs/bogie-gaps.md.** First entry: no singular resource (see the gaps file).
3. **2026-10-18 20:11, `fe30a7f` cart mechanics.** `GET /cart`, `POST /cart_items`, `PATCH /cart_items/:id`, `DELETE /cart_items/:id`. Every cart_items action answers with the whole cart, so the UI just replaces its state. Hand-written: `FindOrCreateCartByToken` (`carts_extra.go`, `carts_extra.sql`), `AddCartItem` (adds to an existing line for the same product, checks quantity > 0 and stock), `UpdateCartItemQuantity`, `CartContents` (a JOIN with products, in `cart_items_extra.go` / `.sql`), `domain.CartLine` (read-only, no table), `app/views/cart.go`, and tests with fake stores. All visitors shared one cart (empty token).
4. **2026-10-18 20:14, `109d62b` cart UI.** A "Cart (N)" badge on both pages, an "Add to cart" button on `product.html` (disabled at stock 0, shows the server's error inline), and `public/cart.html` with a quantity stepper, remove link and server-computed total. Tagged `task-c1`.

### Bogie commands used

- `bogie g model cart token:string` and `bogie g model cart_item cart:references product:references quantity:integer` (inferred from the migrations, generated one second apart).
- `bogie g controller cart index` (the gaps file says so) and `bogie g controller cart_items create update destroy` (inferred: the `SetupRoutes` comment is the generator's, and only those three routes exist). The 501 stubs were never committed separately.
- `make sqlc` for the `_extra.sql` queries (inferred).

### Generated vs hand-written

- Generated: both models in full, the two controller skeletons and their marker registrations in `application.go` and `routes.go`.
- Hand-written: all cart logic, as CLAUDE.md requires ("cart logic is teaching material, never generated"): everything in `carts_extra*`, `cart_items_extra*`, `cart_line.go`, `views/cart.go`, the action bodies, `cartView` (line subtotals and the total, computed not stored), both controller tests, and `public/cart.html`.

### Bogie gaps/bugs found

- **No singular resource.** See [`docs/bogie-gaps.md`](bogie-gaps.md#no-singular-resource-rails-resource-x): `GET /cart` is an `Index` action.
- **Snake-case Go variable from the controller generator.** `cart_items := router.Group("/cart_items")` uses the table name as a variable name. It compiles, but Go style (and linters like revive) wants `cartItems` (inferred template bug).
- **Validation lives outside `Validate`.** `domain.CartItem.Validate` is still empty. Quantity and stock checks are in the store extras, so the generated `CreateCartItem`/`UpdateCartItem` still accept quantity 0 or negative if called directly. Stock needs the database, which is why it can't sit in `Validate`, but `quantity > 0` could (inferred design question).
- **`carts.token` has no unique index**, and `GetCartByToken` is `:one`. The find-then-create race noted in the code comment can create two carts with one token.
- **Comment mismatch.** `CartContents` says "newest first", but the query orders by `created_at` ascending (oldest first).
- **`ImageUrl` again.** The sqlc naming spreads into `CartLine` and `CartLineResponse`.

### Rails comparisons

- `cart:references` gives the foreign key and index, like `t.references`, but no `belongs_to` / `has_many`. The JOIN that Rails writes as `cart.cart_items.includes(:product)` is a hand-written sqlc query here.
- `FindOrCreateCartByToken` is `Cart.find_or_create_by(token:)`.
- Adding the same product twice increments the line. That is exactly the Depot app's `add_product` in *Agile Web Development with Rails*, which this book is modelled on.
- `422` for validation errors, as in Rails.

### Chapter-worthy observations

- **Depot parallels.** The cart task maps almost one to one onto the classic Rails Depot cart, which gives Rails readers a familiar anchor for "Coming from Rails?" boxes.
- **"Answer with the whole cart"** is a good small API design lesson: the client never merges state.
- **Derived values are computed, not stored** (`cartView`). The total depends on current prices, which sets up "price at the time of order" in Task E.
- **Stock is checked when adding, not reserved.** Two shoppers can both add the last item. Good setup for Task E's conditional stock `UPDATE`.

### Still to do

- [ ] Scope cart_items changes to the visitor's cart (see C2, this became a real bug there).
- [ ] Unique index on `carts.token`, or `ON CONFLICT`.
- [ ] Move `quantity > 0` into `CartItem.Validate`, or decide to keep it in the store and say why.
- [ ] Fix the `CartContents` comment.

---

## Pass 1 › Task C2: Each visitor their own cart

### What was done

1. **2026-10-18 20:46, `0c7693c` give each visitor their own cart.** `currentToken` reads an `X-Cart-Token` request header. If there is none it mints 32 random bytes (`crypto/rand`, hex), and either way echoes the token back in the response. No cookie, on purpose: this is an API, and Task D auth will be token-based too. The UI got a small `cartFetch()` wrapper per page that sends the token from `localStorage` and saves whatever comes back. Checked live: first visit mints a token, a reload reuses it (one cart row), another token sees an empty cart. Added the second entry to `docs/bogie-gaps.md`. Tagged `task-c2`.

### Bogie commands used

- None. All hand-written.

### Generated vs hand-written

- Hand-written: `cartTokenHeader`, `currentToken`, `randomToken`, the two new tests, `cartFetch()` in all three pages, and the gaps entry.

### Bogie gaps/bugs found

- **No "anonymous but trackable" identity shape in `g authentication`.** See [`docs/bogie-gaps.md`](bogie-gaps.md#no-anonymous-but-trackable-identity-shape).
- **App bug: `PATCH` and `DELETE /cart_items/:id` don't check the line belongs to the caller's cart.** `UpdateCartItemQuantity` and `DeleteCartItem` look up the line by id only. With per-visitor carts and sequential bigint ids, any visitor can change or delete anyone's cart line by guessing ids. `Update` then answers with the *other* cart's full contents (`respondWithCart(item.CartID)`), so it also leaks a stranger's cart. Not a Bogie bug, but generated store methods only take an id, and bigint ids make it easy to hit. Fix: resolve the cart first and query `WHERE id = $1 AND cart_id = $2`, answering 404 otherwise. Not in `docs/bogie-gaps.md`; a possible Bogie idea is scoped finders for `:references` (Rails' `current_cart.cart_items.find(id)`).
- **The server accepts any token the client sends.** A client can choose its own token (for example `visitor-123`, which the test uses). Low risk for a cart, but the token is not proof the server issued it.
- **Every tokenless request creates a cart row**, including the badge's `GET /cart` on every page for a new browser or a crawler. `GET` has a write side effect, and nothing cleans up old carts.
- **`randomToken` panics if `crypto/rand` fails**, caught by `middlewares.Recover`. Fine, and explained in the code comment.

### Rails comparisons

- `X-Cart-Token` plays the role of Rails' `session[:cart_id]`, and `currentToken` the role of Depot's `CurrentCart` concern with `set_cart`. Rails hides the session in a signed cookie. Here the client holds the token and sends it back.
- Rails would write `current_cart.line_items.find(params[:id])`, which scopes the lookup to the visitor's cart for free. The Go version has to say it in SQL, and forgetting it is the bug above. A good "Coming from Rails?" box.

### Chapter-worthy observations

- **The ownership bug is the best security lesson so far.** It appears exactly when carts stop being shared, and it ties back to the bigint-vs-uuid choice. Pass 2 can make it a deliberate "spot the bug" moment, then fix it with a scoped query and a test.
- **One pattern, two uses.** Using a header token now and token auth in Task D means the reader learns "send what you have, save what comes back" once. The commit's note that a leaked cart token only exposes a cart, not an account, is a good sidebar.
- **Guest cart to user cart.** Task D will need to merge a guest cart into a logged-in user's cart. Rails apps do this by hand too.

### Still to do

- [x] Fix the cart_items ownership bug (scoped queries for update and delete, plus tests with two carts) — done 2026-10-20, see below.
- [ ] Decide whether to only accept server-issued tokens.
- [ ] Clean up abandoned carts, or create a cart only on the first add, not on `GET /cart`.
- [ ] Plan the guest-cart merge for Task D.
- [ ] Task B (categories) and the price rules task are still ahead.

### Addendum, 2026-10-20: fixing the ownership bug

Revisited once Task D (accounts) came up as next: the same gap matters more
once a cart belongs to a real customer, not just a guest token. `Update` and
`Destroy` in `CartItemsController` now resolve the caller's cart from their
own bearer token first (`FindOrCreateCartByToken`), the same thing `Create`
already did, and pass that cart's id into the store. New store methods,
hand-written in `cart_items_extra.go`: `UpdateCartItemQuantity` gained a
`cartID` parameter and now looks the line up with a new query,
`GetCartItemForCart` (`WHERE id = $1 AND cart_id = $2`); `DeleteCartItemForCart`
replaces the generated, unscoped `DeleteCartItem` as what the controller
calls. Either one on a line that exists but belongs to someone else's cart
is `ErrNotFound`, same as a line that doesn't exist at all — a caller can't
tell the two apart, the property the bug was missing.

Verified at both layers: `app/controllers/cart_items_controller_test.go`
checks the controller resolves and forwards the caller's own cart id
(`TestCartItemsUpdateScopesToTheCallersCart`), and a new
`app/models/cart_items_extra_test.go`, run against real Postgres, creates
two carts and proves cart B can neither update nor delete cart A's line,
and that cart A still can. `bin/ci` green throughout, model package
coverage 31% → 47%.

---

## Pass 1 › Price rules: a pricing engine, STI the Go way

### What was done

1. **2026-10-20, `price_rule` model.** `bogie g model price_rule name:string kind:string properties:jsonb active:boolean priority:integer starts_at:datetime ends_at:datetime`, then hand-edited the migration before it was ever applied: `starts_at`/`ends_at` made nullable (an unscheduled rule has neither), `active` defaulted to `true` (matching the source article). The generated store and domain code had to be hand-corrected to match — `StartsAt`/`EndsAt` are `*time.Time` in `domain.PriceRule`, converted to/from `pgtype.Timestamptz` by two small helpers in the generated `price_rules.go`. First nullable column in this app.
2. **2026-10-20, the engine itself: `app/pricing`.** A new package, hand-written in full per CLAUDE.md ("price rule engine... hand-written, never generated"): `Rule` interface (`Apply(ctx) (detail, err)`), a `kind string → Decoder` registry, `PriceContext` (original/current cents, applied rules, metadata), `Calculate` (runs rules in order, Rack-middleware style), `Result` (final cents, total savings, savings percentage). Four rule kinds, one file each: `FixedDiscountRule`, `PercentageDiscountRule`, `BuyXGetYRule`, `MinimumPurchaseRule`, ported from "Building an E-commerce Pricing Rules Engine with Single Table Inheritance in Ruby on Rails" (the same author's article). Table-driven tests in `pricing_test.go` cover every rule, unknown-kind errors, validation failures, and that rule order changes the result. No DB needed for any of it — pure functions over a context.
3. **2026-10-20, store and seeds.** `price_rules_extra.sql`/`.go`: `GetPriceRuleByName`/`FindOrCreatePriceRuleByName` (same idiom as products) and `ListActivePriceRules` (active, and within `starts_at`/`ends_at` if set, ordered by priority). `db/seeds/price_rules.go` seeds the article's four example promotions, adapted to cents: 20% off, $50 off, buy-2-get-1, spend-$500-get-$75-off. Each seed's `properties` is built by marshaling one of `app/pricing`'s own structs, not a hand-written map, so a seed can never drift out of sync with what the engine actually decodes.

### Bogie commands used

- `bogie g model price_rule name:string kind:string properties:jsonb active:boolean priority:integer starts_at:datetime ends_at:datetime`.
- `bogie g seed price_rules`.
- `make sqlc` after editing the migration and after adding `price_rules_extra.sql`.

### Generated vs hand-written

- Generated: the migration (then hand-edited for nullability/default before ever being applied), `domain/price_rule.go` (then hand-edited for the `*time.Time` fields), `app/models/price_rules.go` (then hand-edited for the two timestamp-conversion helpers), the sqlc output, the seed stub.
- Hand-written, start to finish: all of `app/pricing` (interface, registry, four rule kinds, context, calculator, result, tests), `price_rules_extra.go`/`.sql`, and the seed data and its `priceRule` helper.

### Bogie gaps/bugs found

- **`AGENTS.md`'s id-scheme line is stale.** See [`docs/bogie-gaps.md`](bogie-gaps.md#agentsmds-id-scheme-line-doesnt-update-with-bogietoml) — found while checking supported column types for `g model`, unrelated to price rules themselves.
- **First nullable column in the app exposed that `emit_pointers_for_null_types: true` (in `db/sqlc.yaml`) doesn't apply to `pgx/v5`'s own wrapper types.** A nullable `timestamptz` still comes back as `pgtype.Timestamptz`, not `*time.Time`, regardless of that setting. Not a bug — the setting only affects types with no native pgx wrapper — but worth knowing before the next nullable column surprises someone.

### Rails comparisons

- STI (`type` column + Ruby subclassing) becomes `kind` column + a `map[string]Decoder` registry. Go has no open class table, so where Rails resolves the right class automatically on load, this app resolves the right struct explicitly in `pricing.Load`. Adding a new rule kind is still "add one file," the same promise the article makes for STI.
- `properties` JSONB + ActiveRecord's dynamic `attribute :properties, :json` accessors becomes a plain Go struct with `json` tags — `json.Unmarshal` instead of hand-written `properties['amount']` getters/setters.
- The Rails article's `Calculator`/`Result`/`PriceContext` POROs map almost one to one onto Go structs and a plain function; no class needed for either, since Go has no DSL to replace.
- `Rule.where(active: true).order(:priority, :created_at)` becomes `ListActivePriceRules`, one hand-written SQL query with the date window folded in, rather than an AR scope built at call time.

### Chapter-worthy observations

- **A deliberate simplification carried over, and flagged in code:** `BuyXGetYRule` records free items in metadata but, exactly like the source article, does not itself discount the price — doing that correctly needs a specific item's unit price, which a single cart-wide price doesn't carry. The real fix is applying pricing per cart line, not here. Worth turning into an explicit "spot the limitation" moment rather than quietly fixing it, since the source material has the same gap.
- **No admin endpoint, and that's fine for now.** Per the sequencing decision in project memory, rules are seeded, not created through the API — admin management waits for the Task F namespace chapter, same as product CRUD.
- **Wired into the cart, against the total (not per line).** `CartController.Index` and `CartItemsController`'s `respondWithCart` both call `ListActivePriceRules` then run `cartView`'s new pricing step against the cart's subtotal — the simpler of the two options noted above, and the one that matches the source article (a single scalar price through the pipeline). `BuyXGetYRule`'s price-reduction gap (noted above) is therefore still open: the cart integration doesn't pass a `quantity` through metadata yet, so that rule only ever records free items, never discounts. Per-line pricing, if it happens, is a bigger follow-up.
- **No coupon codes.** Discussed and deliberately deferred to Task E: a coupon is modeled as a `price_rule` with an optional `code` column (additive, no rework needed later), but validating a customer-typed code belongs next to checkout, which doesn't exist yet. For now, every active rule applies automatically to every cart — "enable/disable in the database and see what applies" is the whole mechanism, confirmed live: flipping `active` in Postgres directly changed the very next `GET /cart` response, no restart needed.
- **The wire contract changed and the UI changed with it**, per CLAUDE.md: `CartResponse` gained `subtotal_cents`, `discount_cents`, and `applied_rules` (`total_cents` now means what the customer actually pays, not the pre-discount sum). `public/cart.html` shows each applied rule as its own line above the total.

### Still to do

- [ ] Thread real metadata (cart quantity per product, at least) through `pricing.Calculate` so `BuyXGetYRule` has something to act on.
- [ ] Table-driven coverage for `starts_at`/`ends_at` scheduling and `priority` ordering against a live database (`ListActivePriceRules` itself is untested against Postgres, only unit-tested in `app/pricing`).
- [ ] Decide whether `BuyXGetYRule`'s price-reduction gap gets fixed now or stays a documented limitation until cart-line-level pricing exists.
- [ ] Coupon codes, when Task E starts: add `price_rules.code` (nullable), and a checkout-time lookup/validation.

---

## Pass 1 › Task D: Customers need accounts

### What was done

1. **2026-10-20, fixed a real Bogie gap first: `:uniq` duplicates.** Before adding `users.email:uniq`, checked what `g model` does with an existing `:uniq` column (`api_key.prefix`) on a collision: nothing — the raw Postgres `unique_violation` surfaced as an unclassified 500. Fixed in Bogie itself (`internal/generate/model.go`): `Create`/`Update` on any model with a `:uniq` column now catch error code `23505` and return `domain.ErrInvalid` with a readable message ("email is already taken"). Added a golden test (`model_user`) since no existing fixture exercised this path; `api_key`'s golden picked up the same fix. Released as Bogie v0.2.11, then `bogie app:update` here — one unrelated, pre-existing merge conflict in `routes_test.go` (kept the app's own `TestWelcomeInDevelopmentOnly`, same precedent as the v0.2.8 update).
2. **2026-10-20, `user` model.** `bogie g model user email:string:uniq password_digest:string`. Password hashing itself (`app/services/password`) is hand-written and deliberately *not* on `domain.User`: `app/domain` imports nothing third-party (AGENTS.md rule 3), so `bcrypt` lives in its own small service package instead, the same way Bogie's own `api_key` shape keeps bcrypt out of `domain.ApiKey`. `domain.User.Validate` (hand-filled, like every model's) checks the email looks like one and that a digest is present.
3. **2026-10-20, signup and login.** `UsersController` (`POST /users`) hashes, creates, and immediately mints a bearer token — no separate login needed right after registering. `SessionsController` (`POST /sessions`) checks the password (a wrong password and an unknown email answer identically: 401, same message, so a caller can't tell which), then mints a token for the real user id as subject — the exact same `token.Maker`/`RequireUser` machinery guest sessions already used, so nothing downstream (cart, cart_items) needed to change at all.
4. **2026-10-20, the guest-cart merge.** `SessionsController.Create` reads whatever bearer token the login request already carried (optional — a fresh browser with no prior cart sends none), verifies it, and if it names an existing cart, folds every line into the newly-logged-in user's cart (`AddCartItem`, same increment-or-create/stock-check logic cart_items already has) before deleting the old cart (`DeleteCartAndItems`, new: `cart_items` has no `ON DELETE CASCADE` yet, so its lines go first — a real transaction is Task E's job). Best-effort throughout: a missing/expired/garbage old token, or no cart under it, is not an error, just nothing to merge; a line that can't merge (out of stock, say) is logged and skipped rather than failing the whole login.

### Bogie commands used

- `bogie g model user email:string:uniq password_digest:string`.
- `bogie app:update` to v0.2.11 (self-released, see above).
- `make sqlc` for `users_extra.sql` (`GetUserByEmail`).

### Generated vs hand-written

- Generated: the migration, `domain.User`'s fields, the base store CRUD, sqlc output, the model test skeleton (then hand-fixed, see gaps below).
- Hand-written: `app/services/password` (hash/check), `domain.User.Validate`, `UsersController`, `SessionsController` including the whole merge, `users_extra.go` (`GetUserByEmail`), and new additions to `carts_extra.go` (`GetCartByToken` — find without creating, for the merge; `DeleteCartAndItems`).

### Bogie gaps/bugs found

- **The `:uniq` fix, above** — the main one, fixed upstream rather than worked around.
- **A model's generated not-found test breaks the moment `Validate` stops being a no-op.** `users_test.go`'s generated `TestUnknownUsersAreNotFound` calls `UpdateUser` with a zero-value `domain.User{ID: ...}` to prove a missing row is `ErrNotFound` — works for every model so far because every `Validate` has been an empty stub, but `User.Validate` now rejects the zero-value's blank email *before* the row lookup ever runs, so the test failed with `ErrInvalid` instead. Fixed by hand (a valid-looking-but-nonexistent user), the same kind of fix the file's own "TODO: a round trip" comment already invites — not a Bogie bug, since the generator can't know what a future hand-written `Validate` will require, but worth knowing before the next model needs real validation too.
- **The one cosmetic artifact from wiring price rules into the cart (Task before this one):** an empty cart still lists every active rule in `applied_rules` with `discount_cents: 0`, since `PercentageDiscountRule`/`FixedDiscountRule` "apply" even to a zero price. Harmless, slightly noisy; not fixed.

### Rails comparisons

- `password_digest:string` + `app/services/password` is `has_secure_password` + bcrypt, split into Go's two pieces: the column Rails' macro would add, and the hashing Rails' macro would generate for you.
- `POST /users` auto-signing-in after signup is Devise's default behavior; `POST /sessions` is literally Rails' own `sessions#create` naming, chosen on purpose back in Task C2 to pair with `/guest_sessions`.
- A wrong password and an unknown email both answering 401 with the same message is `authenticate`'s behavior in older Rails (newer Rails' `authenticate_by` also pads the timing with a dummy hash comparison; this app doesn't — noted, not fixed, a real but minor timing side-channel).
- The guest-cart merge is hand-written here the same way Rails apps do it by hand too — there's no gem or generator for it on either side.

### Chapter-worthy observations

- **"Same shape, different stakes" paid off exactly as planned back in Task C2.** Reusing the identical `token.Maker`/`RequireUser`/`Subject` machinery for a real login, with zero changes to `CartController`/`CartItemsController`, is a clean demonstration of why that design choice was made before accounts even existed.
- **The merge is best-effort on purpose**, and that's a good small lesson of its own: a login failing because of an unrelated stock conflict in a three-week-old guest cart would be a strange, hard-to-explain bug in production.
- **Finding and fixing the `:uniq` gap in Bogie itself, before using it**, is the clearest example yet of "fixing Bogie is part of the work" — a one-line column modifier that silently did nothing useful until this task needed it to.

### Still to do

- [ ] Pad login's timing with a dummy bcrypt comparison when the email doesn't exist, closing the minor timing side-channel.
- [x] Decide whether `DeleteCartAndItems` should become a real transaction once Task E introduces them — see Task E's notes below.
- [ ] Task E (checkout) is next: orders now belong to a real account.

## Pass 1 › Task E: Let customers place orders

### What was done

1. **2026-10-21, `order` and `order_item` models.** `bogie g model order user:references status:string subtotal_cents:integer discount_cents:integer total_cents:integer` and `bogie g model order_item order:references product:references quantity:integer unit_price_cents:integer`. `Order.Validate`/`OrderItem.Validate` hand-filled (status is one of `pending`/`paid`/`cancelled`; amounts and quantity can't be negative/zero) — same shape as every other model's.
2. **2026-10-21, the transaction helper.** `app/models/store.go` gained an unexported `(s *Store) withTx(ctx, fn func(*Queries) error) error`: begins a pgx transaction, runs `fn` against a `*Queries` rebound to it (sqlc's own `Queries.WithTx(tx)`, which has existed since `bogie new` but nothing above it had ever called), and commits on a nil return or rolls back otherwise. Checked a prior Go project (`line-connect`) for a remembered pattern first — it had the identical idea already, as a private `execTx(ctx, fn func(*Queries) error) error` plus one named method per composite write (`CreateXWithY`); this app's `withTx` and `CreateOrderFromCart` follow that same shape.
3. **2026-10-21, checkout itself.** `app/models/orders_extra.go`'s `CreateOrderFromCart(ctx, userID, cartID)`: reads the cart's lines, prices them through the same `pricing.Calculate` the cart view uses (so the order's total matches what the customer last saw), then — inside one `withTx` — conditionally decrements each line's stock (`UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1`, zero rows affected means not enough left), writes the order and its order_items with today's prices snapshotted in, and clears the cart. Any failure (a line out of stock, mid-way) rolls back everything: no partial stock decrement, no half-written order, cart untouched.
4. **2026-10-21, `DeleteCartAndItems` revisited.** Task D's guest-cart merge left a note that its two sequential deletes (`cart_items` first, no cascade, then the cart) should become a real transaction once Task E had one to offer. Done: it now runs both inside `withTx` too.
5. **2026-10-21, `OrdersController`.** `POST /orders` (checkout), `GET /orders` (history), `GET /orders/:id` (one receipt, scoped to its owner). All three sit behind `s.WithUser`, the same bearer-token gate `/cart` does — but a guest's token is a bare uuid subject with no `users` row behind it, so the controller parses the subject as a user id first and answers 403 ("create an account to place an order") if it isn't one, before ever calling the store.

### Bogie commands used

- `bogie g model order user:references status:string subtotal_cents:integer discount_cents:integer total_cents:integer`
- `bogie g model order_item order:references product:references quantity:integer unit_price_cents:integer`
- `make sqlc` for `products_extra.sql` (`DecrementProductStock`), `cart_items_extra.sql` (`DeleteCartItemsForCart`), `orders_extra.sql` (`ListOrdersForUser`, `GetOrderForUser`), `order_items_extra.sql` (`ListOrderItemsForOrder`).

### Generated vs hand-written

- Generated: both migrations, `domain.Order`/`domain.OrderItem`'s fields, the base store CRUD, sqlc output, the model test skeletons (then hand-fixed, see gaps below).
- Hand-written: `store.go`'s `withTx`, `orders_extra.go` (`CreateOrderFromCart`, `ListOrdersForUser`, `GetOrderForUser`), `domain.OrderLine` (a join type, same role `CartLine` plays), `Order.Validate`/`OrderItem.Validate`, `OrdersController`, and the `DeleteCartAndItems` rewrite.

### Bogie gaps/bugs found

- **No generated transaction helper** — see `docs/bogie-gaps.md`. `store.go` only wrapped `Open`/`Close`/`Ping` until this task; `withTx` is hand-written directly into it rather than fixed upstream, since this is the first task that has concretely needed one (the standing project rule: Bogie gets a capability once something needs it, not speculatively).
- **The generated not-found tests broke again, as `users_test.go`'s did in Task D**, for the identical reason: `UpdateOrder`/`UpdateOrderItem`'s generated test passes a zero-value struct to prove `ErrNotFound`, and a real `Validate` now rejects it before the lookup runs. Fixed the same way — a valid-looking-but-nonexistent row — and added the full round trips the generated files' own "TODO" comments invite.

### Rails comparisons

- `withTx` is Go's explicit version of `ActiveRecord::Base.transaction do ... end`: the boundary is the closure passed in, not "whichever AR calls happen between some implicit begin and end."
- The conditional stock `UPDATE ... WHERE stock >= $1` is the same trick a careful Rails app uses instead of `product.update!(stock: product.stock - qty)` (a read-then-write with no row-level guard) — `UPDATE ... SET stock = stock - ? WHERE stock >= ?` plus checking `update_all`'s affected-row count, not an ActiveRecord validation.
- Saving `unit_price_cents` on `order_items` rather than joining to `products.price_cents` live is the same reason a Rails invoicing app snapshots a `LineItem#unit_price` instead of delegating to `product.price` — a receipt has to stay true after the product's price changes.

### Chapter-worthy observations

- **The transaction boundary is the whole lesson.** Everything else in this task (the models, the controller, the routes) is routine by now; `withTx` and the one `CreateOrderFromCart` method that uses it are the only genuinely new idea, which makes this a good, tight chapter: one new concept, demonstrated once, proven by a test that actually breaks it (insufficient stock mid-checkout) and shows the rollback holding.
- **A prior, unrelated Go project already had the answer.** Checking `line-connect` for "have I done this before" instead of designing from scratch found the exact same shape (`execTx` + one method per composite write) — a good demonstration that this pattern isn't bespoke to this app, it's just what a sqlc-based Go backend's transactions look like.
- **Closing the `DeleteCartAndItems` note from Task D** rather than leaving it as a permanent two-step delete is a small example of a "Still to do" actually getting done once the tool to do it right exists, instead of quietly becoming permanent.

### Still to do

- [ ] Payment capture (`status` transitioning `pending` → `paid`/`cancelled` via a webhook) is out of scope here on purpose — `payment-simulator/` doesn't exist yet, and CLAUDE.md already calls out webhook handling as its own hand-written teaching material, later.
- [ ] `BuyXGetYRule`'s known limitation (metadata only, no actual per-line discount) is unchanged; checkout inherits it from the cart's own pricing calculation rather than introducing a new one.
- [ ] No git tag exists yet for price rules, Task D, or Task E — only `task-c1`/`task-c2` exist so far. Still an open offer, not yet acted on.

## Pass 1 › Task F (1/3): roles, the admin namespace, and product CRUD

Task B (categories) was never built — skipped from Task A straight to C on
2026-10-18 — and gets folded into this task instead of a separate pass:
decided 2026-10-22, since an admin area is exactly where category
management belongs anyway. This note covers the first of three steps:
roles and staff-only product management. Categories and price-rule admin
are still ahead.

### What was done

1. **2026-10-22, `users.role`.** `bogie g migration add_role_to_users role:string`, default `'customer'`. `g migration` only touches the migration and reruns sqlc, as documented (AGENTS.md's "Change a table") — threading the new column through `domain.User`, `UpdateUser`/`CreateUser`'s SQL (hand-added `role` to both in `db/queries/users.sql`), and `userFromRow` was by hand, the same kind of follow-up the nullable `starts_at`/`ends_at` columns needed back in the price rules task. `domain.User.Role` is checked against `UserRoles = []string{"customer", "staff"}` in `Validate`, the same enum-as-a-string-check `Order.Status` already uses. `UsersController.Create` always sets `Role: domain.RoleCustomer` itself, never from the request body — the only way into the staff role is `FindOrCreateUserByEmail` (new, mirrors `FindOrCreatePriceRuleByName`'s idiom) called from a seed.
2. **2026-10-22, `RequireStaff`.** A new middleware, hand-written beside the generated `RequireUser` (`app/middlewares/require_staff.go`): looks up the caller's role fresh from the store on every request — never trusted from the bearer token's claims, since a demotion has to take effect on the very next request. A guest's token, an unknown subject, and a real customer who isn't staff all answer identically, 403 "staff access required" — the caller is never told which.
3. **2026-10-22, the admin namespace.** `bogie g controller admin/products index show create update destroy` — Bogie's namespace generator, used for the first time in this app: writes `app/controllers/admin_controller/` (its own `Server`, `routes.go`, one file per controller) and wires it into the root `Server` and `app/application.go`, exactly as documented. Hand-added: `staffStore`/`staffTokens` interfaces and a `WithStaff` method on the namespace's own `Server` (`RequireUser` then `RequireStaff`, chained once, applied to the whole `/admin` group in `routes.go` — "the exposure boundary is visible in one place," per AGENTS.md's own description of what a namespace's shared middleware is for), and the wire line in `app/application.go` passing the shared `*models.Store` and `server.Tokens()` in — staff log in through the exact same `/sessions` a customer does, no parallel login system.
4. **2026-10-22, admin product CRUD.** `admin_controller.ProductsController` gets the full five actions (the public one still only has Index/Show); `Product.Validate` went from a no-op to real rules (name required, price and stock non-negative) — closing a gap flagged since Task A's own "Still to do". `app/views/products.go` gained `ProductRequest` for the create/update body.
5. **2026-10-22, one seed, no promotion path yet.** `bogie g seed staff_user`, filled in: hashes a fixed development password and creates `staff@shop.test` with `Role: staff`, idempotent via `FindOrCreateUserByEmail`. There is no admin UI or endpoint yet to promote a second person — see "Still to do".

### Bogie commands used

- `bogie g migration add_role_to_users role:string`
- `bogie g controller admin/products index show create update destroy` — the namespace generator, first use in this app.
- `bogie g seed staff_user`

### Generated vs hand-written

- Generated: the migration, the admin namespace's package skeleton (`server.go`, `routes.go`, `products_controller.go` stubs), the seed stub, every registration line (root `Server`, `app/application.go`, `routes.go`, the namespace's own `server.go`/`routes.go`).
- Hand-written: `role`'s threading through `domain.User`/`users.go`/`users.sql` (as `g migration` always leaves to the reader), `RequireStaff`, the namespace's `staffStore`/`staffTokens`/`WithStaff`, every admin product action's real logic, `Product.Validate`, `ProductRequest`, the seed's body.

### Bogie gaps/bugs found

- None this time — the namespace generator (`g controller admin/products ...`) worked exactly as documented on the first try, markers and all, confirmed by `bogie doctor` ("namespace admin registered", "controller admin/products registered").
- The same generated-not-found-test-breaks-when-Validate-stops-being-a-no-op pattern hit `products_test.go` (same as `users_test.go` in Task D, `orders_test.go`/`order_items_test.go` in Task E) once `Product.Validate` became real. Fixed the same way, with a full round trip added too.

### Rails comparisons

- `role:string` + an app-level enum check is `enum :role, [:customer, :staff]` without ActiveRecord's enum macro — Rails would store an integer and map it; this app already made that exact "borrow the idea, write the Go way" call for `Order.Status`, so `User.Role` just repeats it.
- Staff signing in through the same `sessions#create` a customer uses, with a `before_action` doing the role check afterward, is exactly how a Rails app layered with Pundit or CanCanCan separates authentication (`authenticate_user!`) from authorization (`authorize @product`) — this app's split is `RequireUser` / `RequireStaff`, same two concerns, same order.
- `namespace :admin do resources :products end` with `before_action :require_staff!` on the namespace's own controller is the Rails line Bogie's `g controller admin/products` plus `WithStaff` reproduces.

### Chapter-worthy observations

- **Authentication and authorization are two different middlewares, not one.** `RequireUser` only answers "is this a valid, unexpired token" — cryptographic, no database. `RequireStaff` answers "is this particular person allowed here" — a live lookup, and deliberately never cached in the token, so revoking access is instant. Worth a clear diagram: the C2 guest/user token design answered the first question once and reused it for everyone (`same shape, different stakes`); this task adds the second question only for the one group that needs it.
- **Live-checking the role instead of trusting a claim is the whole point of a staff gate**, and worth calling out explicitly as a place a tutorial could cut a corner (bake `role` into the JWT at login, skip the DB hit) that would look fine until someone's access needs to be pulled mid-session.
- **The namespace generator needed nothing fixed**, a quiet contrast with Price Rules, Task D, and Task E, each of which found something. Worth noting in the book as the moment the generator finally meets a shape it was built for exactly as built.

### Still to do

- [ ] No way to promote a second staff account, or change the seeded password, short of editing the database by hand. Likely answer: an admin "manage users" endpoint, itself staff-only — not built yet.
- [ ] Categories (folded in from Task B) and price-rule admin CRUD are still ahead — this note is step 1 of 3.
- [ ] The public `GET /products`/`GET /products/:id` are unchanged; only `/admin/products` gained write actions.

## Pass 1 › Checkout UI (between Task F steps)

Added 2026-10-22, before Task F's categories step, because Tasks D and E had been built backend-only and the UI never caught up.

### What was done

1. **`GET /me`** (`MeController`, behind `s.WithUser`; `bogie g controller me index`, the same singular-resource stand-in the cart uses). Returns `{id, email, role}` for an account, 403 for a guest token. Needed because a guest's bearer token and a real login's are both valid and opaque to the UI; the payment page asks the server instead of decoding the JWT itself.
2. **Checkout button** on `cart.html`, linking to `payment.html`.
3. **`payment.html`**: asks `/me`; 401/403 sends the visitor to `login.html?next=/payment.html`. Signed in, it shows the cart (lines, applied rules, total) and a "Place order" button that calls `POST /orders`, then goes to the receipt. No card is charged: payments are a later chapter, and the page says so.
4. **`login.html`**: one form with a log in / sign up toggle, then back to `next` (a same-site path only; `//host` and absolute URLs fall back to `/`). Signing up always continues into `POST /sessions` carrying the guest token in `Authorization`, because only `SessionsController` merges the guest cart; `POST /users` does not.
5. **`order.html?id=N`**: the receipt, from `GET /orders/:id`.

Verified in real Chrome (puppeteer-core, scratch script, not committed): guest cart → Checkout → login → sign up → payment shows the merged cart → Place order → receipt.

### Still to do

- [ ] `POST /users` could merge the guest cart itself instead of the UI chaining a login after signup.
- [ ] The headless-browser check is not part of `bin/ci`.

### Addendum, 2026-10-22: header account menu and order history

- `public/auth.js` (new; `go:embed` widened to `*.html *.js images`) holds `account()`, an Alpine component every page's header uses: asks `GET /me` who the saved token is for, shows "Log in" (with `next` set to the current page) or "Orders / email / Log out". Log out just forgets the saved token; the next visit mints a guest. A plain global function rather than `Alpine.data(...)`; same effect.
- `orders.html`: the history, from `GET /orders`, each row linking to `order.html?id=N`; a guest is sent to login first. `GET /orders` rows gained `created_at` (additive).
- Checked in real Chrome again: guest header, orders-as-guest redirect, sign up from there, history empty then one row, row opens the receipt, log out.

### Addendum, 2026-10-22: the admin landing page

- `public/admin/index.html`: the first admin page. It asks `/me`; no login goes to `login.html?next=/admin/`, a signed-in non-staff account goes to `/`, staff see a small dashboard (Products with a live count from `GET /admin/products`; Categories and Price rules marked "coming soon"). The redirect is a convenience, not the security: every `/admin` API call is checked again by `RequireStaff`.
- `staticFile` now serves a folder's `index.html` (`/admin/`, with a 301 from `/admin` so relative links work); a folder without one is still 404. The embed widened to `*.html *.js admin images`.
- The header shows an "Admin" link to staff only (`/me` already returned `role`; `account()` gained `isStaff`).
- Checked in real Chrome for all three outcomes: no login, a customer, and staff.
- Categories wait for the admin pages, as agreed; next is the admin products page.
