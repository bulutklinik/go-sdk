# Changelog

All notable changes to the Bulutklinik Go SDK (`github.com/bulutklinik/go-sdk`)
are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
