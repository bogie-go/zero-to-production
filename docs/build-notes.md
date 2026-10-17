# Build notes

Raw material for the book "Bogie, How to Go from zero to production": what was actually done while building the e-commerce API, which Bogie commands were run, what was generated and what was written by hand, and where Bogie fell short. These are notes, not chapters.

Last updated: 2026-10-17. Times are Bangkok time (UTC+7). "(inferred)" marks things read from files, timestamps or the Bogie repo, not confirmed by Mac.

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
10. **2026-10-17 19:36, in progress (uncommitted): upgrade to Bogie v0.2.7 (static files).** Run 8 seconds after `42dca45` (inferred: `bogie app:update`). New: `public/404.html`, `public/500.html`, `public/public.go` (embeds `*.html` only), and `app/controllers/public.go`. In `routes.go`, `s.mountWelcome()` becomes `r.NoRoute(s.staticFile, s.welcome, s.notFound)`. Also changed: `welcome.go`, `routes_test.go`, `middlewares.go` (Recover serves `500.html`), AGENTS.md, and `bogie.toml` (v0.2.7). No products UI page yet.

### Bogie commands used

- `bogie new ecommerce`, with `jobs = false`. Options are inferred from `bogie.toml`.
- `bogie g model product name:string description:text price_cents:integer stock:integer image_url:string` (inferred from the migration). Bogie v0.2.4, tagged 20:01 that evening, allows bare `name` to default to string, so the exact spelling may differ. `g model` runs sqlc itself.
- `bogie g controller products index show` (inferred: only those two routes exist, and the stubs match the generator's 501 text).
- `bogie app:update` to v0.2.5, then v0.2.6, then v0.2.7 (inferred). The commit touched exactly the files the templates own, and `bogie.toml` says "Do not edit by hand".
- `bogie g seed product` (inferred). The doc comment "SeedProduct seeds product. db seed can run this any number of times" comes straight from Bogie's `internal/generate/seed.go` template.
- `make sqlc` (or `go tool sqlc generate`) after adding `GetProductByName`. `products.sql.go` was rewritten 3 seconds after `products.sql` was saved (inferred).
- Probably `bogie routes`, `bogie server`, `bogie db:prepare` / `db:seed` and `bogie test` to check the work. No direct evidence for these.
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

### Bogie gaps/bugs found

- **Seeds were redesigned during the build.** The v0.2.3 seeds were goose SQL files with a version table and `seed-down`. Rails has neither, and SQL seeds can't call app code such as `FindOrCreate`. Bogie commit `e27c65d` (2026-10-17 09:33, released as v0.2.6 at 10:28) made seeds plain Go, run every time. Going from v0.2.5 to v0.2.6 means `app:update` rewrites `migrate.go`, `cli.go`, `db/db.go`, the Makefile and the test helper. That is a big upgrade for a few-days-old app.
- **v0.2.5's `main.go` change is a feature, not a hand fix.** Bogie added `bogie routes` (`3f1c39e`, 21:10 on 10-16), a thin proxy to `go run . routes`. That needs a `runRoutes` function in the app's own `main.go`, so `app:update` merged one in. The upgrade was committed 4 minutes after that Bogie release, which suggests the gap was found and fixed while building Task A (inferred).
- **`string` used to become `text`.** Bogie v0.2.4 (`946f810`, 20:01 on 10-16, an hour before `g model`) fixed this: `string` now becomes `varchar` and a bare field name defaults to string. That is why `name` and `image_url` are varchar and `description` is text (inferred: found while preparing this model).
- **Hand-written store methods went into a generated file.** `GetProductByName` and `FindOrCreateProductByName` were committed in `42dca45` inside `app/models/products.go`, which `g model` rewrites whole. Running `g model` again would report a conflict. Bogie v0.2.7 (`ec8b3ea`, tagged today at 19:34) makes the convention a sibling file, `app/models/products_extra.go`. `app:update` doesn't move them, so this is still to do.
- **The model test's TODO points to a file that doesn't exist.** `products_test.go` says "see posts_test.go for the shape". That text is hard-coded in Bogie's model template (see `model_comment.golden`), but a real app has no `posts_test.go`. The comment in `app/application.go` also uses `posts.New(store)`, a service-based example, while AGENTS.md says scaffolds have no service layer. Both are small template bugs.
- **Leftover generated comments after hand-editing.** `Index` and `Show` still say "calls one service function", but they call the store directly.
- **`ImageUrl`, not `ImageURL`.** sqlc's default naming breaks Go's initialism convention, and the name spreads into the domain and view structs. The JSON tag (`image_url`) is fine. An sqlc `rename` setting or a Bogie default could fix it.
- **`GetProductByName` is `:one` on a column that isn't unique.** With duplicate names it returns whichever row Postgres finds first. The code comment admits that find-then-create can race. A `name:uniq` index, or `ON CONFLICT`, would make this sound.
- **No pagination yet.** The list stops at 100 (`productsListLimit`) and the response gives no count or cursor. `ListProducts` takes only a `limit`.
- **No static file support until v0.2.7.** v0.2.7 (`7e45718`, today; being applied, uncommitted) adds a `public/` folder embedded in the binary, served by a three-handler NoRoute chain, plus 404 and 500 pages. That commit also found a Gin bug pattern: a NoRoute handler must call `c.Abort()`, or the next handler appends its output too. This app needs v0.2.7 for the Alpine.js UI. Still a gap: `public/public.go` embeds only `*.html`, so the planned CSS, JS and Alpine.js files won't be served until the `go:embed` pattern is widened (by hand, or by a later Bogie). Small note: `mustRead` panics at package init, which bends rule 4 ("Only main exits").
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
- Coming in v0.2.7: `public/`, `404.html` and `500.html` like Rails' `public/` and `PublicExceptions`, but embedded in the binary.

### Chapter-worthy observations

- **The upgrade story is a chapter moment.** Twice in one day the framework changed under the app (v0.2.5, then v0.2.6), and `app:update` carried the app across. Readers will meet this in real life. Showing a `bogie.toml` version bump plus a merge is honest and useful.
- **Seeds show "Borrow the idea from Rails, write it the Go way" well.** The first version followed the Go habit of reusing the migration tool. The redesign went back to Rails' idea: run every time and stay safe to repeat. The DESIGN.md reasoning ("no mature Go seeding tool", "a version table has no path to a seed that calls app code") is good sidebar material.
- **Generate, run, read, hand-edit worked as the plan says.** `g controller` gave a 501 stub. Filling it in taught interfaces declared where they are used (rule 2), fakes in tests, wire structs and error mapping. A natural iteration split: A1 model, A2 controller stub plus `bogie routes`, A3 real index/show, A4 seeds.
- **Decide how the reader writes index/show.** If the book says "type this", the code shouldn't secretly be scaffold output. One option: show `g scaffold` later as a "now you know what it writes" moment.
- **Money as `price_cents integer`** is a good early design note for the price rules and orders tasks. Price at the time of order will matter in Task E.
- **The 500 test** ("connection refused to db-1" must not leak into the body) is a short, clear security lesson.
- **The UI decision moved from Svelte to Alpine.js in `public/`** to keep the "no Node" promise. That probably set up the static file work in Bogie v0.2.7.
- **The `c.Abort()` gotcha** from Bogie's v0.2.7 commit could be a "peek under the hood" box when static files arrive.

### Still to do

- [x] Commit the v0.2.6 upgrade and the product seeds (`42dca45`).
- [ ] Move `GetProductByName` / `FindOrCreateProductByName` from `app/models/products.go` into `app/models/products_extra.go` (the v0.2.7 convention).
- [ ] Commit the v0.2.7 upgrade (in progress).
- [ ] Create, update and delete actions (POST, PATCH, DELETE `/products`) with `Validate` rules (name required, price ≥ 0, stock ≥ 0) and 422 responses.
- [ ] Fill in the model round-trip test (the TODO in `products_test.go`).
- [ ] Pagination for `GET /products`: decide offset or cursor, and the response shape (count, next).
- [ ] A basic HTML page listing products: `public/` plus Alpine.js, served from the same origin.
- [ ] Static file support in Bogie. v0.2.7 brings the basics (in progress). Still needed: serving CSS, JS and images (widen `go:embed`) and deciding where `index.html` lives versus the welcome page at `/`.
- [ ] Start `docs/bogie-gaps.md`, which CLAUDE.md asks for, and move the gap list above into it.
- [ ] Tag checkpoints (`task-a1`, ...) as CLAUDE.md describes. There are no tags yet.
