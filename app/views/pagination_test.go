package views

import "testing"

func TestNewPageRequest(t *testing.T) {
	for _, c := range []struct {
		page, per string
		want      PageRequest
		ok        bool
	}{
		{"", "", PageRequest{1, 12}, true},
		{"3", "", PageRequest{3, 12}, true},
		{"2", "5", PageRequest{2, 5}, true},
		{"", "500", PageRequest{1, 50}, true}, // too much is brought down, not refused
		{"0", "", PageRequest{}, false},
		{"-1", "", PageRequest{}, false},
		{"abc", "", PageRequest{}, false},
		{"1.5", "", PageRequest{}, false},
		{"10001", "", PageRequest{}, false},
		{"", "0", PageRequest{}, false},
		{"", "many", PageRequest{}, false},
		{"99999999999", "", PageRequest{}, false},
	} {
		got, err := NewPageRequest(c.page, c.per, 12, 50)
		if (err == nil) != c.ok || (c.ok && got != c.want) {
			t.Errorf("NewPageRequest(%q, %q) = %+v, %v; want %+v ok=%v", c.page, c.per, got, err, c.want, c.ok)
		}
	}
}

func TestOffsetAndPagination(t *testing.T) {
	r := PageRequest{Page: 3, PerPage: 12}
	if r.Offset() != 24 {
		t.Errorf("Offset = %d, want 24", r.Offset())
	}
	for total, pages := range map[int64]int32{0: 0, 1: 1, 12: 1, 13: 2, 36: 3, 37: 4} {
		if got := r.Pagination(total); got.TotalPages != pages || got.Total != total || got.Page != 3 || got.PerPage != 12 {
			t.Errorf("Pagination(%d) = %+v, want %d pages", total, got, pages)
		}
	}
	// 10000 * 50 must still fit the int32 the database call takes.
	if off := (PageRequest{Page: maxPage, PerPage: 50}).Offset(); off != (maxPage-1)*50 || off < 0 {
		t.Errorf("the largest allowed offset = %d", off)
	}
}
