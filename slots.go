package bulutklinik

import (
	"context"
	"encoding/json"
	"net/http"
)

// SlotsService covers doctor availability.
type SlotsService struct{ t *transport }

// Schedule fetches a doctor's free slots (a date-keyed map). ScheduleStep and
// SchedulePage default to 7 and 1 when nil.
func (s *SlotsService) Schedule(ctx context.Context, in ScheduleInput) (json.RawMessage, error) {
	step := in.ScheduleStep
	if step == nil {
		step = 7
	}
	page := in.SchedulePage
	if page == nil {
		page = 1
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/doctorScheduler", auth: authBearer, body: map[string]any{
		"doctorId":     in.DoctorID,
		"scheduleDate": strOrNil(in.ScheduleDate),
		"scheduleStep": step,
		"schedulePage": page,
		"listType":     in.ListType,
	}})
}
