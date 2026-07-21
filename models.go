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
