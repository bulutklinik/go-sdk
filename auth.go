package bulutklinik

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// AuthService covers login, 2FA, token refresh, registration and logout.
type AuthService struct{ t *transport }

// Connect logs in. On success tokens are stored automatically and the result has
// TwoFactorRequired == false. When 2FA is enabled the result carries
// TwoFactorResponse for [AuthService.ConnectWithTwoFactor].
func (s *AuthService) Connect(ctx context.Context, in ConnectInput) (*LoginResult, error) {
	clientID := in.ClientID
	if clientID == "" {
		clientID = s.t.clientID
	}
	clientSecret := in.ClientSecret
	if clientSecret == "" {
		clientSecret = s.t.clientSecret
	}

	body := map[string]any{
		"apiUserName":     in.APIUserName,
		"apiUserPassword": in.APIUserPassword,
		"apiClientId":     clientID,
		"apiSecretKey":    clientSecret,
		"loginMode":       in.LoginMode,
	}
	if in.WithPhoneNumber != "" {
		body["withPhoneNumber"] = in.WithPhoneNumber
	}

	data, err := s.t.do(ctx, request{method: http.MethodPost, path: "/general/connectApi", auth: authPublic, body: body})
	if err != nil {
		return nil, err
	}

	var d loginData
	if len(data) > 0 {
		_ = json.Unmarshal(data, &d)
	}
	if d.AccessToken != "" {
		s.t.tokenStore.SetTokens(d.AccessToken, d.RefreshToken)
		return &LoginResult{TwoFactorRequired: false}, nil
	}
	if d.Response != "" {
		return &LoginResult{TwoFactorRequired: true, TwoFactorResponse: d.Response}, nil
	}
	return &LoginResult{TwoFactorRequired: false}, nil
}

// ConnectWithTwoFactor completes a 2FA login with the SMS code and challenge blob.
func (s *AuthService) ConnectWithTwoFactor(ctx context.Context, smsVerificationCode, response string) error {
	data, err := s.t.do(ctx, request{method: http.MethodPost, path: "/general/connectApiWithTwoFactor", auth: authPublic, body: map[string]any{
		"smsVerificationCode": smsVerificationCode,
		"response":            response,
	}})
	if err != nil {
		return err
	}
	return s.storeTokens(data)
}

// VerifyRegistration is step 1 of registration: it sends the SMS/e-mail
// verification code and returns the raw data containing the encrypted "response"
// blob. It uses the configured partner token (the endpoint is behind
// auth:apiusers, not public). A CAPTCHA token (RecaptchaV2 or Captcha), minted by
// a browser/human, is required. Feed the returned "response" (and the code the
// user receives) into Register.
func (s *AuthService) VerifyRegistration(ctx context.Context, in VerifyRegistrationInput) (json.RawMessage, error) {
	accept := in.AcceptUserAgreement
	if accept == 0 {
		accept = 1
	}
	body := map[string]any{
		"name":                in.Name,
		"surname":             in.Surname,
		"phoneNumber":         in.PhoneNumber,
		"phone_code":          in.PhoneCode,
		"email":               in.Email,
		"password":            in.Password,
		"passwordAgain":       in.Password,
		"acceptUserAgreement": accept,
	}
	if in.RecaptchaV2 != "" {
		body["g-recaptcha-response-v2"] = in.RecaptchaV2
	}
	if in.Captcha != "" {
		body["captcha"] = in.Captcha
	}
	if in.UserAgreements != nil {
		body["userAgreements"] = in.UserAgreements
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/verifyAddingNewPatient", auth: authPartner, body: body})
}

// Register creates a new patient (afterRegister auto-login) and stores tokens.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) error {
	clientID := in.ClientID
	if clientID == "" {
		clientID = s.t.clientID
	}
	clientSecret := in.ClientSecret
	if clientSecret == "" {
		clientSecret = s.t.clientSecret
	}
	accept := in.AcceptUserAgreement
	if accept == 0 {
		accept = 1
	}

	data, err := s.t.do(ctx, request{method: http.MethodPost, path: "/patients/addNewPatient", auth: authPublic, body: map[string]any{
		"name":                in.Name,
		"surname":             in.Surname,
		"apiUserName":         in.APIUserName,
		"phoneNumber":         in.PhoneNumber,
		"password":            in.Password,
		"smsVerificationCode": in.SMSVerificationCode,
		"response":            in.Response,
		"acceptUserAgreement": accept,
		"apiClientId":         clientID,
		"apiSecretKey":        clientSecret,
	}})
	if err != nil {
		return err
	}
	return s.storeTokens(data)
}

