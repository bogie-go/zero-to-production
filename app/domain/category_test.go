package domain

import (
	"errors"
	"testing"
)

func TestCategoryValidate(t *testing.T) {
	if err := (Category{Name: "Audio"}).Validate(); err != nil {
		t.Errorf("a named category: %v", err)
	}
	for _, name := range []string{"", "   "} {
		if err := (Category{Name: name}).Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("name %q: err = %v, want ErrInvalid", name, err)
		}
	}
}
