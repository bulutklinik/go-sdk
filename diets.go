package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DietsService covers the patient's diet lists (a dietitian's "Diyet Listesi").
type DietsService struct{ t *transport }

// List returns the patient's diet lists. Pass page == nil to omit the segment
// (the server defaults to page 1, fixed page size 10), mirroring the
// optional-segment idiom of [DoctorsService.Detail].
func (s *DietsService) List(ctx context.Context, page any) (json.RawMessage, error) {
	path := "/patients/dietLists"
	if page != nil {
		path += fmt.Sprintf("/%v", page)
	}
	return s.t.do(ctx, request{method: http.MethodGet, path: path, auth: authBearer})
}

// Detail returns one diet list by its list_id (from a [DietsService.List] item).
func (s *DietsService) Detail(ctx context.Context, listID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/diet/%s", listID)
	return s.t.do(ctx, request{method: http.MethodGet, path: path, auth: authBearer})
}
