package admin_controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

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

func adminPriceRulesRequest(t *testing.T, store *fakePriceRulesStore, staff fakeStaffStore, method, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ns := NewServer(staff, testTokens, log)
	ns.Products = NewProductsController(&fakeProductsStore{}, log)
	ns.Categories = NewCategoriesController(&fakeCategoriesStore{}, log)
	ns.PriceRules = NewPriceRulesController(store, log)
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

func TestAdminPriceRulesRequireStaff(t *testing.T) {
	store := &fakePriceRulesStore{}
	if rec := adminPriceRulesRequest(t, store, aStaffUser, http.MethodGet, "/admin/price_rules", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: %d, want 401", rec.Code)
	}
	customer := fakeStaffStore{user: domain.User{ID: 9, Role: domain.RoleCustomer}}
	rec := adminPriceRulesRequest(t, store, customer, http.MethodPost, "/admin/price_rules/3/deactivate", "", bearerFor(t, "9"))
	if rec.Code != http.StatusForbidden || store.gotOn != nil {
		t.Errorf("customer: %d (toggled %v), want 403 and nothing changed", rec.Code, store.gotOn)
	}
}

// The form is built from this: every registered kind with its fields.
func TestAdminPriceRuleKindsDescribeEveryKind(t *testing.T) {
	rec := adminPriceRulesRequest(t, &fakePriceRulesStore{}, aStaffUser, http.MethodGet, "/admin/price_rules/kinds", "", bearerFor(t, "1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	for _, want := range []string{`"kind":"fixed_discount"`, `"kind":"percentage_discount"`, `"kind":"minimum_purchase"`, `"kind":"buy_x_get_y"`,
		`"name":"amount_cents"`, `"type":"cents"`, `"type":"percent"`, `"note":"For now this only records`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("kinds lacks %s:\n%s", want, rec.Body)
		}
	}
}

func TestAdminPriceRulesIndexSummarizesEachAndSaysWhereItStands(t *testing.T) {
	past, future := noon.Add(-time.Hour), noon.Add(time.Hour)
	store := &fakePriceRulesStore{list: []domain.PriceRule{
		{ID: 1, Name: "Sale", Kind: "percentage_discount", Properties: json.RawMessage(`{"percentage":20}`), Active: true, Priority: 10},
		{ID: 2, Name: "Paused", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":5000}`), Active: false},
		{ID: 3, Name: "Soon", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":100}`), Active: true, StartsAt: &future},
		{ID: 4, Name: "Over", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":100}`), Active: true, EndsAt: &past},
	}}
	rec := adminPriceRulesRequest(t, store, aStaffUser, http.MethodGet, "/admin/price_rules", "", bearerFor(t, "1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var got struct {
		PriceRules []struct {
			Name      string
			Summary   string
			Status    string
			KindLabel string `json:"kind_label"`
		} `json:"price_rules"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := []string{"Sale|20% off|active|Percentage off", "Paused|$50.00 off|off|Fixed amount off", "Soon|$1.00 off|scheduled|Fixed amount off", "Over|$1.00 off|ended|Fixed amount off"}
	for i, r := range got.PriceRules {
		if line := r.Name + "|" + r.Summary + "|" + r.Status + "|" + r.KindLabel; line != want[i] {
			t.Errorf("rule %d = %q, want %q", i, line, want[i])
		}
	}
}

// What is stored is the rule decoded by its own kind and written back out:
// unknown keys the client sent never reach the database.
func TestAdminPriceRulesCreateStoresTheKindsOwnJSON(t *testing.T) {
	store := &fakePriceRulesStore{}
	body := `{"name":"Spring","kind":"percentage_discount","properties":{"percentage":15,"evil":"x"},"active":true,"priority":5}`
	rec := adminPriceRulesRequest(t, store, aStaffUser, http.MethodPost, "/admin/price_rules", body, bearerFor(t, "1"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if string(store.saved.Properties) != `{"percentage":15}` || store.saved.Name != "Spring" || !store.saved.Active || store.saved.Priority != 5 {
		t.Errorf("saved = %+v (properties %s), want only the kind's own keys", store.saved, store.saved.Properties)
	}
}

// Each kind's own validation answers 422 with its message: nothing here knows
// the rules.
func TestAdminPriceRulesRefuseWhatTheKindRefuses(t *testing.T) {
	staff := bearerFor(t, "1")
	for name, body := range map[string]string{
		"percentage over 100":      `{"name":"x","kind":"percentage_discount","properties":{"percentage":500}}`,
		"negative amount":          `{"name":"x","kind":"fixed_discount","properties":{"amount_cents":-5}}`,
		"buy needs quantities":     `{"name":"x","kind":"buy_x_get_y","properties":{}}`,
		"unknown kind":             `{"name":"x","kind":"nope","properties":{}}`,
		"properties not an object": `{"name":"x","kind":"fixed_discount","properties":[1]}`,
		"no name":                  `{"name":" ","kind":"fixed_discount","properties":{"amount_cents":5}}`,
	} {
		store := &fakePriceRulesStore{}
		store.err = nil
		rec := adminPriceRulesRequest(t, store, aStaffUser, http.MethodPost, "/admin/price_rules", body, staff)
		// The blank name is caught by the store's Validate in production; the
		// fake has none, so check the rest here and the name at the model.
		if name == "no name" {
			continue
		}
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: status %d, want 422: %s", name, rec.Code, rec.Body)
		}
		if store.saved.Name != "" {
			t.Errorf("%s: something was saved", name)
		}
	}
	if rec := adminPriceRulesRequest(t, &fakePriceRulesStore{}, aStaffUser, http.MethodPost, "/admin/price_rules", `{"nope":`, staff); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed JSON: %d, want 400", rec.Code)
	}
}

func TestAdminPriceRulesUpdateCarriesTheIDOver(t *testing.T) {
	store := &fakePriceRulesStore{rule: domain.PriceRule{ID: 4, Name: "Old", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":1}`)}}
	body := `{"name":"New","kind":"fixed_discount","properties":{"amount_cents":700},"active":false,"priority":3}`
	rec := adminPriceRulesRequest(t, store, aStaffUser, http.MethodPatch, "/admin/price_rules/4", body, bearerFor(t, "1"))
	if rec.Code != http.StatusOK || store.saved.ID != 4 || store.saved.Name != "New" || store.saved.Active {
		t.Errorf("status %d, saved %+v; want id 4 kept with the new fields", rec.Code, store.saved)
	}
}

func TestAdminPriceRulesTurnOnAndOff(t *testing.T) {
	store := &fakePriceRulesStore{rule: domain.PriceRule{ID: 4, Name: "Sale", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":1}`)}}
	staff := bearerFor(t, "1")

	rec := adminPriceRulesRequest(t, store, aStaffUser, http.MethodPost, "/admin/price_rules/4/activate", "", staff)
	if rec.Code != http.StatusOK || store.gotOn == nil || !*store.gotOn || store.gotID != "4" || !strings.Contains(rec.Body.String(), `"status":"active"`) {
		t.Errorf("activate: %d %s", rec.Code, rec.Body)
	}
	rec = adminPriceRulesRequest(t, store, aStaffUser, http.MethodPost, "/admin/price_rules/4/deactivate", "", staff)
	if rec.Code != http.StatusOK || *store.gotOn || !strings.Contains(rec.Body.String(), `"status":"off"`) {
		t.Errorf("deactivate: %d %s", rec.Code, rec.Body)
	}
}

func TestAdminPriceRulesShowAndDeleteAndErrors(t *testing.T) {
	staff := bearerFor(t, "1")
	store := &fakePriceRulesStore{rule: domain.PriceRule{ID: 4, Name: "Sale", Kind: "fixed_discount", Properties: json.RawMessage(`{"amount_cents":1}`)}}
	if rec := adminPriceRulesRequest(t, store, aStaffUser, http.MethodGet, "/admin/price_rules/4", "", staff); rec.Code != http.StatusOK {
		t.Errorf("show: %d", rec.Code)
	}
	if rec := adminPriceRulesRequest(t, store, aStaffUser, http.MethodDelete, "/admin/price_rules/4", "", staff); rec.Code != http.StatusNoContent || !store.deleted {
		t.Errorf("delete: %d deleted=%v", rec.Code, store.deleted)
	}
	if rec := adminPriceRulesRequest(t, &fakePriceRulesStore{err: domain.ErrNotFound}, aStaffUser, http.MethodGet, "/admin/price_rules/9", "", staff); rec.Code != http.StatusNotFound {
		t.Errorf("unknown id: %d, want 404", rec.Code)
	}
	rec := adminPriceRulesRequest(t, &fakePriceRulesStore{err: errors.New("db down")}, aStaffUser, http.MethodGet, "/admin/price_rules", "", staff)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") {
		t.Errorf("store error: %d %s, want 500 without the cause", rec.Code, rec.Body)
	}
}
