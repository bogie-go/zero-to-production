# Companion repo for "Bogie: How to Go from Zero to Production"

This repo holds the reference e-commerce shop built in the book, plus everything readers need alongside it. The full plan and the reasoning behind it are in @docs/book-plan.md. Read it before making structural changes.

## Repo layout
- `app/` — the reference shop, built with Bogie (Go, API-only, sqlc). Tagged at each book iteration (`task-a1`, `task-a2`, ...).
- `ui/` — the frontend source (Svelte) plus the built static files Bogie serves via `embed`. Readers never build this; they only use the built output.
- `payment-simulator/` — a fake payment provider that sends webhooks to the app, including late and duplicate ones.
- `requests/` — `.http` files, one per chapter.

## Rules for working in this repo
- Write the app the way a reader would: use Bogie's generators first, then hand-edit. Don't add libraries Bogie doesn't already bring unless the plan calls for it.
- Keep these hand-written, never generated: the price rule engine, cart logic, and webhook handling. They are teaching material.
- If Bogie is missing something the app needs, note it in `docs/bogie-gaps.md` rather than working around it silently. Fixing Bogie is part of the work.
- The UI is the API contract. Don't change endpoint paths or JSON shapes without updating the contract and the UI together.
- Keep the UI plain and buildless for readers: committed static output, no Node required to run the app.
- SQL stays simple and readable; readers only get a light introduction to SQL.
- Prefer idiomatic Go over Rails imitation. Borrow Rails ideas, write them the Go way (for example, STI-style price rules use an interface, a type column, and a registry).
