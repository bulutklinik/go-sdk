package bulutklinik

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// ErrMissingClientCredentials is returned by [AuthService.Connect] when neither
// the call nor the client carries a client id and secret.
var ErrMissingClientCredentials = errors.New("bulutklinik: clientId and clientSecret are required")

// LoginResult is the outcome of [AuthService.Connect].
//
// When TwoFactorRequired is true no tokens were stored and TwoFactorResponse
// carries the server's challenge blob.
type LoginResult struct {
	TwoFactorRequired bool
	TwoFactorResponse string
	PasswordPolicy    json.RawMessage
}

// ConnectInput carries the credentials a portal application issues. ClientID and
// ClientSecret fall back to the values the client was built with.
type ConnectInput struct {
	// APIUserName is the project-specific service identity, not an e-mail you chose.
	APIUserName string
	// APIUserPassword is the password set when registering on the portal.
	APIUserPassword string
	ClientID        string
	ClientSecret    string
	// LoginMode defaults to "email".
	LoginMode string
}

// AuthService covers the token lifecycle.
//
// The Developer Platform issues a client id, a client secret and a
// project-specific service identity per approved application. Connect exchanges
// those for an access + refresh token pair, which every other service then uses.
// Connect and Refresh are the only endpoints here that are not
// partner-authenticated: they are what produce the credential.
type AuthService struct{ t *transport }

// Connect logs in and stores the resulting tokens.
//
// If the account has SMS 2FA enabled the API returns a challenge instead of a
// token pair; the result reports TwoFactorRequired rather than erroring.
func (s *AuthService) Connect(ctx context.Context, in ConnectInput) (LoginResult, error) {
	clientID, clientSecret := in.ClientID, in.ClientSecret
	if clientID == "" {
		clientID = s.t.clientID
	}
	if clientSecret == "" {
		clientSecret = s.t.clientSecret
	}
	if clientID == "" || clientSecret == "" {
		return LoginResult{}, ErrMissingClientCredentials
	}
	loginMode := in.LoginMode
	if loginMode == "" {
		loginMode = "email"
	}

	raw, err := s.t.do(ctx, request{method: http.MethodPost, path: "/general/connectApi", auth: authPublic, body: map[string]any{
		"apiClientId":     clientID,
		"apiSecretKey":    clientSecret,
		"apiUserName":     in.APIUserName,
		"apiUserPassword": in.APIUserPassword,
		"loginMode":       loginMode,
	}})
	if err != nil {
		return LoginResult{}, err
	}

	var payload struct {
		AccessToken    string          `json:"access_token"`
		RefreshToken   string          `json:"refresh_token"`
		Response       string          `json:"response"`
		PasswordPolicy json.RawMessage `json:"password_policy"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return LoginResult{}, &TransportError{Message: "bulutklinik: decode connectApi response", Err: err}
	}

	if payload.AccessToken != "" {
		s.t.setTokens(payload.AccessToken, payload.RefreshToken)
		return LoginResult{PasswordPolicy: payload.PasswordPolicy}, nil
	}
	return LoginResult{TwoFactorRequired: true, TwoFactorResponse: payload.Response}, nil
}

// Refresh rotates both tokens. The transport already does this automatically on
// a 401 / resultType 4, so calling it by hand only refreshes ahead of time.
func (s *AuthService) Refresh(ctx context.Context) error {
	return s.t.refresh(ctx)
}

// Disconnect revokes the access token and all of its refresh tokens, then clears
// the store.
//
// Sent with an empty body on purpose: the endpoint also accepts a device-token
// cleanup whose device mapping has no default branch server-side, and there is no
// partner use for it.
func (s *AuthService) Disconnect(ctx context.Context) error {
	if _, err := s.t.do(ctx, request{
		method: http.MethodPost,
		path:   "/general/disconnectApi",
		auth:   authPartner,
		body:   map[string]any{},
	}); err != nil {
		return err
	}
	s.t.clearTokens()
	return nil
}
