# Book plan

## Promise to the reader
By the end of this book, you can build a Go backend with Bogie, design it well enough that it survives changing business rules, work with the people around it, and run it in production.

## Readers
- Main: people who can code a little in any language but have never shipped a web app.
- Secondary: Rails developers. They get "Coming from Rails?" boxes mapping Rails concepts to Go.
- Prerequisites (not taught): basic programming, terminal, Git basics.
- Taught lightly, just in time: SQL.
- Not taught: frontend development, Git.

## Approach
- Pragmatic Bookshelf style: one project built in named tasks and iterations, each driven by a client request.
- Bogie first. Each iteration follows: generate, run, read the generated code, change it by hand, optionally peek under the hood.
- A plain `net/http` server is shown once early, as the foundation under everything.
- Arc: early tasks mostly generated code; middle tasks hand-written around it; late tasks designed independently (price rules).
- Theme: borrow the idea from Rails, write it the Go way.

## The shop: Bogie Depot
The shop in the book is **Bogie Depot** (name decided 2026-10-25). It is not a made-up example: https://depot.bogie-go.com is the live store and **will sell real Bogie merchandise**, so the book's reference app and a working shop are the same code. The owner character (who runs Bogie Depot in the story, and how they talk to the reader) is still open.

- **The catalog is merchandise with no variants:** stickers, a mug, a desk mat, a keycap, a poster and a notebook, one price and one stock count each. The images are placeholders until there are real photos.
- **Out of the book, in the live store later:** real payments, shipping, tax and product variants (T-shirt sizes and the like). The book stops where its scope stops; these are added to the live store afterwards and appear in "Where to go next" as the next steps a real shop takes. Until then the live store takes no real orders.

## The project: an e-commerce backend (API only)
| Task | Client request | Teaches |
|---|---|---|
| A | Show our products | First model, CRUD, JSON responses, pagination |
| B | Group products into categories | Relationships, filtering, JOIN |
| C | Let customers add to a cart | State, validation, guest vs logged-in users |
| Price rules | Run a sale / buy two get one | STI-style rules: interface, type column, registry, JSON params, table-driven tests |
| D | Customers need accounts | Token authentication, password handling |
| E | Let customers place orders | Transactions, conditional stock UPDATE, saving price at time of order |
| F | Only staff can manage products | Roles and authorization |
| G | Email a confirmation | Background jobs, mailer |
| H | Let's go live | Deployment, config, secrets |
| I | Is it healthy? | Logging, monitoring, safe updates |

Testing runs through every task (see "Testing" below). Out of scope (sketched in "Where to go next"): real payments, shipping, tax, reviews, image uploads, multiple vendors, product variants (size/color, per-variant price and stock — considered for Task B, deferred: it would mean reworking the cart and seeds already built around a bare `product_id`, not just adding something new).

## Pass 1 status and checkpoints (2026-10-22)

Pass 1 is built through Task G. Task B (categories) was skipped at first and built inside Task F, where the admin pages it needs live. Each checkpoint is a git tag on the commit that finished that step. They are Pass 1's, so provisional: Pass 2 rebuilds from an empty folder and re-cuts them the way a reader would.

| Task | Tag | What the checkpoint is |
|---|---|---|
| A | `task-a1` | products: model, seeds' API, index and show |
| A | `task-a2` | the catalog front page, seeds with real photos |
| A | `task-a3` | the product detail page (Task A complete) |
| C | `task-c1` | cart mechanics and UI |
| C | `task-c2` | personal carts (Task C complete) |
| Price rules | `task-pricing` | the rule engine, wired into the cart |
| D | `task-d` | accounts: signup, login, merging a guest's cart |
| E | `task-e1` | checkout: a real transaction, stock, price snapshot |
| E | `task-e2` | the checkout UI, receipts, order history |
| F | `task-f1` | roles, the admin namespace, product CRUD API |
| F | `task-f2` | the admin UI and soft delete (archive) |
| F | `task-f3` | categories: relationship, JOIN, filtering |
| F | `task-f4` | price rules in the admin (Task F complete) |
| G | `task-g1` | background jobs: the order confirmation |
| G | `task-g2` | the welcome email |
| G | `task-g3` | the sign-in alert (Task G complete) |

H (deploy) is done: the shop runs at https://depot.bogie-go.com. The OpenAPI contract is drafted and enforced (`docs/openapi.yaml`, `docs/api-contract.md`, 2026-10-25); Bogie's generator entries for it are not built yet. Next: Postgres backups, then Pass 2. I (health) is built (2026-10-24, see `docs/operations.md`). `docs/build-notes.md` has the open items.

## Testing

Testing is taught in the tasks, not in a chapter of its own: each task ends green on `bin/ci`, and the new kind of test arrives with the task that needs it.

**Outside in.** A feature starts from a request test: the real router, the real database, the factories for the arrangement, and assertions on status and JSON, which is the contract the UI relies on. The layers below it (store, domain, services) get the edge cases the request test would make tedious, added afterwards. There is no controller test with a fake store: the request test goes through the whole `Server`, so the wiring is tested too, and an unexpected failure is provoked for real (the store closed). Tests are written with Ginkgo and Gomega, RSpec-style, in external `_test` packages.

