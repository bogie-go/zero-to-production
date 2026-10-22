package controllers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ecommerce/app/domain"
)

type fakeCategoriesStore struct {
	list []domain.Category
	err  error
}

func (f fakeCategoriesStore) ListCategoriesWithCounts(context.Context) ([]domain.Category, error) {
	return f.list, f.err
}

func categoriesRequest(t *testing.T, store fakeCategoriesStore) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewCategoriesController(store, slog.New(slog.NewTextHandler(io.Discard, nil))).SetupRoutes(&router.RouterGroup)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/categories", nil))
	return rec
}

func TestCategoriesIndexListsEachWithItsCount(t *testing.T) {
	rec := categoriesRequest(t, fakeCategoriesStore{list: []domain.Category{
		{ID: 1, Name: "Audio", ProductCount: 3}, {ID: 2, Name: "Empty"},
	}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	for _, want := range []string{`"name":"Audio","product_count":3`, `"name":"Empty","product_count":0`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("body = %s, want %s", rec.Body, want)
		}
	}
}

// No categories is an empty list, not null: the UI iterates it.
func TestCategoriesIndexIsAnEmptyListNotNull(t *testing.T) {
	if body := categoriesRequest(t, fakeCategoriesStore{}).Body.String(); !strings.Contains(body, `"categories":[]`) {
		t.Errorf("body = %s, want an empty array", body)
	}
}

func TestCategoriesIndexAnswers500WithoutLeaking(t *testing.T) {
	rec := categoriesRequest(t, fakeCategoriesStore{err: errors.New("connection refused to db-1")})
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db-1") {
		t.Errorf("status %d, body %s; want 500 without the cause", rec.Code, rec.Body)
	}
}
