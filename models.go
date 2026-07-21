package bulutklinik

// LoginResult is returned by [AuthService.Connect]. When TwoFactorRequired is
// true, pass TwoFactorResponse (with the SMS code) to
// [AuthService.ConnectWithTwoFactor].
type LoginResult struct {
	TwoFactorRequired bool
	TwoFactorResponse string
}

// ConnectInput holds the login parameters. ClientID/ClientSecret default to the
// client's configured credentials when empty.
type ConnectInput struct {
	APIUserName     string
	APIUserPassword string
	LoginMode       string
	ClientID        string
	ClientSecret    string
	WithPhoneNumber string
}

// RegisterInput holds new-patient registration parameters.
type RegisterInput struct {
	Name                string
	Surname             string
	APIUserName         string
	PhoneNumber         string
	Password            string
	SMSVerificationCode string
	Response            string
	AcceptUserAgreement int
	ClientID            string
	ClientSecret        string
}

// VerifyRegistrationInput holds the fields for the registration verify step
// (auth.VerifyRegistration). The endpoint requires a CAPTCHA token (RecaptchaV2 or
// Captcha) minted by a browser/human, and is authorized with the partner token.
type VerifyRegistrationInput struct {
	Name        string
	Surname     string
	PhoneNumber string
	// PhoneCode is the country dial code only, e.g. "+90" (matches ^\+\d{1,3}$).
	PhoneCode           string
	Email               string
	Password            string
	AcceptUserAgreement int
	// RecaptchaV2 is sent as "g-recaptcha-response-v2". Provide this or Captcha.
	RecaptchaV2 string
	// Captcha is sent as "captcha". Provide this or RecaptchaV2.
	Captcha string
	// UserAgreements is passed through verbatim when non-nil.
	UserAgreements []any
}

// ConfirmRegistrationEmailInput is step 2 of the e-mail-branch registration.
type ConfirmRegistrationEmailInput struct {
	VerificationCode string
	// Response is the blob from VerifyRegistration (when confirmationType was "email").
	Response       string
	UserAgreements []any
}

// VerifyRegistrationSocialInput is step 1 of social sign-up (public; no CAPTCHA/partner token).
type VerifyRegistrationSocialInput struct {
	Name        string
	Surname     string
	PhoneNumber string
	Password    string
	// SocialType is the provider identifier (e.g. "google", "apple").
	SocialType string
	// Key is the social provider key/token identifying the user.
	Key                 string
	Email               string
	AcceptUserAgreement int
	UserAgreements      []any
}

// RegisterSocialInput is step 2 of social sign-up. It does NOT mint tokens.
type RegisterSocialInput struct {
	SMSVerificationCode string
	// Response is the blob from VerifyRegistrationSocial.
	Response       string
	UserAgreements []any
}

// ForgotPasswordInput is step 1 of password reset.
type ForgotPasswordInput struct {
	PhoneNumber string
	// Birthdate is optional "YYYY-MM-DD"; required by installs that verify identity.
	Birthdate string
	// RecaptchaV2 is sent as "g-recaptcha-response-v2". Set this or Captcha (outside local env).
	RecaptchaV2 string
	Captcha     string
}

// ResetPasswordInput is step 2 of password reset.
type ResetPasswordInput struct {
	SMSConfirmCode string
	// Response is the blob from ForgotPassword.
	Response string
	Password string
}

// AddressInput creates a patient address (addresses.Add).
type AddressInput struct {
	Title       string
	Description  string
	CityID      any // from Doctors.Locations (location_id)
	DistrictID  any // from GET /getConfig (cities[].districts[])
	Address     string
	LocationLat string
	LocationLng string
	// IsDefault: 1 makes it the default. Nil omits the field (first address is default anyway).
	IsDefault *int
}

// AddressUpdateInput updates a patient address by ID (addresses.Update).
// Zero-valued string fields are omitted; send only what you change (or just ID+IsDefault).
type AddressUpdateInput struct {
	ID          any
	Title       string
	Description  string
	CityID      any
	DistrictID  any
	Address     string
	LocationLat string
	LocationLng string
	IsDefault   *int
}

// SearchInput holds filtered doctor search parameters.
type SearchInput struct {
	SearchParams map[string]any
	OrderParams  []string
	OtherParams  []string
	CurrentPage  int
	PerPageLimit int
}

// ScheduleInput holds doctor scheduler parameters.
type ScheduleInput struct {
	DoctorID     any
	ListType     string
	ScheduleDate string
	ScheduleStep any
	SchedulePage any
}

// DiscountInput holds discount-code check parameters.
type DiscountInput struct {
	CheckType        string
	DiscountCode     string
	DoctorID         any
	OrderID          any
	SpecialServiceID any
	ProgramSlug      string
}

// CardInfo holds plain card fields for saveCard and inline payment.
type CardInfo struct {
	CardHolder   string `json:"cardHolder"`
	CardNumber   string `json:"cardNumber"`
	CardExpMonth string `json:"cardExpMonth"`
	CardExpYear  string `json:"cardExpYear"`
	CardCvv      string `json:"cardCvv"`
}

// PaymentInput holds appointment payment parameters. Provide either CardInfo
// (a new card) or CardID (a saved card). AppointmentType defaults to "interview".
type PaymentInput struct {
	DoctorID        any
	AppointmentDate string
	Is3D            bool
	TermsAccept     bool
	AppointmentType string
	CardInfo        *CardInfo
	CardID          any
	SaveCard        int
	DiscountCode    string
	CaseDetail      string
}

// MealInput holds parameters for [MealsService.Analyze]. Image and MealType are
// required. PortionSize is one of "small", "medium", "large" or "custom".
// PortionGrams and Note are optional pointers so they are omitted from the
// request body when nil; PortionGrams is required when PortionSize == "custom".
type MealInput struct {
	Image        string
	PortionSize  string
	MealType     string
	PortionGrams *int
	Note         *string
}

// LabOrderInput holds parameters for [LaboratoryService.Order]. All three ids
// are required and serialize to the request body keys testId/addressId/
// laboratoryId.
type LabOrderInput struct {
	TestID       string
	AddressID    string
	LaboratoryID string
}
