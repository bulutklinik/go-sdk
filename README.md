# sdk-go — Bulutklinik API SDK for Go

Official Bulutklinik API SDK for Go. Standard-library only (no dependencies),
context-aware, concurrency-safe.

Covers the patient flow: **auth, doctor search, slots, appointments, payments,
health measures, AI image analysis, laboratory results, and diet lists**. See
[`DESIGN.md`](./DESIGN.md) for the full wire contract.

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
	"fmt"

	bk "github.com/bulutklinik/go-sdk"
)

func main() {
	client := bk.NewClient(
		bk.WithEnvironment(bk.Production), // Production | Test | Local
		bk.WithCredentials("clientID", "clientSecret"),
	)
	ctx := context.Background()

	// 1) Log in (tokens are stored automatically)
	login, err := client.Auth.Connect(ctx, bk.ConnectInput{
		APIUserName:     "patient@example.com",
		APIUserPassword: "•••••••",
		LoginMode:       "email",
	})
	if err != nil {
		panic(err)
	}
	if login.TwoFactorRequired {
		_ = client.Auth.ConnectWithTwoFactor(ctx, "123456", login.TwoFactorResponse)
	}

	// 2) Find a doctor; decode the data into your own type
	raw, err := client.Doctors.Search(ctx, bk.SearchInput{
		SearchParams: map[string]any{"withFreeText": "kardiyoloji"},
		OrderParams:  []string{"slot"},
		OtherParams:  []string{"isInterviewable"},
	})
	if err != nil {
		panic(err)
	}
	var result struct {
		FoundDoctors []struct {
			DoctorID int `json:"doctor_id"`
		} `json:"foundDoctors"`
	}
	_ = json.Unmarshal(raw, &result)
	fmt.Println(result.FoundDoctors)
}
```

Every data method returns `json.RawMessage` — unmarshal it into your own structs.

## Services

| Field                  | Methods |
|------------------------|---------|
| `client.Auth`          | `Connect`, `ConnectWithTwoFactor`, `VerifyRegistration`, `Register`, `Refresh`, `Disconnect` |
| `client.Doctors`       | `Branches`, `Locations`, `QuickSearch`, `Search`, `Detail` |
| `client.Slots`         | `Schedule` |
| `client.Appointments`  | `ReserveInterview`, `AddPhysical`, `Cancel` |
| `client.Payments`      | `CheckDiscountCode`, `GetCards`, `SaveCard`, `Pay`, `DeleteCard` |
| `client.Measures`      | `AddList`, `Add`, `Update`, `Delete`, `Last`, `List`, `Graph`, `PartnerHealthInformation` |
| `client.Skin`          | `Analyze` |
| `client.Meals`         | `Analyze` |
| `client.Laboratory`    | `Results`, `ResultDetail`, `Catalog`, `CatalogDetail`, `Order` |
| `client.Diets`         | `List`, `Detail` |

## Authentication & tokens

- `Connect` / `ConnectWithTwoFactor` / `Register` store tokens automatically.
- On a `401` (or `resultType 4`), the SDK silently refreshes once and retries.
  Refresh is concurrency-safe (a single refresh is shared across goroutines).
- Inject a custom store with `bk.WithTokenStore(...)` (implement `bk.TokenStore`).

## Errors

Match failures with `errors.Is` against the sentinels, and inspect with
`errors.As`:

```go
_, err := client.Payments.Pay(ctx, in)
switch {
case errors.Is(err, bk.ErrRateLimit):
	var apiErr *bk.APIError
	errors.As(err, &apiErr)
	fmt.Println("retry after", apiErr.RetryAfter)
case errors.Is(err, bk.ErrValidation):
	// ...
case errors.Is(err, bk.ErrTransport):
	// network failure
}
```

Sentinels: `ErrTransport`, `ErrAPI`, `ErrValidation`, `ErrAuthentication`,
`ErrAuthorization`, `ErrNotFound`, `ErrRateLimit`. Every API failure also matches
`ErrAPI`.

## Payments (3-D Secure)

`Payments.Pay` returns data containing `payment3DUrl` on a 3DS flow — a browser
URL to open. The bank → server callback completes the capture; the SDK never
follows the URL.

## AI image analysis

```go
// "Cildimde Neyim Var" — analyze one or more skin photos (base64).
// Each image is a loose record; branch_id is optional.
raw, err := client.Skin.Analyze(ctx, []map[string]any{
	{"image": base64Jpeg, "branch_id": 42},
})
// data.status[i].case_detail can be forwarded verbatim as a payment's CaseDetail.

// Meal photo → calorie/nutrition estimate.
grams := 300
raw, err = client.Meals.Analyze(ctx, bk.MealInput{
	Image:        base64Jpeg,
	PortionSize:  "medium", // small | medium | large | custom
	MealType:     "lunch",  // breakfast | lunch | dinner | snack
	PortionGrams: &grams,   // required when PortionSize is "custom"
	// Note:      &note,    // optional free text (≤1000 chars)
})
```

`PortionGrams` and `Note` are `*int` / `*string` — leave them nil to omit them
from the request. `Skin.Analyze` returns `{ status: [...] }`; `Meals.Analyze`
returns `{ status: { comment } }` where `comment` is a JSON-object string.

## Development

```bash
gofmt -l .
go vet ./...
go build ./...
go test ./...
```

## License

MIT
