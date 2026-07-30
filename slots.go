package bulutklinik

import (
	"context"
	"encoding/json"
	"net/http"
)

// SlotsService covers doctor availability.
type SlotsService struct{ t *transport }

// Schedule returns bookable slots for a doctor as a date-keyed map. slotId feeds
// [AppointmentsService.Reserve]; an appointmentDate elsewhere is the date key
// plus slotStart with the seconds dropped ("Y-m-d H:i").
func (s *SlotsService) Schedule(ctx context.Context, in ScheduleInput) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/doctorSlots", auth: authPartner, body: in})
}
