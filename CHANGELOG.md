# Changelog

All notable changes to the Bulutklinik Go SDK (`github.com/bulutklinik/go-sdk`)
are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1]

Documentation and contract corrections found by auditing the SDKs against the
API source before release. No wire change.

### Fixed

- `doctors.search` no longer lets `searchParams` default to an empty map. The
  server rule is `required|array` and PHP's `required` rejects an empty array, so
  `{}` was a guaranteed `422` rather than an unfiltered search.
- Corrected the `measures.HealthInformation` note. The defect it described — the API
  nulling `identity` before the patient lookup — was fixed API-side on
  2026-07-21. What actually remains is looser and worth knowing: the lookup is
  `identity OR phoneNumber` against the global user table and takes the first
  row, so a phone number alone can resolve a person whose TCKN differs from the
  one you sent.

## [1.0.0]

The SDK becomes **partner-only**. Everything that required a patient login is
gone; the company-scoped `/outher` surface that shipped under `client.Partner`
in 0.6.0 is now the client root. See `DESIGN.md` §12 for the full migration.

### Changed — BREAKING

- **`client.Partner.<Group>` → `client.<Group>`.** The six partner groups
  (`Doctors`, `Slots`, `Appointments`, `Measures`, `Laboratory`, `Diets`) moved
  to the root. Their paths, bodies and behaviour are unchanged — this is a
  rename. Service types lost the `Partner` prefix (`PartnerDoctorsService` →
  `DoctorsService`); `PartnerService` is gone. `PartnerScheduleInput` →
  `ScheduleInput` and `PartnerAppointmentWithoutSlotInput` →
  `AppointmentWithoutSlotInput`.
- **`NewClient` now returns `(*Client, error)`.** It reports
  `ErrCredentialConflict` when both `WithPartnerToken` and `WithTokenStore` are
  supplied, rather than silently picking one.
- **`TokenStore` now holds one partner token**: `Token()` / `SetToken(string)` /
  `Clear()` replace `AccessToken()` / `RefreshToken()` / `SetTokens()`.
  `NewInMemoryTokenStore` takes a single token argument.
- **`WithPartnerToken` is now the client's credential** and is required for every
  call.
- **No silent refresh.** A `401` / `resultType 4` returns an `ErrAuthentication`
  failure with no retry — a partner token is issued out of band and cannot be
  renewed from here. Install a newly issued token in the token store instead.
- **A missing token fails before dispatch** with `ErrAuthentication`, rather than
  sending an anonymous request that returns an opaque `401`.
- **`RequestOptions.Auth` defaults to `"partner"`**; the `"bearer"` mode no
  longer exists. `"public"` remains, for unauthenticated endpoints outside the
  surface.
- `Doctors.Search` now takes `(searchParams map[string]any, currentPage int,
  orderParams []string)` instead of a `SearchInput` struct — the `/outher` search
  has no `otherParams` or `perPageLimit`, and `orderParams` excludes `point`.
- `Measures.PartnerHealthInformation` → `Measures.HealthInformation`.

### Added

- **`APIVersion` (`V3` / `V4`) and `WithAPIVersion`.** Every path is
  version-agnostic, so targeting v4 is configuration, not a code change. Default
  stays `V3`.
- `ErrCredentialConflict` sentinel.

### Removed

- `client.Auth` (all 11 methods), `client.Payments` (5), `client.Skin`,
  `client.Meals`, `client.Addresses` (4) — no company-scoped equivalent exists.
- The patient-persona `Doctors` / `Slots` / `Appointments` / `Measures` /
  `Laboratory` / `Diets` that lived at the root in 0.6.0.
- `WithCredentials` (OAuth client id/secret).
- The `LoginResult`, `ConnectInput`, `RegisterInput`, `VerifyRegistrationInput`,
  `ConfirmRegistrationEmailInput`, `VerifyRegistrationSocialInput`,
  `RegisterSocialInput`, `ForgotPasswordInput`, `ResetPasswordInput`,
  `AddressInput`, `AddressUpdateInput`, `SearchInput`, `DiscountInput`,
  `CardInfo`, `PaymentInput`, `MealInput` and `LabOrderInput` types.

