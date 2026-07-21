package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// AppointmentsService covers reservation, physical appointment and cancellation.
type AppointmentsService struct{ t *transport }

// ReserveInterview reserves an online (interview) slot. appointmentType defaults
// to "interview" when empty.
func (s *AppointmentsService) ReserveInterview(ctx context.Context, doctorID any, appointmentDate, appointmentType string) (json.RawMessage, error) {
	if appointmentType == "" {
		appointmentType = "interview"
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/addInterviewDateReservation", auth: authBearer, body: map[string]any{
		"doctorId":        doctorID,
		"appointmentDate": appointmentDate,
		"appointmentType": appointmentType,
	}})
}

// AddPhysical creates a physical appointment.
func (s *AppointmentsService) AddPhysical(ctx context.Context, doctorID any, appointmentDate string) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/addNewAppointment", auth: authBearer, body: map[string]any{
		"doctorId":        doctorID,
		"appointmentDate": appointmentDate,
	}})
}

// Cancel cancels an appointment by event id (cln_events.id).
func (s *AppointmentsService) Cancel(ctx context.Context, eventID any) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/deleteUserAppointment/%v", eventID)
	return s.t.do(ctx, request{method: http.MethodDelete, path: path, auth: authBearer})
}

// List returns the patient's appointments ({foundAppointmentsCount, foundAppointments}).
// Each item's event_id is the id for Cancel; rows with event_id "0" are paid-order/refund
// entries (not cancellable). Server paging is disabled, so page <= 1 returns the full list;
// pass 0 to omit the page segment.
func (s *AppointmentsService) List(ctx context.Context, page int) (json.RawMessage, error) {
	path := "/patients/userAppointments"
	if page > 0 {
		path = fmt.Sprintf("/patients/userAppointments/%d", page)
	}
	return s.t.do(ctx, request{method: http.MethodGet, path: path, auth: authBearer})
}

// Reservations returns the patient's active online-slot reservation holds
// (with a minute_diff/second_diff countdown).
func (s *AppointmentsService) Reservations(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/patients/userReservations", auth: authBearer})
}
