package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// MeasuresService covers health measurements.
//
// Scope: measurements are written into and read from your own company. Values
// the patient entered in the Bulutklinik mobile app are not visible here, and a
// value you write does not appear in their app — a consequence of tenant
// isolation, not a bug.
//
// Writes take the descriptive [Patient] shape (created if absent); reads and
// edits need only the reference fields.
type MeasuresService struct{ t *transport }

// Last returns the most recent value of every measurement type.
func (s *MeasuresService) Last(ctx context.Context, patient Patient) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/lastMeasures", auth: authPartner, body: map[string]any{"patient": patient}})
}

// List returns the paginated history of one measurement type. glucoseType
// applies to "glucose" only (0=fasting, 1=postprandial).
func (s *MeasuresService) List(ctx context.Context, patient Patient, measureType string, page any, glucoseType *int) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measuresList/%s", measureType)
	return s.t.do(ctx, request{method: http.MethodPost, path: path, auth: authPartner, body: map[string]any{
		"patient": patient, "currentPage": page, "glucoseType": glucoseType,
	}})
}

// Graph returns a time-bucketed series. period: 1=day, 2=week, 3=month, 4=year.
func (s *MeasuresService) Graph(ctx context.Context, patient Patient, measureType string, period int, page any, glucoseType *int) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measuresGraph/%s/%d", measureType, period)
	return s.t.do(ctx, request{method: http.MethodPost, path: path, auth: authPartner, body: map[string]any{
		"patient": patient, "currentPage": page, "glucoseType": glucoseType,
	}})
}

// AddList writes several measurements of mixed types in one transaction. The
// server caps a single call at 200 rows.
func (s *MeasuresService) AddList(ctx context.Context, patient Patient, data []map[string]any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/measures", auth: authPartner, body: map[string]any{
		"patient": patient, "data": data,
	}})
}

// Add writes a single measurement. fields carry date_time plus the type's own
// columns and are flattened alongside patient, matching the server shape.
func (s *MeasuresService) Add(ctx context.Context, patient Patient, measureType string, fields map[string]any) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measure/%s", measureType)
	return s.t.do(ctx, request{method: http.MethodPost, path: path, auth: authPartner, body: withPatient(patient, fields, nil)})
}

// Update edits one measurement row. id comes from List.
func (s *MeasuresService) Update(ctx context.Context, patient Patient, measureType string, id any, fields map[string]any) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measure/%s", measureType)
	return s.t.do(ctx, request{method: http.MethodPut, path: path, auth: authPartner, body: withPatient(patient, fields, id)})
}

// Delete removes one measurement row.
func (s *MeasuresService) Delete(ctx context.Context, patient Patient, measureType string, id any) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measure/%s", measureType)
	return s.t.do(ctx, request{method: http.MethodDelete, path: path, auth: authPartner, body: withPatient(patient, nil, id)})
}

// HealthInformation is the legacy bulk submission for teusan integrations.
//
// Deprecated: it requires the teusan scope instead of apiouther, takes a flat
// identity + phoneNumber instead of a patient object, and writes into the shared
// consumer tenant rather than your own company — so the values are not readable
// through Last or List. Prefer AddList.
func (s *MeasuresService) HealthInformation(ctx context.Context, identity, phoneNumber string, data []map[string]any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/healthInformation", auth: authPartner, body: map[string]any{
		"identity": strOrNil(identity), "phoneNumber": strOrNil(phoneNumber), "data": data,
	}})
}
