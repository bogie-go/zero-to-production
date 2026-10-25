# Operating the shop

What to look at, and what to do, once it is deployed. The setting names are
in `config/deploy.yml`; the reasons are in `docs/build-notes.md` (Task I).

## Is it healthy?

| Question | Where | Healthy |
|---|---|---|
| Is the process alive? | `GET /healthz` | 200 `ok`. Touches nothing, so it stays 200 when the database is down. |
| Can it do its job? | `GET /readyz` | 200 with `{"status":"ok","checks":{"database":"ok"}}`; 503 when the database does not answer. The body never says why. |
| What is running, and what is the queue doing? | `/admin/ops.html` (staff), or `GET /admin/ops` | the release (git sha), the schema version, jobs by state and the queue's lag |

Point an external uptime monitor at `/readyz`. Alert on three things on the
Health page: **Retrying** or **Gave up** above zero (an email is not
sending), **Waiting** growing, and a **lag** of more than a minute (the worker
is not running).

`bin/ecommerce routes` lists what is mounted. `kamal ready` and `kamal status`
(aliases in `deploy.yml`) ask a running container for `/readyz` and `/healthz`.

## Logs

One JSON line per request on stdout: `method`, `path` (the route template,
never the URL, so a query string with an email or token is never logged),
`status`, `took`, `bytes`, `ip`, `request_id` and `subject` (the token's
user or guest id). A 5xx is an `ERROR`; everything else is `INFO`; the health
probes are `DEBUG`, so at the default level they are silent.

The `request_id` is also the `X-Request-Id` response header and is on every
error line the request caused, so a customer's report ("it failed at 3pm,
here is the id") finds the line and its cause: `kamal logs | grep <id>`.
Level is `server.log_level` (`SERVER_LOG_LEVEL`): debug, info, warn, error.
No header, body or password is ever logged.

## Deploying safely

`kamal deploy` starts the new container beside the old one. It applies pending
migrations (under a database lock, so the web and job containers starting
together do not both run them) and takes traffic only when `/readyz` answers
200 within `deploy_timeout` (60s). If it never does, the deploy fails and the
old container keeps serving. The old one gets `drain_timeout` (30s) to finish
its requests; the app itself waits `server.shutdown_grace` (10s) after SIGTERM.

**A migration must work with the code that is still running**, because for a
few seconds the old release serves against the new schema. So change the
schema in two releases, never one:

- **Add** a column or table first (nullable, or with a default), deploy; then
  deploy the code that uses it.
- **Remove** by deploying code that no longer uses it first; then drop it.
- **Rename** is add + copy + switch + drop, never `RENAME`.

## Rolling back

- **Code:** `kamal rollback VERSION` (the sha from `kamal app version` or the
  Health page's release). The previous image starts, passes `/readyz`, and
  takes over.
- **Schema, only if a migration itself is the problem:**
  `kamal app exec "ecommerce migrate status"`, then
  `kamal app exec "ecommerce migrate down --yes"`. Migrations drop on the way
  down and there is no undo, so prefer fixing forward with a new migration.

## Accounts

The first staff account is made by signing up on the site, then
`kamal app exec "ecommerce staff promote you@example.com"`. The development
staff login in `db/seeds` never exists on a deployed system.

## Backups

The showcase does not back up its database, on purpose: it holds seed data and
test accounts, and `kamal setup` plus `ecommerce db seed` rebuilds it in a few
minutes. **A shop with real customers must, and this is the first thing to add
before it has any.** The database is the only thing worth keeping; the image, the
config and the credentials are all in git and the registry.

What to do, in the order that matters:

1. **Dump it.** Postgres runs as the `ecommerce-db` accessory, so the dump runs
   inside that container and streams to a file on the VM:

   ```
   kamal accessory exec db "pg_dump -U ecommerce -Fc ecommerce_production" > shop-$(date +%F).dump
   ```

   `-Fc` is the compressed custom format that `pg_restore` reads. Run it from cron
   or a systemd timer on the VM, nightly at least; how often is "how much work can
   we afford to lose".
2. **Copy it off the VM.** A backup on the machine that died is not a backup. Object
   storage (any S3-compatible bucket) or a second host, with an expiry so old dumps
   do not pile up. Encrypt it if the bucket is not yours alone: it holds customer
   emails and password digests.
3. **Restore one, before you need it.** A backup is real only once you have restored
   it into an empty database and the shop worked against it:

   ```
   createdb -U ecommerce ecommerce_restore_check
   pg_restore -U ecommerce -d ecommerce_restore_check shop-2026-10-25.dump
   ```

   Do this on a schedule too; a dump that has silently stopped being valid looks
   exactly like one that is fine.

A managed Postgres (the provider's point-in-time recovery) replaces steps 1 and 2 and
is the better answer once the data matters; step 3 still applies. That is the
"Where to go next" version of this section.
