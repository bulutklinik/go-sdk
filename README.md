# sdk-go — Bulutklinik partner API SDK for Go

Official Bulutklinik **partner** API SDK for Go. Standard-library only (no
dependencies), context-aware, concurrency-safe.

This is a single-persona SDK: every call runs on the company-scoped `/outher`
surface with the partner token issued for your integration. You act on the
patients of **your own company**, and the patient is named inline on each
request — there is no login and no session. See [`DESIGN.md`](./DESIGN.md) for
the full wire contract.

> **v1.0.0 is a breaking release.** The patient persona (login, registration,
> payments, AI analysis, address book) has been removed and the former
> `client.Partner.*` namespace was lifted to the client root. See
> [CHANGELOG.md](./CHANGELOG.md) and DESIGN.md §12 for the migration.

## Install

```bash
go get github.com/bulutklinik/go-sdk
```

## Quick start

```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	bk "github.com/bulutklinik/go-sdk"
)

func main() {
	client, err := bk.NewClient(
		bk.WithEnvironment(bk.Production), // Production | Test | Local
		bk.WithAPIVersion(bk.V3),          // V3 (default) | V4
		bk.WithPartnerToken(os.Getenv("BK_PARTNER_TOKEN")),
	)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	// 1) Find a doctor you can book
	raw, err := client.Doctors.Search(ctx,
		map[string]any{"withFreeText": "kardiyoloji"}, 1, []string{"slot"})
	if err != nil {
		log.Fatal(err)
	}
	var found struct {
		FoundDoctors []struct {
			DoctorID int `json:"doctor_id"`
		} `json:"foundDoctors"`
	}
	_ = json.Unmarshal(raw, &found)
	doctorID := found.FoundDoctors[0].DoctorID

	// 2) Free slots
	schedule, err := client.Slots.Schedule(ctx, bk.ScheduleInput{
		DoctorID:     doctorID,
		ScheduleDate: "2026-08-01",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 3) Hold one for a patient — named inline, no session
	held, err := client.Appointments.ReserveWithoutAgreement(ctx, slotID, doctorID, bk.Patient{
		Name:        "Ada",
		Surname:     "Lovelace",
		PhoneNumber: "+905551112233",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 4) Confirm before the hold's reservationExpired passes
	_, err = client.Appointments.Create(ctx, hash, outherProcessID)
	_ = schedule
	_ = held
}
```

Every method returns the unwrapped `data` payload as a `json.RawMessage` —
unmarshal it into your own type.

## Services

28 endpoints across six groups.

| Group                  | Methods |
|------------------------|---------|
| `client.Doctors`       | `Search`, `Branches`, `Detail`, `Locations` |
| `client.Slots`         | `Schedule` |
| `client.Appointments`  | `Reserve`, `ReserveWithoutAgreement`, `InstantReserve`, `Create`, `CreateWithoutSlot`, `CancelWithoutSlot`, `List`, `Info`, `CheckDoctor` |
| `client.Measures`      | `Last`, `List`, `Graph`, `AddList`, `Add`, `Update`, `Delete`, `HealthInformation` |
| `client.Laboratory`    | `Catalog`, `CatalogDetail`, `Results`, `ResultDetail` |
| `client.Diets`         | `List`, `Detail` |

## Naming a patient

There is no session, so every patient-scoped call carries the patient in its
body — never in the URL, since a TCKN in a path segment would land in access
logs, proxy logs and error breadcrumbs.

**Reads** need only the reference fields. The server looks solely inside your own
company and never creates anything:

```go
client.Measures.Last(ctx, bk.Patient{IdentityNumber: "12345678901"})
client.Diets.List(ctx, bk.Patient{PhoneNumber: "+905551112233"}, nil)
```

`IdentityNumber` is primary; `PhoneNumber` is a fallback accepted only when it
matches exactly one patient (the column is not unique — family members share
numbers). A patient you have never treated resolves to "not found", with the same
message as "not yours" so the endpoint cannot be used to probe for TCKNs.

**Writes** need `Name`, `Surname` and `PhoneNumber` too, because the patient is
created inside your company if absent:

```go
client.Measures.AddList(ctx,
	bk.Patient{Name: "Ada", Surname: "Lovelace", PhoneNumber: "+905551112233"},
	[]map[string]any{{"type": "pulse", "date_time": "2026-06-17 09:31", "pulse": 72}},
)
```

## Booking

Two flows, depending on who collects the agreements and the payment:

```go
// (A) Hand off to the patient — returns a browser url for agreements + payment.
held, err := client.Appointments.Reserve(ctx, slotID, doctorID, user)

// (B) You already collected them — returns a hash to confirm yourself.
held, err := client.Appointments.ReserveWithoutAgreement(ctx, slotID, doctorID, user)
_, err = client.Appointments.Create(ctx, hash, outherProcessID)
```

