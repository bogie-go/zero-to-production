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

Testing runs through every task. Out of scope (sketched in "Where to go next"): real payments, shipping, tax, reviews, image uploads, multiple vendors, product variants (size/color, per-variant price and stock — considered for Task B, deferred: it would mean reworking the cart and seeds already built around a bare `product_id`, not just adding something new).

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
