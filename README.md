# sdk-go — Bulutklinik partner API SDK for Go

Official Bulutklinik **partner** API SDK for Go. Standard-library only (no
dependencies), context-aware, concurrency-safe.

This is a single-persona SDK: every call runs on the company-scoped `/outher`
surface with the partner token issued for your integration. You act on the
patients of **your own company**, and the patient is named inline on each
request — there is no patient session. See [`DESIGN.md`](./DESIGN.md) for
the full wire contract.

> **v1.1.0 restores `client.Auth`.** v1.0.x wrongly assumed the partner token
> could only be issued out of band; it is in fact minted by `connectApi` from
> your portal credentials, and it is refreshable. Existing v1.0.x code that uses
> `WithPartnerToken` keeps working. See [CHANGELOG.md](./CHANGELOG.md).

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
		bk.WithCredentials(os.Getenv("BK_CLIENT_ID"), os.Getenv("BK_CLIENT_SECRET")),
	)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	// 0) Log in. Tokens are stored and refreshed for you.
	if _, err := client.Auth.Connect(ctx, bk.ConnectInput{
		APIUserName:     os.Getenv("BK_SERVICE_IDENTITY"),
		APIUserPassword: os.Getenv("BK_PASSWORD"),
	}); err != nil {
		log.Fatal(err)
	}

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
		PhoneNumber: "+90 5551112233",
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

31 endpoints across seven groups.

| Group                  | Methods |
|------------------------|---------|
| `client.Auth`          | `Connect`, `Refresh`, `Disconnect` |
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
client.Diets.List(ctx, bk.Patient{PhoneNumber: "+90 5551112233"}, nil)
```

`IdentityNumber` is primary; `PhoneNumber` is a fallback accepted only when it
matches exactly one patient (the column is not unique — family members share
numbers). A patient you have never treated resolves to "not found", with the same
message as "not yours" so the endpoint cannot be used to probe for TCKNs.

**Writes** need `Name`, `Surname` and `PhoneNumber` too, because the patient is
created inside your company if absent:

```go
client.Measures.AddList(ctx,
	bk.Patient{Name: "Ada", Surname: "Lovelace", PhoneNumber: "+90 5551112233"},
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

Your portal application issues four values: a **client ID**, a **client secret**,
a project-specific **service identity** and an **application password**. The
password belongs to this application only — it is not your portal account
password, and you can regenerate it in the portal if it leaks. `Auth.Connect` exchanges them for an access token and
a refresh token:

```go
client, err := bk.NewClient(bk.WithCredentials(clientID, clientSecret))

result, err := client.Auth.Connect(ctx, bk.ConnectInput{
	APIUserName:     "svc@your-app.bulutklinik",
	APIUserPassword: "your-portal-password",
	// LoginMode defaults to "email".
})
```

The granted scope comes from the credentials, not the request — a partner
application is provisioned with `apiouther`, which is what makes `/outher`
reachable. Already holding a token? Use `WithPartnerToken` and skip the login.

### Refresh

Access tokens last ~30 days, refresh tokens ~130. You do not normally call
`Refresh` yourself: on a `401` / `resultType 4` the SDK refreshes once and
retries the original request, and concurrent calls share one in-flight refresh.

```go
err := client.Auth.Refresh(ctx)     // only useful to refresh ahead of time
err = client.Auth.Disconnect(ctx)   // revokes both tokens and clears the store
```

If the refresh fails — or there is no refresh token because you supplied a bare
partner token — the call returns an `ErrAuthentication` failure and you should
`Auth.Connect` again.

### Token storage

Tokens are read from a `TokenStore` on **every** request, so a long-running
process can rotate them without being rebuilt. Implement `RefreshTokenStore` to
persist both:

```go
type VaultStore struct{ /* … */ }

func (s *VaultStore) Token() string              { /* … */ }
func (s *VaultStore) SetToken(t string)          { /* … */ }
func (s *VaultStore) RefreshToken() string       { /* … */ }
func (s *VaultStore) SetRefreshToken(t string)   { /* … */ }
func (s *VaultStore) Clear()                     { /* … */ }

client, err := bk.NewClient(bk.WithTokenStore(&VaultStore{}), bk.WithCredentials(id, secret))
```

The two refresh methods are **optional**. A plain `TokenStore` — the v1.0.x
shape, access token only — still works; the SDK then keeps the refresh token in
memory, so a process restart needs `Auth.Connect` rather than a refresh.

An `ErrAuthorization` (403) means the credential itself is wrong: either the
granted scope does not include `apiouther`, or the account has no company. The
company boundary comes from the token, never from request input.

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
shared consumer tenant. Its patient matching is an **OR**, and it is loose: the lookup is
`identity OR phoneNumber` against the *global* user table and takes the first
row, so a phone number alone can resolve someone whose TCKN differs from the one
you sent. Send both, but do not assume they are checked as a pair — the
`apiouther` reads above do the opposite, scoping to your company and failing
closed on ambiguity. Prefer `AddList` for anything new.

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
