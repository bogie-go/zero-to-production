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
- Arc: early tasks mostly generated code; middle tasks hand-written around it; late tasks designed independently (price rules, payment webhooks).
- Theme: borrow the idea from Rails, write it the Go way.

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

Next in the plan: H (deploy) and I (health). `docs/build-notes.md` has the open items.

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
| Webhooks | late, duplicate and out-of-order events from the payment simulator leave one correct result | payment simulator |

Rules for tests in the book:
- Every task adds the test with the code, in the same step, never as a later "testing pass".
- The test database is separate and is configured only through the environment (`ECOMMERCE_TEST_DATABASE_URL`), so a test can never touch development data.
- Hand-written teaching code (price rules, cart, webhooks) gets the most tests, including the edge cases a reader would not think of (time windows, stacking, duplicates).
- Bugs found later come with a test that fails first (the category select, the cart merge, the soft delete are the examples).

Gaps to close in Pass 2:
- The browser checks used in Pass 1 were ad hoc puppeteer scripts. Turn them into `bin/e2e`, kept out of `bin/ci` so readers need no Node for the core loop.
- Request tests now cover every endpoint (`test/requests`: catalog, shopping, accounts, cart, admin). Still to add: concurrency (two buyers, one unit of stock) and a failing-first test with every future bug.
- A concurrency test for checkout and for `carts.token`.
- Decide whether `bogie g model` should generate tests that survive a real `Validate` (see `docs/bogie-gaps.md`).

## Supporting pieces
- UI: ready-made, acts as the API contract. Features unlock as endpoints appear; shows clear errors when a response doesn't match the contract; one setting for the base URL. Served by Bogie from the same origin (no CORS).
- Payments: simulated provider that sends webhooks (teaches idempotency and handling late/duplicate events).
- Collaboration cast: shop owner (changing requirements), frontend developer (contract), payment provider (external system), senior reviewer (code review notes). "Working with others" boxes.

## Build process
1. Pass 1: build the complete shop quickly to settle the data model, contract, and Bogie gaps.
2. Freeze the API contract; build the UI and payment simulator against it.
3. Pass 2: rebuild from an empty folder, iteration by iteration, exactly as a reader would. Commit and tag each iteration; keep notes as chapter raw material. Script it where possible so checkpoints can be regenerated when Bogie changes.
4. Provide each checkpoint as a Git tag and a zip.

## Still open
- The shop's identity (name, owner, what it sells).
- Book language (Thai, English, or both).
- Whether Bogie generates an OpenAPI spec.
