# Changelog

All notable changes to the Bulutklinik Go SDK (`github.com/bulutklinik/go-sdk`)
are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
