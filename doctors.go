package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DoctorsService covers branches, locations, search and doctor detail.
type DoctorsService struct{ t *transport }

// Branches returns the branch list.
func (s *DoctorsService) Branches(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/patients/allBranches", auth: authBearer})
}

// Locations returns the city list.
func (s *DoctorsService) Locations(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/patients/allLocations", auth: authBearer})
}

// QuickSearch performs autocomplete search. listType and location may be empty.
func (s *DoctorsService) QuickSearch(ctx context.Context, searchText, listType, location string) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/quickSearch", auth: authBearer, body: map[string]any{
		"searchText": searchText,
		"listType":   strOrNil(listType),
		"location":   strOrNil(location),
	}})
}

// Search performs filtered doctor search.
func (s *DoctorsService) Search(ctx context.Context, in SearchInput) (json.RawMessage, error) {
	searchParams := in.SearchParams
	if searchParams == nil {
		searchParams = map[string]any{}
	}
	orderParams := in.OrderParams
	if orderParams == nil {
		orderParams = []string{}
	}
	otherParams := in.OtherParams
	if otherParams == nil {
		otherParams = []string{}
	}
	page := in.CurrentPage
	if page == 0 {
		page = 1
	}
	limit := in.PerPageLimit
	if limit == 0 {
		limit = 20
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/filteredSearch", auth: authBearer, body: map[string]any{
		"searchParams": searchParams,
		"orderParams":  orderParams,
		"otherParams":  otherParams,
		"currentPage":  page,
		"perPageLimit": limit,
	}})
}

// Detail returns doctor detail. Pass corporate == nil to omit the segment.
func (s *DoctorsService) Detail(ctx context.Context, id, corporate any) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/doctorDetail/%v", id)
	if corporate != nil {
		path += fmt.Sprintf("/%v", corporate)
	}
	return s.t.do(ctx, request{method: http.MethodGet, path: path, auth: authBearer})
}