**Payment is never taken through the API.** No partner endpoint produces a
financial record; the browser hand-off in (A) is where payment happens. The SDK
returns the url verbatim and never opens or follows it.

`CreateWithoutSlot` books a free-form range outside the slot grid, for
integrations running their own calendar; `CancelWithoutSlot` reverses it — and
only it.

## Authentication

The partner token is **issued out of band** through the Bulutklinik Developer
Platform. It behaves like an API key: there is no login method, and the SDK
cannot renew it.

The token is read from a `TokenStore` on **every** request, so a long-running
process can pick up a newly issued one without being rebuilt:

```go
type VaultStore struct{ /* … */ }

func (s *VaultStore) Token() string        { /* … */ }
func (s *VaultStore) SetToken(t string)    { /* … */ }
func (s *VaultStore) Clear()               { /* … */ }

client, err := bk.NewClient(bk.WithTokenStore(&VaultStore{}))

// …or rotate the default in-memory store in place:
client.TokenStore().SetToken(newlyIssuedToken)
```

Pass `WithPartnerToken` **or** `WithTokenStore`, not both — `NewClient` returns
`ErrCredentialConflict` rather than guessing which one you meant.

### When the token expires

Tokens last about 30 days. An expired one comes back as `401` / `resultType 4`;
the SDK returns an `ErrAuthentication` failure and does **not** retry — there is
nothing to refresh. Recovery is operational: obtain a newly issued token and
write it into the store.

> This is the one behaviour that changed meaning in v1.0.0. On the patient SDK
> `resultType 4` meant "the SDK will fix this silently". Here it means the opposite.

An `ErrAuthorization` (403) means the credential itself is wrong — either the
token lacks the `apiouther` scope, or it resolves to a user with no company. The
company boundary comes from the token, never from request input, so retrying with
different body parameters will not help.

## Health measures

```go
ref := bk.Patient{IdentityNumber: "12345678901"}

// Write several measurements at once (max 200 per call, one transaction)
client.Measures.AddList(ctx, patient, []map[string]any{
	{"type": "tension", "date_time": "2026-06-17 09:30", "hypertension": 120, "hypotension": 80},
	{"type": "glucose", "date_time": "2026-06-17 09:35", "glucose": 95, "glucose_type": 0},
})

client.Measures.Last(ctx, ref)
client.Measures.List(ctx, ref, "glucose", 1, &glucoseType) // 0=fasting, 1=postprandial
client.Measures.Graph(ctx, ref, "tension", 2, nil, nil)    // period 2 = weekly
```

> Measurements are written to **your own company**. A value you write does not
> appear in the patient's Bulutklinik mobile app, and values they entered there
> are not visible to you. That is tenant isolation working as intended.

`Measures.HealthInformation` is the legacy `teusan` bulk endpoint, kept for
existing integrations: it needs the `teusan` scope instead of `apiouther`, takes
a flat identity + phone number instead of a patient object, and writes into the
shared consumer tenant. The API currently matches on phone number only (a
server-side bug nulls `identity` during validation); pass both for forward
compatibility. Prefer `AddList` for anything new.

## Escape hatch

Not every endpoint has a typed method. `client.Do` reuses the same transport, so
headers, envelope unwrapping and typed errors all still apply:

```go
data, err := client.Do(ctx, "GET", "/outher/somethingNew", nil)

// "public" reaches unauthenticated endpoints outside the partner surface,
// e.g. the city/district catalogue that feeds address forms.
cfg, err := client.Do(ctx, "GET", "/general/getConfig", &bk.RequestOptions{Auth: "public"})
```

## Errors

Match with `errors.Is` against the sentinels and inspect with `errors.As`:

`ErrTransport` (network) · `ErrAPI` → `ErrValidation` (422), `ErrAuthentication`
(401 / revoked / expired), `ErrAuthorization` (403), `ErrNotFound` (404),
`ErrRateLimit` (429).

```go
_, err := client.Measures.Last(ctx, ref)
switch {
case errors.Is(err, bk.ErrRateLimit):
	var apiErr *bk.APIError
	errors.As(err, &apiErr)
	log.Printf("retry after %v", apiErr.RetryAfter)
case errors.Is(err, bk.ErrValidation):
	log.Printf("invalid request")
}
```

Note that `/outher` reports most business-rule failures as HTTP **`501`** with
`resultType 1` — "patient not found in your company", "slot no longer free",
"doctor not bookable through your integration". It is not a server crash; read
the message.

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

A read-only live smoke test doubles as a full example:

```bash
BK_PARTNER_TOKEN=... go run ./examples/livecheck
```

## License

MIT
