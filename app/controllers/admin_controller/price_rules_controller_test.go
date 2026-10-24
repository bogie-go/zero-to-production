package admin_controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/controllers/admin_controller"
	"ecommerce/app/domain"
)

type fakePriceRulesStore struct {
	rule domain.PriceRule
	list []domain.PriceRule
	err  error

	saved   domain.PriceRule
	gotID   string
	gotOn   *bool
	deleted bool
}

func (f *fakePriceRulesStore) ListAllPriceRules(context.Context) ([]domain.PriceRule, error) {
	return f.list, f.err
}
func (f *fakePriceRulesStore) GetPriceRule(_ context.Context, id string) (domain.PriceRule, error) {
	f.gotID = id
	return f.rule, f.err
}
func (f *fakePriceRulesStore) CreatePriceRule(_ context.Context, p domain.PriceRule) (domain.PriceRule, error) {
	f.saved = p
	p.ID = 9
	return p, f.err
}
func (f *fakePriceRulesStore) UpdatePriceRule(_ context.Context, p domain.PriceRule) (domain.PriceRule, error) {
	f.saved = p
	return p, f.err
}
func (f *fakePriceRulesStore) DeletePriceRule(_ context.Context, id string) error {
	f.gotID, f.deleted = id, true
	return f.err
}
func (f *fakePriceRulesStore) SetPriceRuleActive(_ context.Context, id string, active bool) (domain.PriceRule, error) {
	f.gotID, f.gotOn = id, &active
	r := f.rule
	r.Active = active
	return r, f.err
}

var noon = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

