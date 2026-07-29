package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Patient identifies a patient on the partner surface.
//
// Reads need only IdentityNumber (primary) or PhoneNumber (accepted solely when
// it matches exactly one patient in your company — the column is not unique, and
// the server fails closed rather than guessing).
//
// Writes need Name, Surname and PhoneNumber as well: if no matching patient
// exists in your company the server creates one.
type Patient struct {
	Name           string  `json:"name,omitempty"`
	Surname        string  `json:"surname,omitempty"`
	PhoneNumber    string  `json:"phoneNumber,omitempty"`
	IdentityNumber string  `json:"identityNumber,omitempty"`
	Email          string  `json:"email,omitempty"`
	Birthdate      string  `json:"birthdate,omitempty"`
	Nationality    string  `json:"nationality,omitempty"`
	Price          float64 `json:"price,omitempty"`
}

// AppointmentLookup addresses one appointment either by its process
// (Hash + OutherProcessID) or by its coordinates (DoctorID + AppointmentDate +
// IsOutherDoctor). Supply one pair or the other.
type AppointmentLookup struct {
	Hash            string `json:"hash,omitempty"`
	OutherProcessID any    `json:"outherProcessId,omitempty"`
	DoctorID        any    `json:"doctorId,omitempty"`
	AppointmentDate string `json:"appointmentDate,omitempty"`
	IsOutherDoctor  *int   `json:"isOutherDoctor,omitempty"`
}

// PartnerService is the company-scoped partner surface (/outher), reachable as
// client.Partner.
//
// It is a second persona, not a replacement for the patient one:
//
//	                        patient (client.*)          partner (client.Partner.*)
//	auth                    patient login, access token pre-issued partner token
//	data belongs to         the patient, all clinics    your own company only
//	how a patient is named  implicit (the session)      inline, per request
//
// Requests here use the configured partner token; no patient login is involved
// and the silent access-token refresh does not apply.
//
// Patient login/registration, the card vault, 3-D Secure payment, self-service
// AI and address CRUD have no partner equivalent and stay on the patient surface
// by design.
type PartnerService struct {
	Doctors      *PartnerDoctorsService
	Slots        *PartnerSlotsService
	Appointments *PartnerAppointmentsService
	Diets        *PartnerDietsService
	Laboratory   *PartnerLaboratoryService
	Measures     *PartnerMeasuresService
}

func newPartnerService(t *transport) *PartnerService {
	return &PartnerService{
		Doctors:      &PartnerDoctorsService{t},
		Slots:        &PartnerSlotsService{t},
		Appointments: &PartnerAppointmentsService{t},
		Diets:        &PartnerDietsService{t},
		Laboratory:   &PartnerLaboratoryService{t},
		Measures:     &PartnerMeasuresService{t},
	}
}

// PartnerDoctorsService covers doctor discovery on the partner surface. Results
// are scoped to the doctors enabled for your integration, so a doctor returned
// here is one you can actually book.
type PartnerDoctorsService struct{ t *transport }

// Search runs a filtered doctor search.
func (s *PartnerDoctorsService) Search(ctx context.Context, searchParams map[string]any, currentPage int, orderParams []string) (json.RawMessage, error) {
	if orderParams == nil {
		orderParams = []string{}
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/search", auth: authPartner, body: map[string]any{
		"searchParams": searchParams,
		"orderParams":  orderParams,
		"currentPage":  currentPage,
	}})
}

// Branches lists the branches available through your integration.
func (s *PartnerDoctorsService) Branches(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/outher/branches", auth: authPartner})
}

// Detail returns a single doctor.
func (s *PartnerDoctorsService) Detail(ctx context.Context, doctorID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: fmt.Sprintf("/outher/doctorInfos/%v", doctorID), auth: authPartner})
}

// Locations returns the city list. Global catalogue — not scoped to your company.
func (s *PartnerDoctorsService) Locations(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/outher/locations", auth: authPartner})
}

// PartnerSlotsService covers doctor availability on the partner surface.
type PartnerSlotsService struct{ t *transport }

// PartnerScheduleInput selects a doctor's bookable slots. Either set
// ScheduleDate (Y-m-d), or page through with ScheduleStep + SchedulePage; the
// server requires one of the two forms.
type PartnerScheduleInput struct {
	DoctorID     any    `json:"doctorId"`
	ScheduleDate string `json:"scheduleDate,omitempty"`
	ScheduleStep *int   `json:"scheduleStep,omitempty"`
	SchedulePage *int   `json:"schedulePage,omitempty"`
}

// Schedule returns bookable slots for a doctor.
func (s *PartnerSlotsService) Schedule(ctx context.Context, in PartnerScheduleInput) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/doctorSlots", auth: authPartner, body: in})
}

