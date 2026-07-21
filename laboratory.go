package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// LaboratoryService covers the patient's own laboratory results, the orderable
// test-group catalog and test pre-ordering.
type LaboratoryService struct{ t *transport }

// Results returns the patient's completed/in-progress lab results. Pass page ==
// nil to omit the segment (the server defaults to page 1), mirroring the
// optional-segment idiom of [DoctorsService.Detail].
func (s *LaboratoryService) Results(ctx context.Context, page any) (json.RawMessage, error) {
	path := "/patients/userLabTestList"
	if page != nil {
		path += fmt.Sprintf("/%v", page)
	}
	return s.t.do(ctx, request{method: http.MethodGet, path: path, auth: authBearer})
}

// ResultDetail returns one lab result. testID is a string: a plain id ("123")
// or a TMC-lab id ("123-lab"); it is interpolated verbatim.
func (s *LaboratoryService) ResultDetail(ctx context.Context, testID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/userLabTestDetail/%s", testID)
	return s.t.do(ctx, request{method: http.MethodGet, path: path, auth: authBearer})
}

// Catalog returns the orderable test-group catalog.
func (s *LaboratoryService) Catalog(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/patients/allLaboratoryTests", auth: authBearer})
}

// CatalogDetail returns the single matching catalog test group by id.
func (s *LaboratoryService) CatalogDetail(ctx context.Context, id string) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/laboratoryTestDetail/%s", id)
	return s.t.do(ctx, request{method: http.MethodGet, path: path, auth: authBearer})
}

// Order pre-orders a laboratory test. TestID, AddressID and LaboratoryID are all
// required and serialize to the body keys testId/addressId/laboratoryId. Success
// returns data { preOrderId }.
func (s *LaboratoryService) Order(ctx context.Context, input LabOrderInput) (json.RawMessage, error) {
	body := map[string]any{
		"testId":       input.TestID,
		"addressId":    input.AddressID,
		"laboratoryId": input.LaboratoryID,
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/addNewLaboratoryTest", auth: authBearer, body: body})
}