// ConfirmRegistrationEmail is step 2 of the e-mail-branch registration. When
// VerifyRegistration returned confirmationType "email", confirm the e-mailed code
// here with the same response blob; the server sends an SMS code and returns a
// fresh response blob (confirmationType "sms") to feed into Register. Public.
func (s *AuthService) ConfirmRegistrationEmail(ctx context.Context, in ConfirmRegistrationEmailInput) (json.RawMessage, error) {
	body := map[string]any{
		"verificationCode": in.VerificationCode,
		"response":         in.Response,
	}
	if in.UserAgreements != nil {
		body["userAgreements"] = in.UserAgreements
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/emailConfirmationRegister", auth: authPublic, body: body})
}

// VerifyRegistrationSocial is step 1 of social sign-up: it sends the SMS code and
// returns the raw data holding the response blob. Public — no CAPTCHA and no
// partner token. Feed response + the SMS code into RegisterSocial.
func (s *AuthService) VerifyRegistrationSocial(ctx context.Context, in VerifyRegistrationSocialInput) (json.RawMessage, error) {
	accept := in.AcceptUserAgreement
	if accept == 0 {
		accept = 1
	}
	body := map[string]any{
		"name":                in.Name,
		"surname":             in.Surname,
		"phoneNumber":         in.PhoneNumber,
		"password":            in.Password,
		"passwordAgain":       in.Password,
		"socialType":          in.SocialType,
		"key":                 in.Key,
		"acceptUserAgreement": accept,
	}
	if in.Email != "" {
		body["email"] = in.Email
	}
	if in.UserAgreements != nil {
		body["userAgreements"] = in.UserAgreements
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/verifyAddingNewPatientSocial", auth: authPublic, body: body})
}

// RegisterSocial is step 2 of social sign-up: it creates the social patient. Unlike
// Register it does NOT log in — call Connect with LoginMode "social" afterwards. Public.
func (s *AuthService) RegisterSocial(ctx context.Context, in RegisterSocialInput) error {
	body := map[string]any{
		"smsVerificationCode": in.SMSVerificationCode,
		"response":            in.Response,
	}
	if in.UserAgreements != nil {
		body["userAgreements"] = in.UserAgreements
	}
	_, err := s.t.do(ctx, request{method: http.MethodPost, path: "/patients/addNewPatientWithSocial", auth: authPublic, body: body})
	return err
}

// ForgotPassword is step 1 of password reset: it sends the SMS confirm code to a
// registered phone and returns the raw data holding the response blob. A CAPTCHA
// token (RecaptchaV2 or Captcha) is required outside the local environment. Public.
func (s *AuthService) ForgotPassword(ctx context.Context, in ForgotPasswordInput) (json.RawMessage, error) {
	body := map[string]any{"phoneNumber": in.PhoneNumber}
	if in.Birthdate != "" {
		body["birthdate"] = in.Birthdate
	}
	if in.RecaptchaV2 != "" {
		body["g-recaptcha-response-v2"] = in.RecaptchaV2
	}
	if in.Captcha != "" {
		body["captcha"] = in.Captcha
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/forgotPassword", auth: authPublic, body: body})
}

// ResetPassword is step 2 of password reset: it sets the new password using the SMS
// confirm code and the response blob from ForgotPassword. Public.
func (s *AuthService) ResetPassword(ctx context.Context, in ResetPasswordInput) error {
	_, err := s.t.do(ctx, request{method: http.MethodPut, path: "/patients/forgotPassword", auth: authPublic, body: map[string]any{
		"smsConfirmCode": in.SMSConfirmCode,
		"response":       in.Response,
		"password":       in.Password,
		"passwordAgain":  in.Password,
	}})
	return err
}

// Refresh manually refreshes the access token using the stored refresh token.
func (s *AuthService) Refresh(ctx context.Context) error { return s.t.refresh(ctx) }

// Disconnect revokes the current tokens server-side and clears the token store.
func (s *AuthService) Disconnect(ctx context.Context) error {
	_, err := s.t.do(ctx, request{method: http.MethodPost, path: "/general/disconnectApi", auth: authBearer, body: map[string]any{}})
	s.t.tokenStore.Clear()
	return err
}

type loginData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Response     string `json:"response"`
}

func (s *AuthService) storeTokens(data json.RawMessage) error {
	var d loginData
	if err := json.Unmarshal(data, &d); err != nil || d.AccessToken == "" {
		return errors.New("bulutklinik: login response did not contain an access token")
	}
	s.t.tokenStore.SetTokens(d.AccessToken, d.RefreshToken)
	return nil
}
