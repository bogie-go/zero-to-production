package testapp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

// The API contract is docs/openapi.yaml. Every request test checks what it
// sends and what comes back against it, so a response the file does not
// describe fails the test that provoked it, and so does a spec the code has
// outgrown. A request the spec calls invalid is only checked when the
// server accepted it: a test that sends a bad body on purpose is asserting
// the 4xx, which the spec documents as the answer.

var (
	contractOnce   sync.Once
	contractDoc    *openapi3.T
	contractRouter routers.Router
	contractErr    error
)

func loadContract() (*openapi3.T, routers.Router, error) {
	contractOnce.Do(func() {
		_, here, _, _ := runtime.Caller(0)
		path := filepath.Join(filepath.Dir(here), "..", "..", "docs", "openapi.yaml")
		doc, err := openapi3.NewLoader().LoadFromFile(path)
		if err != nil {
			contractErr = err
			return
		}
		if err := doc.Validate(context.Background()); err != nil {
			contractErr = err
			return
		}
		// The servers describe where the API lives; a test request has no
		// host worth matching.
		doc.Servers = nil
		router, err := legacy.NewRouter(doc)
		if err != nil {
			contractErr = err
			return
		}
		contractDoc, contractRouter = doc, router
	})
	return contractDoc, contractRouter, contractErr
}

// ContractRoutes lists "METHOD /path" for every operation in the spec, with
// path parameters in Gin's :name form, so a test can compare it with the
// routes the server registered.
func ContractRoutes() ([]string, error) {
	doc, _, err := loadContract()
	if err != nil {
		return nil, err
	}
	var out []string
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			out = append(out, method+" "+ginPath(path))
		}
	}
	sort.Strings(out)
	return out, nil
}

// ginPath turns /orders/{id} into /orders/:id.
func ginPath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			parts[i] = ":" + s[1:len(s)-1]
		}
	}
	return strings.Join(parts, "/")
}

// checkContract validates one exchange. A path the spec does not know is
// left alone: those are the static files and the welcome page, and a route
// registered without a spec entry is caught by the wiring test instead.
func checkContract(t T, req *http.Request, body []byte, rec *httptest.ResponseRecorder) {
	t.Helper()
	_, router, err := loadContract()
	if err != nil {
		t.Fatalf("docs/openapi.yaml is not a valid spec: %v", err)
		return
	}
	// The recorded request was consumed by the handler; validate a copy.
	check := req.Clone(req.Context())
	check.Body = io.NopCloser(bytes.NewReader(body))

	route, pathParams, err := router.FindRoute(check)
	if err != nil {
		return
	}
	input := &openapi3filter.RequestValidationInput{
		Request:    check,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			// Authentication is the server's job and has its own tests;
			// the spec only declares that a bearer token is expected.
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
	}
	if rec.Code < 400 {
		if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
			t.Fatalf("%s %s is accepted by the server but not allowed by docs/openapi.yaml:\n%v", req.Method, req.URL.Path, err)
			return
		}
	}
	out := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: input,
		Status:                 rec.Code,
		Header:                 rec.Header(),
		Options:                &openapi3filter.Options{IncludeResponseStatus: true},
	}
	out.SetBodyBytes(rec.Body.Bytes())
	if err := openapi3filter.ValidateResponse(context.Background(), out); err != nil {
		t.Fatalf("%s %s answered %d, which docs/openapi.yaml does not describe:\n%v\nbody: %s",
			req.Method, req.URL.Path, rec.Code, err, rec.Body.String())
	}
}
