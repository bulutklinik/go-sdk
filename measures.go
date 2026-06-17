package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// MeasuresService covers health measurement CRUD, listing, graph and the partner
// submission endpoint.
type MeasuresService struct{ t *transport }

// AddList submits multiple measurements of any types in one call.
func (s *MeasuresService) AddList(ctx context.Context, records []map[string]any) (json.RawMessage, error) {
	return s.t.do(ctx, request{http.MethodPost, "/patients/addNewUserMeasures", authBearer, map[string]any{"data": records}})
}

// Add submits a single measurement of one type (fields include date_time).
func (s *MeasuresService) Add(ctx context.Context, measureType string, fields map[string]any) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/addNewUserMeasures/%s", measureType)
	return s.t.do(ctx, request{http.MethodPost, path, authBearer, fields})
}

// Update edits a measurement (fields include id and date_time).
func (s *MeasuresService) Update(ctx context.Context, measureType string, fields map[string]any) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/updateUserMeasures/%s", measureType)
	return s.t.do(ctx, request{http.MethodPut, path, authBearer, fields})
}

// Delete removes a measurement by id.
func (s *MeasuresService) Delete(ctx context.Context, measureType string, id any) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/deleteUserMeasures/%s", measureType)
	return s.t.do(ctx, request{http.MethodDelete, path, authBearer, map[string]any{"id": id}})
}

// Last returns the latest value of each measurement type.
func (s *MeasuresService) Last(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{http.MethodGet, "/patients/measuresList", authBearer, nil})
}

// List returns the paginated history for one type. glucoseType applies only to
// glucose; pass nil to omit it.
func (s *MeasuresService) List(ctx context.Context, measureType string, page any, glucoseType *int) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/userMeasuresList/%s/%v", measureType, page)
	if glucoseType != nil {
		path += fmt.Sprintf("/%d", *glucoseType)
	}
	return s.t.do(ctx, request{http.MethodGet, path, authBearer, nil})
}

// Graph returns grouped graph data. period: 1=day, 2=week, 3=month, 4=year.
func (s *MeasuresService) Graph(ctx context.Context, measureType string, period int, page any, glucoseType *int) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/userMeasuresGraph/%s/%d/%v", measureType, period, page)
	if glucoseType != nil {
		path += fmt.Sprintf("/%d", *glucoseType)
	}
	return s.t.do(ctx, request{http.MethodGet, path, authBearer, nil})
}

// PartnerHealthInformation submits measurements as a partner (teusan) using the
// configured partner token.
func (s *MeasuresService) PartnerHealthInformation(ctx context.Context, identity, phoneNumber string, data []map[string]any) (json.RawMessage, error) {
	return s.t.do(ctx, request{http.MethodPost, "/outher/healthInformation", authPartner, map[string]any{
		"identity":    strOrNil(identity),
		"phoneNumber": strOrNil(phoneNumber),
		"data":        data,
	}})
}
