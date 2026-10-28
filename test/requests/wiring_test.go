package requests_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The Server is built in app/application.go and mounted in routes.go. These
// specs say it is all connected: nothing left unconstructed, nothing declared
// and not mounted.
var _ = Describe("The wired server", func() {
	// nilFields names every pointer field of s that was never assigned. A namespace
	// (admin_controllers.Server) is entered.
	var nilFields func(prefix string, s any) []string
	nilFields = func(prefix string, s any) []string {
		var missing []string
		v := reflect.ValueOf(s).Elem()
		for i := range v.NumField() {
			f, name := v.Field(i), prefix+v.Type().Field(i).Name
			if f.Kind() != reflect.Pointer || !f.CanInterface() || v.Type().Field(i).Name == "Router" || v.Type().Field(i).Name == "Log" {
				continue
			}
			if f.IsNil() {
				missing = append(missing, name)
			} else if f.Elem().Kind() == reflect.Struct && v.Type().Field(i).Name == "Admin" {
				missing = append(missing, nilFields(name+".", f.Interface())...)
			}
		}
		return missing
	}

	It("has every controller constructed", func() {
		Expect(nilFields("", app.Server)).To(BeEmpty())
	})

	It("mounts a handler on every route", func() {
		routes := app.Server.Router.Routes()
		Expect(routes).NotTo(BeEmpty())
		for _, r := range routes {
			Expect(r.HandlerFunc).NotTo(BeNil(), "%s %s", r.Method, r.Path)
		}
	})

	It("mounts the routes the UI relies on", func() {
		mounted := map[string]bool{}
		for _, r := range app.Server.Router.Routes() {
			mounted[r.Method+" "+r.Path] = true
		}
		for _, want := range []string{
			"GET /healthz", "GET /products", "GET /products/:id", "GET /categories",
			"POST /guest_sessions", "POST /users", "POST /sessions", "GET /me",
			"GET /cart", "POST /cart_items", "PATCH /cart_items/:id", "DELETE /cart_items/:id",
			"POST /orders", "GET /orders", "GET /orders/:id",
			"GET /admin/products", "POST /admin/products", "GET /admin/categories", "GET /admin/price_rules",
		} {
			Expect(mounted).To(HaveKey(want))
		}
	})
})
