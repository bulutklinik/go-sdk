package bulutklinik

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Environment selects a base URL preset.
type Environment string

const (
	Production Environment = "production"
	Test       Environment = "test"
	Local      Environment = "local"
)

var baseURLs = map[Environment]string{
	Production: "https://api.bulutklinik.com/api/v3",
	Test:       "https://apitest.bulutklinik.com/api/v3",
	Local:      "https://api-bulutklinik.test/api/v3",
}

// Client is the Bulutklinik API client. Create it with [NewClient] and use the
// service fields. A Client is safe for concurrent use.
type Client struct {
	Auth         *AuthService
	Doctors      *DoctorsService
	Slots        *SlotsService
	Appointments *AppointmentsService
	Payments     *PaymentsService
	Measures     *MeasuresService
	Skin         *SkinService
	Meals        *MealsService
	Laboratory   *LaboratoryService
	Diets        *DietsService

	transport *transport
}

type options struct {
	environment  Environment
	baseURL      string
	lang         string
	clientID     string
	clientSecret string
	partnerToken string
	tokenStore   TokenStore
	httpClient   *http.Client
	timeout      time.Duration
}

// Option configures a [Client].
type Option func(*options)

// WithEnvironment selects a base URL preset (default [Production]).
func WithEnvironment(env Environment) Option { return func(o *options) { o.environment = env } }

// WithBaseURL overrides the base URL (takes precedence over WithEnvironment).
func WithBaseURL(u string) Option { return func(o *options) { o.baseURL = u } }

// WithLang sets the default lang header (default "tr").
func WithLang(lang string) Option { return func(o *options) { o.lang = lang } }

// WithCredentials sets the OAuth client id/secret (used for login and refresh).
func WithCredentials(clientID, clientSecret string) Option {
	return func(o *options) { o.clientID = clientID; o.clientSecret = clientSecret }
}

// WithPartnerToken sets the bearer token for the partner (teusan) endpoint.
func WithPartnerToken(token string) Option { return func(o *options) { o.partnerToken = token } }

// WithTokenStore injects a custom token store (default in-memory).
func WithTokenStore(ts TokenStore) Option { return func(o *options) { o.tokenStore = ts } }

// WithHTTPClient injects a custom *http.Client.
func WithHTTPClient(c *http.Client) Option { return func(o *options) { o.httpClient = c } }

// WithTimeout sets the request timeout when no custom client is provided.
func WithTimeout(d time.Duration) Option { return func(o *options) { o.timeout = d } }

// NewClient builds a client. With no options it targets production with an
// in-memory token store and a 30s timeout.
func NewClient(opts ...Option) *Client {
	o := options{environment: Production, lang: "tr"}
	for _, opt := range opts {
		opt(&o)
	}

	base := o.baseURL
	if base == "" {
		base = baseURLs[o.environment]
	}
	base = strings.TrimRight(base, "/")

	store := o.tokenStore
	if store == nil {
		store = NewInMemoryTokenStore("", "")
	}

	httpClient := o.httpClient
	if httpClient == nil {
		timeout := o.timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	tr := &transport{
		httpClient:   httpClient,
		baseURL:      base,
		lang:         o.lang,
		clientID:     o.clientID,
		clientSecret: o.clientSecret,
		partnerToken: o.partnerToken,
		tokenStore:   store,
	}

	c := &Client{transport: tr}
	c.Auth = &AuthService{tr}
	c.Doctors = &DoctorsService{tr}
	c.Slots = &SlotsService{tr}
	c.Appointments = &AppointmentsService{tr}
	c.Payments = &PaymentsService{tr}
	c.Measures = &MeasuresService{tr}
	c.Skin = &SkinService{tr}
	c.Meals = &MealsService{tr}
	c.Laboratory = &LaboratoryService{tr}
	c.Diets = &DietsService{tr}
	return c
}

// TokenStore returns the active token store.
func (c *Client) TokenStore() TokenStore { return c.transport.tokenStore }

// RequestOptions configures a [Client.Do] call. The zero value (or a nil
// *RequestOptions) means a bearer-authenticated request with no body and the
// client's default lang.
type RequestOptions struct {
	// Auth selects the authentication mode: "public", "bearer" or "partner".
	// Empty defaults to "bearer".
	Auth string
	// Body is an optional JSON payload (any value that encodes with
	// encoding/json). It is ignored on GET requests.
	Body any
	// Lang optionally overrides the client's default lang header for this
	// request. Empty uses the client default.
	Lang string
}

// Do is an escape hatch for calling any Bulutklinik API endpoint that does not
// yet have a typed resource method. The request goes through the same transport
// as the typed methods, so default headers, the chosen auth mode (bearer by
// default), silent token refresh + retry, response-envelope unwrapping and the
// typed error hierarchy all apply.
//
// method is one of GET, POST, PUT, DELETE; path is relative to the configured
// base URL with a leading slash (for example "/patients/allBranches"). Pass nil
// opts for a bearer GET/DELETE with no body. It returns the unwrapped "data"
// payload as a json.RawMessage (unmarshal it into your own type) plus an error,
// exactly like the typed resource methods.
//
// Prefer a typed resource method when one exists; reach for Do only for the
// endpoints the SDK does not cover yet.
func (c *Client) Do(ctx context.Context, method, path string, opts *RequestOptions) (json.RawMessage, error) {
	r := request{method: method, path: path, auth: authBearer}
	if opts != nil {
		r.auth = authModeFromString(opts.Auth)
		r.body = opts.Body
		r.lang = opts.Lang
	}
	return c.transport.do(ctx, r)
}

// authModeFromString maps the public string auth labels to the internal
// authMode. Unknown or empty values default to bearer.
func authModeFromString(s string) authMode {
	switch strings.ToLower(s) {
	case "public":
		return authPublic
	case "partner":
		return authPartner
	default:
		return authBearer
	}
}
