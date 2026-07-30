package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DoctorsService covers doctor discovery. Results are scoped to the doctors
// enabled for your integration (the server filters on your partner slug), so a
// doctor returned here is one you can actually book. [DoctorsService.Locations]
// is the exception — a global city catalogue, not company-scoped.
type DoctorsService struct{ t *transport }

// Search runs a filtered doctor search. orderParams accepts "name", "order" and
// "slot".
//
// searchParams must carry at least one key: the server rule is required|array and
// PHP's required rejects an empty map, so an empty or nil searchParams is a
// validation error rather than an unfiltered search.
func (s *DoctorsService) Search(ctx context.Context, searchParams map[string]any, currentPage int, orderParams []string) (json.RawMessage, error) {
	if orderParams == nil {
		orderParams = []string{}
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/search", auth: authPartner, body: map[string]any{
		"searchParams": searchParams,
		"orderParams":  orderParams,
		"currentPage":  currentPage,
	}})
}

// Branches lists the branches available through your integration.
func (s *DoctorsService) Branches(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/outher/branches", auth: authPartner})
}

// Detail returns a single doctor. The doctor_id here feeds [SlotsService.Schedule].
func (s *DoctorsService) Detail(ctx context.Context, doctorID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: fmt.Sprintf("/outher/doctorInfos/%v", doctorID), auth: authPartner})
}

// Locations returns the city list. Global catalogue — not scoped to your company.
func (s *DoctorsService) Locations(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/outher/locations", auth: authPartner})
}
