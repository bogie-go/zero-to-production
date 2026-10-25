# The API contract

`docs/openapi.yaml` is the contract between the API and its UI (`public/`).
It is an OpenAPI 3.0 file written by hand: every path, the JSON each one
takes and returns, and the status codes it can answer with. Open it in any
OpenAPI viewer (Swagger Editor, Redoc, an editor plugin), or just read it; it
is plain YAML.

## It is checked, not just written

Every request test goes through `test/testapp`, and `testapp` checks each
exchange against the file ([kin-openapi](https://github.com/getkin/kin-openapi),
a test-only dependency):

- **A response the spec does not describe fails the test.** That covers a
  status code the operation does not list, a field with the wrong type, a
  missing required field, and a field the code sends that the spec does not
  mention (response objects are closed).
- **A request the server accepted must be one the spec allows.** A test that
  sends a bad body on purpose is asserting the 4xx and is not held to the
  request schema.
- **The routes match.** `test/requests/contract_test.go` compares the routes
  the server mounts with the operations in the file, both ways, so a new
  route without an entry fails, and so does an entry for a route that does
  not exist.

So the file cannot drift quietly. Change a view struct and the tests that
touch it fail until the spec says the same.

## Changing the API

1. Change the code and the spec together (the UI is the contract, so the UI
   too).
2. Run `go test -p 1 ./...`. The message names the operation and the field.
3. Money is whole cents. Errors are always `{"status": "error", "error": "..."}`.

## What it does not check

Only exchanges a test actually makes are validated. An endpoint with no
request test is covered by the route comparison (it must exist in the spec)
but not by a shape check, so every endpoint should have a request test for
its success and its main failures. Authentication itself is the server's job
and has its own tests; the spec only declares that a bearer token is
expected.
