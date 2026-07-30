package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// LaboratoryService covers the laboratory catalogue (global, static) and results
// (your company only, merging HBYS lab requests and TmcLab order groups).
//
// Ordering a test is not available here — it creates a financial record.
type LaboratoryService struct{ t *transport }

// Catalog returns the orderable test packages. Static catalogue, no patient context.
func (s *LaboratoryService) Catalog(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/outher/laboratoryCatalog", auth: authPartner})
}

// CatalogDetail returns one catalogue package. Prices are the plain list prices —
// the patient-side discount pass does not apply here.
func (s *LaboratoryService) CatalogDetail(ctx context.Context, testID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: fmt.Sprintf("/outher/laboratoryCatalog/%v", testID), auth: authPartner})
}

// Results returns paginated laboratory results. Each item's id is accepted
// verbatim by ResultDetail; a "-lab" suffix marks a TmcLab order group.
func (s *LaboratoryService) Results(ctx context.Context, patient Patient, page any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/laboratoryResults", auth: authPartner, body: map[string]any{
		"patient": patient, "currentPage": page,
	}})
}

// ResultDetail returns one result. Pass the id from Results unchanged — it is
// sent as a string so a "-lab" suffix survives the round trip.
func (s *LaboratoryService) ResultDetail(ctx context.Context, patient Patient, testID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/laboratoryResult", auth: authPartner, body: map[string]any{
		"patient": patient, "testId": fmt.Sprintf("%v", testID),
	}})
}