Shared helpers live in `test/`, as Rails' `test/` does: `test/testdb` (a clean test database), `test/factories` (hand-written, no library: valid defaults, typed overrides, saved through the real store), `test/fake` (outside systems), `test/fixtures` (fixed files).

| Layer | What it checks | Where it is taught |
|---|---|---|
| Unit, table-driven | pure logic with no I/O: validation, the price rules, pagination parsing, password and token helpers | A (validation), Price rules, D |
| Store / model | real SQL against a real Postgres test database, one clean state per test: ordering, limit and offset, counts, constraints, FK and unique errors | A, B, E |
| Transaction | all-or-nothing: a failing line leaves stock, order and cart untouched; concurrent buyers never oversell | E |
| Controller (HTTP) | `httptest` against the router with a fake store declared next to the controller (rule 2): statuses, JSON shapes, 401/403/422 | every task that adds an endpoint |
| Jobs and mail | a worker run directly with a fake mailer; the enqueue happens inside the transaction and rolls back with it | G |
| Contract | the JSON the UI relies on does not change by accident (golden or shape checks), since the UI is the contract | A onward, before the UI freeze |
| End to end | a real browser against a running server: browse, add to cart, sign up, check out, admin edit | Pass 2, `bin/e2e` |
| Fake outside server | an `httptest.NewServer` standing in for a third-party API, so the real client code runs against a fake that can fail on demand | Task G (the mailer's fake SMTP server), described for HTTP APIs |

Rules for tests in the book:
- Every task adds the test with the code, in the same step, never as a later "testing pass".
- The test database is separate and is configured only through the environment (`ECOMMERCE_TEST_DATABASE_URL`), so a test can never touch development data.
- Hand-written teaching code (price rules, cart) gets the most tests, including the edge cases a reader would not think of (time windows, stacking, duplicates).
- Bugs found later come with a test that fails first (the category select, the cart merge, the soft delete are the examples).

Gaps to close in Pass 2:
- The browser checks used in Pass 1 were ad hoc puppeteer scripts. Turn them into `bin/e2e`, kept out of `bin/ci` so readers need no Node for the core loop.
- Request tests cover every endpoint, one file beside each controller (`app/controllers/*_controller_test.go`), with the cross-controller journey, wiring and failure specs in `test/requests`. Still to add: concurrency (two buyers, one unit of stock) and a failing-first test with every future bug.
- A concurrency test for checkout and for `carts.token`.
- Decide whether `bogie g model` should generate tests that survive a real `Validate` (see `docs/bogie-gaps.md`).

## Supporting pieces
- UI: ready-made, acts as the API contract. Features unlock as endpoints appear; shows clear errors when a response doesn't match the contract; one setting for the base URL. Served by Bogie from the same origin (no CORS).
- No payment simulator (decided 2026-10-24). Its only purpose in the book was to show a fake server in tests, and the shop already has one: the mailer's test starts a fake SMTP server (`app/services/mailer`), and the testing chapter shows the same idea for an HTTP API with `httptest.NewServer`. Real payments, webhooks and their idempotency stay in "Where to go next".
- Collaboration cast: shop owner (changing requirements), frontend developer (contract), mail provider (external system), senior reviewer (code review notes). "Working with others" boxes.

## The API contract and OpenAPI

The UI is the contract today: its fetch calls and the `views` types decide what the API says. At the contract freeze (step 2 of the build process) the contract gets written down as an OpenAPI file and enforced by the tests, so the book can say where the contract lives and show it failing when it drifts.

- **One spec file in the app,** `docs/openapi.yaml`, hand-editable: the descriptions and examples are written by people.
- **Tests keep it honest.** `testapp` validates every request and response in the request tests against the spec with `kin-openapi` (a test-only dependency). Code that drifts from the spec, or a spec that drifts from the code, fails a request test. This is what makes the UI-is-the-contract claim checkable, and what no code-comment generator (swaggo-style annotations) gives, since comments cannot be validated.
- **A docs page,** buildless like the rest of `public/`: `public/docs.html` loads Swagger UI or Redoc from a CDN and shows `/docs`.
- **When:** after Task I, as one task: draft the spec for the shop from the routes and `views` types, add the validation to `testapp`, fix every mismatch it finds, then freeze.
- **Bogie follows the shop,** as it did for tests: once a proven spec and validator exist, `g scaffold` and `g controller` insert their paths and schemas above marker comments in `docs/openapi.yaml` (schemas from the model's attributes and the `views` structs), and the validation ships in the fresh-app template. Decided 2026-10-24; not built yet.

## Build process
1. Pass 1: build the complete shop quickly to settle the data model, contract, and Bogie gaps.
2. Freeze the API contract (written as OpenAPI and enforced by the request tests; see above); build the UI against it.
3. Pass 2: rebuild from an empty folder, iteration by iteration, exactly as a reader would. Commit and tag each iteration; keep notes as chapter raw material. Script it where possible so checkpoints can be regenerated when Bogie changes.
4. Provide each checkpoint as a Git tag and a zip.

## Still open
- The shop's identity: ~~name~~ Bogie Depot and what it sells (Bogie merchandise) are decided (2026-10-25); the **owner character** is still open.
- ~~Book language.~~ English (decided 2026-10-25).
- ~~Whether Bogie generates an OpenAPI spec.~~ Yes: generators insert entries into a hand-editable `docs/openapi.yaml`, and the request tests validate against it (see "The API contract and OpenAPI").
