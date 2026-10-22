package views

import (
	"fmt"
	"strconv"
)

// Pagination is the paging block of a list response: where this page is, and
// how much there is in all, so a client can draw "Page 2 of 4" and decide
// whether there is a next page without asking for it.
type Pagination struct {
	Page       int32 `json:"page"`
	PerPage    int32 `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int32 `json:"total_pages"`
}

// PageRequest is which page a client asked for: ?page=2&per_page=12. Offset
// paging, because a catalog wants page numbers and a total. (Keyset paging,
// "everything after this id", is cheaper on very deep pages and does not shift
// when rows are added, but it cannot say "page 3 of 7"; it is the upgrade for
// an infinite scroll or a table of millions.)
type PageRequest struct {
	Page    int32
	PerPage int32
}

// maxPage keeps the offset (page * per_page) inside what the database call
// takes, and is far beyond any real catalog; past it, narrow the list instead.
const maxPage = 10000

// NewPageRequest reads the two query values, either of which may be empty. A
// missing page is 1 and a missing per_page is def; a per_page above max is
// brought down to it (asking for too much gets what is allowed, not an error).
// A value that is not a whole number of at least 1, or a page past maxPage, is
// an error the caller answers 422.
func NewPageRequest(rawPage, rawPerPage string, def, max int32) (PageRequest, error) {
	req := PageRequest{Page: 1, PerPage: def}
	if rawPage != "" {
		n, err := strconv.ParseInt(rawPage, 10, 32)
		if err != nil || n < 1 || n > maxPage {
			return PageRequest{}, fmt.Errorf("page must be a whole number from 1 to %d", maxPage)
		}
		req.Page = int32(n)
	}
	if rawPerPage != "" {
		n, err := strconv.ParseInt(rawPerPage, 10, 32)
		if err != nil || n < 1 {
			return PageRequest{}, fmt.Errorf("per_page must be a whole number of at least 1")
		}
		req.PerPage = int32(n)
	}
	if req.PerPage > max {
		req.PerPage = max
	}
	return req, nil
}

// Offset is how many rows come before this page.
func (r PageRequest) Offset() int32 { return (r.Page - 1) * r.PerPage }

// Pagination is the response block for a list of total rows. A page past the
// last is not an error: it is an empty list that says how many pages there are.
func (r PageRequest) Pagination(total int64) Pagination {
	pages := int32((total + int64(r.PerPage) - 1) / int64(r.PerPage))
	return Pagination{Page: r.Page, PerPage: r.PerPage, Total: total, TotalPages: pages}
}
