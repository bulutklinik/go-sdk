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
	return s.t.do(ctx, request{http.MethodPost, "/patients/addInterviewDateReservation", authBearer, map[string]any{
		"doctorId":        doctorID,
		"appointmentDate": appointmentDate,
		"appointmentType": appointmentType,
	}})
}

// AddPhysical creates a physical appointment.
func (s *AppointmentsService) AddPhysical(ctx context.Context, doctorID any, appointmentDate string) (json.RawMessage, error) {
	return s.t.do(ctx, request{http.MethodPost, "/patients/addNewAppointment", authBearer, map[string]any{
		"doctorId":        doctorID,
		"appointmentDate": appointmentDate,
	}})
}

// Cancel cancels an appointment by event id (cln_events.id).
func (s *AppointmentsService) Cancel(ctx context.Context, eventID any) (json.RawMessage, error) {
	path := fmt.Sprintf("/patients/deleteUserAppointment/%v", eventID)
	return s.t.do(ctx, request{http.MethodDelete, path, authBearer, nil})
}