func adminPriceRulesRequest(store *fakePriceRulesStore, staff fakeStaffStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ns := admin_controller.NewServer(staff, testTokens, log)
	ns.Products = admin_controller.NewProductsController(&fakeProductsStore{}, log)
	ns.Categories = admin_controller.NewCategoriesController(&fakeCategoriesStore{}, log)
	ns.PriceRules = admin_controller.NewPriceRulesController(store, log)
	ns.PriceRules.Now = func() time.Time { return noon }
	ns.SetupRoutes(&router.RouterGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("admin price rules", func() {
	var store *fakePriceRulesStore
	var staff string

	BeforeEach(func() {
		store = &fakePriceRulesStore{}
		staff = bearerFor("1")
	})

	Describe("staff gate", func() {
		It("answers 401 with no token", func() {
			rec := adminPriceRulesRequest(store, aStaffUser, http.MethodGet, "/admin/price_rules", "", "")
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})

		It("answers 403 for a customer and changes nothing", func() {
			customer := fakeStaffStore{user: domain.User{ID: 9, Role: domain.RoleCustomer}}
			rec := adminPriceRulesRequest(store, customer, http.MethodPost, "/admin/price_rules/3/deactivate", "", bearerFor("9"))
			Expect(rec.Code).To(Equal(http.StatusForbidden))
			Expect(store.gotOn).To(BeNil())
		})
	})

	// The form is built from this: every registered kind with its fields.
	Describe("GET /admin/price_rules/kinds", func() {
		It("describes every kind", func() {
			rec := adminPriceRulesRequest(store, aStaffUser, http.MethodGet, "/admin/price_rules/kinds", "", staff)
			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			for _, want := range []string{`"kind":"fixed_discount"`, `"kind":"percentage_discount"`, `"kind":"minimum_purchase"`, `"kind":"buy_x_get_y"`,
				`"name":"amount_cents"`, `"type":"cents"`, `"type":"percent"`, `"note":"For now this only records`} {
				Expect(rec.Body.String()).To(ContainSubstring(want))
			}
		})
	})

	Describe("GET /admin/price_rules", func() {
		It("summarizes each rule and says where it stands", func() {
			past, future := noon.Add(-time.Hour), noon.Add(time.Hour)
			store.list = []domain.PriceRule{
				{ID: 1, Name: "Sale", Kind: "percentage_discount", Properties: json.RawMessage(`{"percentage":20}`), Active: true, Priority: 10},
				{ID: 2, Name: "Paused", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":5000}`), Active: false},
				{ID: 3, Name: "Soon", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":100}`), Active: true, StartsAt: &future},
				{ID: 4, Name: "Over", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":100}`), Active: true, EndsAt: &past},
			}
			rec := adminPriceRulesRequest(store, aStaffUser, http.MethodGet, "/admin/price_rules", "", staff)
			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())

			var got struct {
				PriceRules []struct {
					Name      string
					Summary   string
					Status    string
					KindLabel string `json:"kind_label"`
				} `json:"price_rules"`
			}
			Expect(json.Unmarshal(rec.Body.Bytes(), &got)).To(Succeed())

			var lines []string
			for _, r := range got.PriceRules {
				lines = append(lines, r.Name+"|"+r.Summary+"|"+r.Status+"|"+r.KindLabel)
			}
			Expect(lines).To(Equal([]string{"Sale|20% off|active|Percentage off", "Paused|$50.00 off|off|Fixed amount off", "Soon|$1.00 off|scheduled|Fixed amount off", "Over|$1.00 off|ended|Fixed amount off"}))
		})

		It("answers 500 without the cause when the store fails", func() {
			rec := adminPriceRulesRequest(&fakePriceRulesStore{err: errors.New("db down")}, aStaffUser, http.MethodGet, "/admin/price_rules", "", staff)
			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			Expect(rec.Body.String()).NotTo(ContainSubstring("db down"))
		})
	})

	Describe("POST /admin/price_rules", func() {
		// What is stored is the rule decoded by its own kind and written back
		// out: unknown keys the client sent never reach the database.
		It("stores the kind's own JSON", func() {
			body := `{"name":"Spring","kind":"percentage_discount","properties":{"percentage":15,"evil":"x"},"active":true,"priority":5}`
			rec := adminPriceRulesRequest(store, aStaffUser, http.MethodPost, "/admin/price_rules", body, staff)
			Expect(rec.Code).To(Equal(http.StatusCreated), rec.Body.String())
			Expect(string(store.saved.Properties)).To(Equal(`{"percentage":15}`))
			Expect(store.saved.Name).To(Equal("Spring"))
			Expect(store.saved.Active).To(BeTrue())
			Expect(store.saved.Priority).To(BeEquivalentTo(5))
		})

		// Each kind's own validation answers 422 with its message: nothing here
		// knows the rules. (A blank name is caught by the store's Validate in
		// production; the fake has none, so it is checked at the model.)
		DescribeTable("refuses what the kind refuses",
			func(body string) {
				rec := adminPriceRulesRequest(store, aStaffUser, http.MethodPost, "/admin/price_rules", body, staff)
				Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
				Expect(store.saved.Name).To(BeEmpty(), "something was saved")
			},
			Entry("percentage over 100", `{"name":"x","kind":"percentage_discount","properties":{"percentage":500}}`),
			Entry("negative amount", `{"name":"x","kind":"fixed_discount","properties":{"amount_cents":-5}}`),
			Entry("buy needs quantities", `{"name":"x","kind":"buy_x_get_y","properties":{}}`),
			Entry("unknown kind", `{"name":"x","kind":"nope","properties":{}}`),
			Entry("properties not an object", `{"name":"x","kind":"fixed_discount","properties":[1]}`),
		)

		It("answers 400 for malformed JSON", func() {
			rec := adminPriceRulesRequest(store, aStaffUser, http.MethodPost, "/admin/price_rules", `{"nope":`, staff)
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("GET, PATCH, DELETE /admin/price_rules/:id", func() {
		BeforeEach(func() {
			store.rule = domain.PriceRule{ID: 4, Name: "Sale", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":1}`)}
		})

		It("shows", func() {
			rec := adminPriceRulesRequest(store, aStaffUser, http.MethodGet, "/admin/price_rules/4", "", staff)
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("updates, carrying the id over", func() {
			body := `{"name":"New","kind":"fixed_discount","properties":{"amount_cents":700},"active":false,"priority":3}`
			rec := adminPriceRulesRequest(store, aStaffUser, http.MethodPatch, "/admin/price_rules/4", body, staff)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(store.saved.ID).To(BeEquivalentTo(4))
			Expect(store.saved.Name).To(Equal("New"))
			Expect(store.saved.Active).To(BeFalse())
		})

		It("deletes", func() {
			rec := adminPriceRulesRequest(store, aStaffUser, http.MethodDelete, "/admin/price_rules/4", "", staff)
			Expect(rec.Code).To(Equal(http.StatusNoContent))
			Expect(store.deleted).To(BeTrue())
		})

		It("answers 404 for an unknown id", func() {
			rec := adminPriceRulesRequest(&fakePriceRulesStore{err: domain.ErrNotFound}, aStaffUser, http.MethodGet, "/admin/price_rules/9", "", staff)
			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})

		It("turns on and off", func() {
			rec := adminPriceRulesRequest(store, aStaffUser, http.MethodPost, "/admin/price_rules/4/activate", "", staff)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(store.gotOn).NotTo(BeNil())
			Expect(*store.gotOn).To(BeTrue())
			Expect(store.gotID).To(Equal("4"))
			Expect(rec.Body.String()).To(ContainSubstring(`"status":"active"`))

			rec = adminPriceRulesRequest(store, aStaffUser, http.MethodPost, "/admin/price_rules/4/deactivate", "", staff)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(*store.gotOn).To(BeFalse())
			Expect(rec.Body.String()).To(ContainSubstring(`"status":"off"`))
		})
	})
})
