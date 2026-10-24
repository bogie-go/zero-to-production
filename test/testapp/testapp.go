// Package testapp is the whole application for a test: the real router, the
// real controllers and the real store on a clean test database, with a
// request helper on top. A test written with it is a request test, the
// outermost layer: it sends HTTP in and checks the status and JSON that come
// back, which is the contract the UI relies on.
//
//	app := testapp.New(t)
//	product := factories.Product(t, app.Store)
//	res := app.Get("/products")
//	res.Status(http.StatusOK)
//
// Background jobs are inserted, not worked: a request test asserts on what a
// request queued (Jobs), and the worker's own tests cover what a job does.
package testapp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"ecommerce/app"
	"ecommerce/app/domain"
	"ecommerce/test/testdb"
)

// T is the part of *testing.T the helpers need, so they also work with
// Ginkgo's GinkgoT().
type T interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Cleanup(func())
	Setenv(key, value string)
}

// App is a running application and the helpers around it.
type App struct {
	*app.Application
	t T
}

// New builds the application against an emptied test database. The
// environment is forced to test by testdb, so development data is safe.
func New(t T) *App {
	t.Helper()
	// The request log is noise in a test; failures still print.
	t.Setenv("SERVER_LOG_LEVEL", "error")
	testdb.Store(t) // migrates and empties the tables, and pins the environment
	a, err := app.NewApplication(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Close)
	return &App{Application: a, t: t}
}

// Request is one HTTP request being built; Do sends it.
type Request struct {
	app           *App
	method, path  string
	body          any
	authorization string
}

// Get starts a GET request.
func (a *App) Get(path string) *Request { return a.request(http.MethodGet, path) }

// Post starts a POST request with a JSON body (nil for none).
func (a *App) Post(path string, body any) *Request {
	return a.request(http.MethodPost, path).Body(body)
}

// Patch starts a PATCH request with a JSON body.
func (a *App) Patch(path string, body any) *Request {
	return a.request(http.MethodPatch, path).Body(body)
}

// Delete starts a DELETE request.
func (a *App) Delete(path string) *Request { return a.request(http.MethodDelete, path) }

func (a *App) request(method, path string) *Request {
	return &Request{app: a, method: method, path: path}
}

// Body sets the JSON body: a string is sent as it is, anything else is encoded.
func (r *Request) Body(body any) *Request {
	r.body = body
	return r
}

// AsUser signs the request in as that account.
func (r *Request) AsUser(u domain.User) *Request { return r.as(strconv.FormatInt(u.ID, 10)) }

// AsGuest signs the request in as a guest with that id, which is its cart's
// token. Any id will do; the server never looks a guest up.
func (r *Request) AsGuest(id string) *Request { return r.as(id) }

func (r *Request) as(subject string) *Request {
	signed, err := r.app.Server.Tokens().Mint(subject, time.Hour)
	if err != nil {
		r.app.t.Fatal(err)
	}
	r.authorization = "Bearer " + signed
	return r
}

// WithAuthorization sets the Authorization header as it is, for the cases
// that send a bad or foreign token.
func (r *Request) WithAuthorization(v string) *Request {
	r.authorization = v
	return r
}

// Do sends the request through the real router.
func (r *Request) Do() *Response {
	t := r.app.t
	t.Helper()
	var reader *bytes.Reader
	switch b := r.body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(b))
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(r.method, r.path, reader)
	req.Header.Set("Content-Type", "application/json")
	if r.authorization != "" {
		req.Header.Set("Authorization", r.authorization)
	}
	rec := httptest.NewRecorder()
	r.app.Server.Router.ServeHTTP(rec, req)
	return &Response{t: t, Code: rec.Code, Body: rec.Body.Bytes()}
}

// Response is what came back.
type Response struct {
	t    T
	Code int
	Body []byte
}

// Status fails the test, showing the body, unless the status is want.
func (r *Response) Status(want int) *Response {
	r.t.Helper()
	if r.Code != want {
		r.t.Fatalf("status = %d, want %d; body: %s", r.Code, want, r.Body)
	}
	return r
}

// JSON decodes the body into v.
func (r *Response) JSON(v any) *Response {
	r.t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		r.t.Fatalf("body is not the JSON expected: %v\n%s", err, r.Body)
	}
	return r
}

// Contains fails the test unless the body has the text.
func (r *Response) Contains(text string) *Response {
	r.t.Helper()
	if !strings.Contains(string(r.Body), text) {
		r.t.Fatalf("body = %s, want it to contain %q", r.Body, text)
	}
	return r
}

// Jobs counts the queued background jobs of a kind ("send_order_confirmation").
func (a *App) Jobs(kind string) int {
	a.t.Helper()
	var n int
	if err := a.Store.Pool().QueryRow(context.Background(), "SELECT count(*) FROM river_job WHERE kind = $1", kind).Scan(&n); err != nil {
		a.t.Fatal(err)
	}
	return n
}
