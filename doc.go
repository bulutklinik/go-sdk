// Package bulutklinik is the official Bulutklinik partner API SDK for Go.
//
// It is a single-persona SDK: every call runs on the company-scoped /outher
// surface with the partner token issued for your integration. You act on the
// patients of your own company, and the patient is named inline on each request —
// there is no login and no session.
//
// Construct a client with [NewClient] and use the service fields (Doctors,
// Slots, Appointments, Measures, Laboratory, Diets).
//
// Every method takes a context.Context and returns the decoded "data" payload as
// a json.RawMessage (unmarshal it into your own type) plus an error. Errors are
// matched with errors.Is against the package's sentinel errors (for example
// [ErrNotFound], [ErrValidation]) and inspected with errors.As into [*APIError].
//
// The partner token is issued out of band through the Bulutklinik Developer
// Platform and cannot be renewed from here: an expired one (401 / resultType 4)
// surfaces as [ErrAuthentication] with no retry. Recovery is operational — write
// a newly issued token into the [TokenStore].
//
// Note that /outher reports most business-rule failures as HTTP 501 with
// resultType 1 ("patient not found in your company", "slot no longer free"). It
// is not a server crash; read the message.
package bulutklinik