// PartnerAppointmentsService covers the appointment lifecycle on the partner
// surface. The patient is supplied inline; the server materialises it inside
// your company on write.
//
// Payment is not taken through the API: Reserve returns a process that is
// settled through the hosted web checkout.
type PartnerAppointmentsService struct{ t *transport }

// Reserve holds an online slot for the given patient.
func (s *PartnerAppointmentsService) Reserve(ctx context.Context, slotID, doctorID any, user Patient) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/reservation", auth: authPartner, body: map[string]any{
		"slotId": slotID, "doctorId": doctorID, "user": user,
	}})
}

// ReserveWithoutAgreement is Reserve for integrations that collect the
// agreements themselves.
func (s *PartnerAppointmentsService) ReserveWithoutAgreement(ctx context.Context, slotID, doctorID any, user Patient) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/reservationWithoutAgreement", auth: authPartner, body: map[string]any{
		"slotId": slotID, "doctorId": doctorID, "user": user,
	}})
}

// InstantReserve creates an instant (no slot) reservation.
func (s *PartnerAppointmentsService) InstantReserve(ctx context.Context, user Patient) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/instantReservation", auth: authPartner, body: map[string]any{"user": user}})
}

// Create turns a reservation into a confirmed appointment.
func (s *PartnerAppointmentsService) Create(ctx context.Context, hash string, outherProcessID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/appointment", auth: authPartner, body: map[string]any{
		"hash": hash, "outherProcessId": outherProcessID,
	}})
}

// PartnerAppointmentWithoutSlotInput books a free-form time range without going
// through a slot. StartDate/FinishDate are Y-m-d H:i.
type PartnerAppointmentWithoutSlotInput struct {
	DoctorID       any     `json:"doctorId"`
	StartDate      string  `json:"startDate"`
	FinishDate     string  `json:"finishDate"`
	IsOutherDoctor *int    `json:"isOutherDoctor,omitempty"`
	User           Patient `json:"user"`
}

// CreateWithoutSlot books a free-form time range.
func (s *PartnerAppointmentsService) CreateWithoutSlot(ctx context.Context, in PartnerAppointmentWithoutSlotInput) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/appointmentWithoutSlot", auth: authPartner, body: in})
}

// CancelWithoutSlot cancels an appointment created with CreateWithoutSlot.
func (s *PartnerAppointmentsService) CancelWithoutSlot(ctx context.Context, lookup AppointmentLookup) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodDelete, path: "/outher/appointmentWithoutSlot", auth: authPartner, body: lookup})
}

// List returns the appointments you created for the given phone number.
func (s *PartnerAppointmentsService) List(ctx context.Context, phoneNumber string, page any, appointmentType string) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/appointments", auth: authPartner, body: map[string]any{
		"phoneNumber": phoneNumber, "page": page, "type": strOrNil(appointmentType),
	}})
}

// Info returns a single appointment, addressed by process or by coordinates.
func (s *PartnerAppointmentsService) Info(ctx context.Context, lookup AppointmentLookup) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/appointmentInfo", auth: authPartner, body: lookup})
}

// CheckDoctor reports whether a doctor is bookable through your integration.
func (s *PartnerAppointmentsService) CheckDoctor(ctx context.Context, doctorID any, isOutherDoctor int) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/checkDoctor", auth: authPartner, body: map[string]any{
		"doctorId": doctorID, "isOutherDoctor": isOutherDoctor,
	}})
}

// PartnerDietsService reads diet lists recorded for a patient inside your own
// company. Lists written by other clinics are not visible.
type PartnerDietsService struct{ t *transport }

// List returns paginated diet lists.
func (s *PartnerDietsService) List(ctx context.Context, patient Patient, page any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/dietLists", auth: authPartner, body: map[string]any{
		"patient": patient, "currentPage": page,
	}})
}

// Detail returns the meal breakdown of one diet list. listID comes from List.
func (s *PartnerDietsService) Detail(ctx context.Context, patient Patient, listID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/diet", auth: authPartner, body: map[string]any{
		"patient": patient, "listId": listID,
	}})
}

// PartnerLaboratoryService covers the laboratory catalogue (global, static) and
// results (your company only, merging HBYS lab requests and TmcLab order groups).
type PartnerLaboratoryService struct{ t *transport }

// Catalog returns the orderable test packages.
func (s *PartnerLaboratoryService) Catalog(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: "/outher/laboratoryCatalog", auth: authPartner})
}