## [0.6.0]

### Added

- `client.Auth.ConfirmRegistrationEmail(ctx, in)` — the **required** e-mail-branch middle
  step of registration (`POST /patients/emailConfirmationRegister`). A headerless SDK
  caller always gets `confirmationType "email"` from `VerifyRegistration`; confirm the
  e-mailed code here to receive the SMS blob that `Register` consumes (without it,
  `Register` returns 501).
- Social sign-up: `client.Auth.VerifyRegistrationSocial(ctx, in)` +
  `client.Auth.RegisterSocial(ctx, in)` (both public; `RegisterSocial` does not
  auto-login — call `Connect` with LoginMode `social` after).
- Password reset: `client.Auth.ForgotPassword(ctx, in)` + `client.Auth.ResetPassword(ctx, in)`.
- `client.Appointments.List(ctx, page)` (`GET /patients/userAppointments`) — the source of the
  `event_id` that `Cancel` requires — and `client.Appointments.Reservations(ctx)`.
- New `client.Addresses` service (`List`/`Add`/`Update`/`Delete`) over `/patients/userAddress`,
  required by `Laboratory.Order` (which needs an `addressId`).
- Types: `ConfirmRegistrationEmailInput`, `VerifyRegistrationSocialInput`,
  `RegisterSocialInput`, `ForgotPasswordInput`, `ResetPasswordInput`, `AddressInput`,
  `AddressUpdateInput`.

## [0.5.0]

### Added

- `client.Auth.VerifyRegistration(ctx, in)` — step 1 of registration
  (`POST /patients/verifyAddingNewPatient`): sends the verification code and returns
  the raw data holding the encrypted `response` blob to pass to `Register`. Uses the
  configured partner token (`auth:apiusers`, not public) and requires a
  browser-minted CAPTCHA token (`RecaptchaV2` or `Captcha`).
- Type: `VerifyRegistrationInput`.

## [0.4.0]

### Added

- `client.Laboratory` — the patient's laboratory results, the orderable test
  catalog and test pre-ordering (DESIGN.md §6.9): `Results` (`GET
  /patients/userLabTestList/{page?}`), `ResultDetail` (`GET
  /patients/userLabTestDetail/{testId}`, string id), `Catalog` (`GET
  /patients/allLaboratoryTests`), `CatalogDetail` (`GET
  /patients/laboratoryTestDetail/{id}`) and `Order` (`POST
  /patients/addNewLaboratoryTest`).
- `client.Diets` — the patient's diet lists (DESIGN.md §6.10): `List` (`GET
  /patients/dietLists/{page?}`) and `Detail` (`GET /patients/diet/{listId}`).
- Type: `LabOrderInput` (`TestID`, `AddressID`, `LaboratoryID`) for
  `Laboratory.Order`.

## [0.3.0]

### Added

- `client.Skin.Analyze(ctx, images)` — "Cildimde Neyim Var" AI skin-lesion
  analysis (`POST /patients/imageCheck`). Returns per-image lesion `label`, a
  Turkish AI `comment`, `confidence`, `possible_icd` and an opaque `case_detail`
  blob (which can be forwarded as a payment's `CaseDetail`).
- `client.Meals.Analyze(ctx, input)` — AI meal-photo calorie/nutrition
  estimation (`POST /patients/imageAnalyzeMeal`).
- Type: `MealInput` (`Image`, `PortionSize`, `MealType`, optional `PortionGrams`
  and `Note` pointers).

## [0.2.0]

### Added

- `client.Do(ctx, method, path, *RequestOptions)` escape hatch for calling any
  endpoint not yet covered by a typed resource method (DESIGN.md §7.2).

## [0.1.0]

### Added

- Initial release: `Auth`, `Doctors`, `Slots`, `Appointments`, `Payments`,
  `Measures` service groups over a shared transport with silent token refresh.
