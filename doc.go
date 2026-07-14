// Package bulutklinik is the official Bulutklinik API SDK for Go.
//
// It covers the patient flow: auth, doctor search, slots, appointments,
// payments, health measures and AI image analysis. Construct a client with
// [NewClient] and use the service fields (Auth, Doctors, Slots, Appointments,
// Payments, Measures, Skin, Meals).
//
// Every method takes a context.Context and returns the decoded "data" payload as
// a json.RawMessage (unmarshal it into your own type) plus an error. Errors are
// matched with errors.Is against the package's sentinel errors (for example
// [ErrNotFound], [ErrValidation]) and inspected with errors.As into [*APIError].
package bulutklinik