// CatalogDetail returns one catalogue package. Prices are the plain list prices —
// the patient-side discount pass does not apply on the partner surface.
func (s *PartnerLaboratoryService) CatalogDetail(ctx context.Context, testID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodGet, path: fmt.Sprintf("/outher/laboratoryCatalog/%v", testID), auth: authPartner})
}

// Results returns paginated laboratory results. Each item's id is accepted
// verbatim by ResultDetail; a "-lab" suffix marks a TmcLab order group.
func (s *PartnerLaboratoryService) Results(ctx context.Context, patient Patient, page any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/laboratoryResults", auth: authPartner, body: map[string]any{
		"patient": patient, "currentPage": page,
	}})
}

// ResultDetail returns one result. Pass the id from Results unchanged — it is
// sent as a string so a "-lab" suffix survives the round trip.
func (s *PartnerLaboratoryService) ResultDetail(ctx context.Context, patient Patient, testID any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/laboratoryResult", auth: authPartner, body: map[string]any{
		"patient": patient, "testId": fmt.Sprintf("%v", testID),
	}})
}

// PartnerMeasuresService covers health measurements on the partner surface.
//
// Scope: measurements are written into and read from your own company. Values
// the patient entered in the Bulutklinik mobile app live in the consumer tenant
// and are not visible here — a consequence of tenant isolation, not a bug.
type PartnerMeasuresService struct{ t *transport }

// Last returns the most recent value of every measurement type.
func (s *PartnerMeasuresService) Last(ctx context.Context, patient Patient) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/lastMeasures", auth: authPartner, body: map[string]any{"patient": patient}})
}

// List returns the paginated history of one measurement type.
func (s *PartnerMeasuresService) List(ctx context.Context, patient Patient, measureType string, page any, glucoseType *int) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measuresList/%s", measureType)
	return s.t.do(ctx, request{method: http.MethodPost, path: path, auth: authPartner, body: map[string]any{
		"patient": patient, "currentPage": page, "glucoseType": glucoseType,
	}})
}

// Graph returns a time-bucketed series. period: 1=day, 2=week, 3=month, 4=year.
func (s *PartnerMeasuresService) Graph(ctx context.Context, patient Patient, measureType string, period int, page any, glucoseType *int) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measuresGraph/%s/%d", measureType, period)
	return s.t.do(ctx, request{method: http.MethodPost, path: path, auth: authPartner, body: map[string]any{
		"patient": patient, "currentPage": page, "glucoseType": glucoseType,
	}})
}

// AddList writes several measurements of mixed types in one transaction. The
// server caps a single call at 200 rows.
func (s *PartnerMeasuresService) AddList(ctx context.Context, patient Patient, data []map[string]any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/measures", auth: authPartner, body: map[string]any{
		"patient": patient, "data": data,
	}})
}

// Add writes a single measurement. fields carry date_time plus the type's own
// columns and are flattened alongside patient, matching the server shape.
func (s *PartnerMeasuresService) Add(ctx context.Context, patient Patient, measureType string, fields map[string]any) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measure/%s", measureType)
	return s.t.do(ctx, request{method: http.MethodPost, path: path, auth: authPartner, body: withPatient(patient, fields, nil)})
}

// Update edits one measurement row. id comes from List.
func (s *PartnerMeasuresService) Update(ctx context.Context, patient Patient, measureType string, id any, fields map[string]any) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measure/%s", measureType)
	return s.t.do(ctx, request{method: http.MethodPut, path: path, auth: authPartner, body: withPatient(patient, fields, id)})
}

// Delete removes one measurement row.
func (s *PartnerMeasuresService) Delete(ctx context.Context, patient Patient, measureType string, id any) (json.RawMessage, error) {
	path := fmt.Sprintf("/outher/measure/%s", measureType)
	return s.t.do(ctx, request{method: http.MethodDelete, path: path, auth: authPartner, body: withPatient(patient, nil, id)})
}

// HealthInformation is the legacy teusan bulk submission.
//
// Deprecated: it writes into the shared consumer tenant rather than your own
// company, so the values are not readable through Last or List. Prefer AddList.
// Kept for existing teusan integrations.
func (s *PartnerMeasuresService) HealthInformation(ctx context.Context, identity, phoneNumber string, data []map[string]any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/outher/healthInformation", auth: authPartner, body: map[string]any{
		"identity": strOrNil(identity), "phoneNumber": strOrNil(phoneNumber), "data": data,
	}})
}

// withPatient flattens measure fields next to the patient reference, optionally
// adding an id. The server expects the columns at the top level, not nested.
func withPatient(patient Patient, fields map[string]any, id any) map[string]any {
	body := map[string]any{"patient": patient}
	if id != nil {
		body["id"] = id
	}
	for k, v := range fields {
		body[k] = v
	}
	return body
}
