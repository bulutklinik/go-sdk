package bulutklinik

// Patient identifies a patient.
//
// Reads need only IdentityNumber (primary) or PhoneNumber (accepted solely when
// it matches exactly one patient in your company — the column is not unique, and
// the server fails closed rather than guessing). The server looks only inside
// your own company on this path and never creates anything, so a patient you
// have never treated resolves to "not found" — with the same message as "not
// yours", so the endpoint cannot be used to probe for TCKNs.
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

// ScheduleInput selects a doctor's bookable slots. Either set ScheduleDate
// (Y-m-d), or page through with ScheduleStep + SchedulePage; the server requires
// one of the two forms.
type ScheduleInput struct {
	DoctorID     any    `json:"doctorId"`
	ScheduleDate string `json:"scheduleDate,omitempty"`
	ScheduleStep *int   `json:"scheduleStep,omitempty"`
	SchedulePage *int   `json:"schedulePage,omitempty"`
}

// AppointmentWithoutSlotInput books a free-form time range outside the slot
// grid, for integrations running their own calendar. StartDate/FinishDate are
// Y-m-d H:i.
type AppointmentWithoutSlotInput struct {
	DoctorID       any     `json:"doctorId"`
	StartDate      string  `json:"startDate"`
	FinishDate     string  `json:"finishDate"`
	IsOutherDoctor *int    `json:"isOutherDoctor,omitempty"`
	User           Patient `json:"user"`
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
