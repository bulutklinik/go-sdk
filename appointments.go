package bulutklinik

import (
	"context"
	"encoding/json"
	"net/http"
)

// AppointmentsService covers the appointment lifecycle. The patient is supplied
// inline as user; the server materialises it inside your company on write.
//
// Two booking flows:
//
//   - Hand off to the patient: [AppointmentsService.Reserve] returns a url the
//     patient opens in a browser to accept the agreements and pay.
//   - You collected the agreements: [AppointmentsService.ReserveWithoutAgreement]
//     returns a hash; feed it plus outherProcessId into [AppointmentsService.Create].
//
// Payment is never taken through the API. No partner endpoint produces a
// financial record; that is what the browser hand-off is for.
type AppointmentsService struct{ t *transport }

// Reserve holds an online slot for the given patient and returns a url for the
// patient to complete agreements and payment in a browser.
func (s *AppointmentsService) Reserve(ctx context.Context, slotID, doctorID any, user Patient) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/reservation", auth: authPartner, body: map[string]any{
		"slotId": slotID, "doctorId": doctorID, "user": user,
	}})
}

// ReserveWithoutAgreement is Reserve for integrations that collect the
// agreements themselves. It returns hash, doctorId, slotId, phoneNumber and
// reservationExpired — confirm with Create before the hold expires.
func (s *AppointmentsService) ReserveWithoutAgreement(ctx context.Context, slotID, doctorID any, user Patient) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/reservationWithoutAgreement", auth: authPartner, body: map[string]any{
		"slotId": slotID, "doctorId": doctorID, "user": user,
	}})
}

// InstantReserve creates an instant reservation — no slot; the server picks an
// available doctor.
func (s *AppointmentsService) InstantReserve(ctx context.Context, user Patient) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/instantReservation", auth: authPartner, body: map[string]any{"user": user}})
}

// Create turns a reservation into a confirmed appointment. Both arguments come
// from the reservation response. It returns the appointment plus
// last_delete_time, the cancellation deadline.
func (s *AppointmentsService) Create(ctx context.Context, hash string, outherProcessID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/appointment", auth: authPartner, body: map[string]any{
		"hash": hash, "outherProcessId": outherProcessID,
	}})
}

// CreateWithoutSlot books a free-form time range outside the slot grid.
func (s *AppointmentsService) CreateWithoutSlot(ctx context.Context, in AppointmentWithoutSlotInput) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/appointmentWithoutSlot", auth: authPartner, body: in})
}

// CancelWithoutSlot cancels an appointment created with CreateWithoutSlot — and
// only those; appointments confirmed through Create are not cancellable here.
func (s *AppointmentsService) CancelWithoutSlot(ctx context.Context, lookup AppointmentLookup) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodDelete, path: "/outher/appointmentWithoutSlot", auth: authPartner, body: lookup})
}

// List returns the appointments you created for the given phone number, not the
// patient's history across the platform. appointmentType is "normal" or
// "instant"; empty sends null.
func (s *AppointmentsService) List(ctx context.Context, phoneNumber string, page any, appointmentType string) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/appointments", auth: authPartner, body: map[string]any{
		"phoneNumber": phoneNumber, "page": page, "type": strOrNil(appointmentType),
	}})
}

// Info returns a single appointment, addressed by process or by coordinates.
func (s *AppointmentsService) Info(ctx context.Context, lookup AppointmentLookup) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/appointmentInfo", auth: authPartner, body: lookup})
}

// CheckDoctor reports whether a doctor is bookable through your integration. It
// fails with 501 when they are not — call it before offering a doctor.
func (s *AppointmentsService) CheckDoctor(ctx context.Context, doctorID any, isOutherDoctor int) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/checkDoctor", auth: authPartner, body: map[string]any{
		"doctorId": doctorID, "isOutherDoctor": isOutherDoctor,
	}})
}
