package bulutklinik

import (
	"context"
	"encoding/json"
	"net/http"
)

// DietsService reads diet lists recorded for a patient inside your own company.
// Lists written by other clinics are not visible.
type DietsService struct{ t *transport }

// List returns paginated diet lists. Page size is fixed to 20 server-side.
func (s *DietsService) List(ctx context.Context, patient Patient, page any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/dietLists", auth: authPartner, body: map[string]any{
		"patient": patient, "currentPage": page,
	}})
}

// Detail returns the meal breakdown of one diet list. listID comes from List; one
// that is not this patient's fails with the same generic error as "not found".
func (s *DietsService) Detail(ctx context.Context, patient Patient, listID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/diet", auth: authPartner, body: map[string]any{
		"patient": patient, "listId": listID,
	}})
}
