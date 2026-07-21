package bulutklinik

import (
	"context"
	"encoding/json"
	"net/http"
)

// AddressesService covers the patient's saved addresses. Required by
// Laboratory.Order (which needs an addressId). Add/Update take a CityID (from
// Doctors.Locations) and a DistrictID (from GET /getConfig, cities[].districts[]).
type AddressesService struct{ t *transport }

// List returns the patient's saved addresses (default first). Each item's "id" is
// the addressId used by Update, Delete and Laboratory.Order.
func (s *AddressesService) List(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/patients/userAddress", auth: authBearer})
}

// Add creates an address. Success data is {"addressId": ...}. The first address is
// always the default.
func (s *AddressesService) Add(ctx context.Context, in AddressInput) (json.RawMessage, error) {
	body := map[string]any{
		"title":       in.Title,
		"cityId":      in.CityID,
		"districtId":  in.DistrictID,
		"address":     in.Address,
		"locationLat": in.LocationLat,
		"locationLng": in.LocationLng,
	}
	if in.Description != "" {
		body["description"] = in.Description
	}
	if in.IsDefault != nil {
		body["isDefault"] = *in.IsDefault
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/userAddress", auth: authBearer, body: body})
}

// Update edits an address by ID. Send only ID + IsDefault to flip the default flag,
// or any other field to edit it (zero-valued strings are omitted).
func (s *AddressesService) Update(ctx context.Context, in AddressUpdateInput) (json.RawMessage, error) {
	body := map[string]any{"id": in.ID}
	if in.Title != "" {
		body["title"] = in.Title
	}
	if in.Description != "" {
		body["description"] = in.Description
	}
	if in.CityID != nil {
		body["cityId"] = in.CityID
	}
	if in.DistrictID != nil {
		body["districtId"] = in.DistrictID
	}
	if in.Address != "" {
		body["address"] = in.Address
	}
	if in.LocationLat != "" {
		body["locationLat"] = in.LocationLat
	}
	if in.LocationLng != "" {
		body["locationLng"] = in.LocationLng
	}
	if in.IsDefault != nil {
		body["isDefault"] = *in.IsDefault
	}
	return s.t.do(ctx, request{method: http.MethodPut, path: "/patients/userAddress", auth: authBearer, body: body})
}

// Delete removes an address by id (sent in the body). The default address cannot be
// deleted (reassign the default via Update first), nor can an address used on an order.
func (s *AddressesService) Delete(ctx context.Context, id any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodDelete, path: "/patients/userAddress", auth: authBearer, body: map[string]any{"id": id}})
}
