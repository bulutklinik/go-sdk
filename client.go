package bulutklinik

import (
	"context"
	"encoding/json"
	"errors"
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

// APIVersion selects the API version segment. The /outher surface is
// route-for-route identical on both, so switching is configuration rather than a
// code change.
type APIVersion string

const (
	V3 APIVersion = "v3"
	V4 APIVersion = "v4"
)

// apiRoots hold the version-less API root per environment. The base URL is
// root + "/" + APIVersion.
var apiRoots = map[Environment]string{
	Production: "https://api.bulutklinik.com/api",
	Test:       "https://apitest.bulutklinik.com/api",
	Local:      "https://api-bulutklinik.test/api",
}

// ErrCredentialConflict is returned by [NewClient] when both WithPartnerToken
// and WithTokenStore are given. Either the literal or the store is the source of
// truth for the credential, and guessing which one the caller meant is how
// credential bugs get shipped.
var ErrCredentialConflict = errors.New("bulutklinik: pass either WithPartnerToken or WithTokenStore, not both")

// Client is the Bulutklinik partner API client. Create it with [NewClient] and
// use the service fields. A Client is safe for concurrent use.
//
// Every call runs on the company-scoped /outher surface with the partner token
// issued for your integration: you act on the patients of your own company, and
// the patient is named inline on each request — there is no login and no session.
type Client struct {
	// Doctors covers discovery: search, branches, detail, city list.
	Doctors *DoctorsService
	// Slots covers doctor availability (materialized slots).
	Slots *SlotsService
	// Appointments covers reserve, confirm, free-form booking, cancel, list, lookup.
	Appointments *AppointmentsService
	// Measures covers health measurements for a named patient, read and write.
	Measures *MeasuresService
	// Laboratory covers lab results for a named patient plus the test catalog.
	Laboratory *LaboratoryService
	// Diets covers diet lists written by a dietitian, for a named patient.
	Diets *DietsService

	transport *transport
}

type options struct {
	environment  Environment
	apiVersion   APIVersion
	baseURL      string
	lang         string
	partnerToken string
	tokenStore   TokenStore
	httpClient   *http.Client
	timeout      time.Duration
}

// Option configures a [Client].
type Option func(*options)

// WithEnvironment selects a base URL preset (default [Production]).
func WithEnvironment(env Environment) Option { return func(o *options) { o.environment = env } }

// WithAPIVersion selects the API version segment (default [V3]).
func WithAPIVersion(v APIVersion) Option { return func(o *options) { o.apiVersion = v } }

// WithBaseURL overrides the base URL (takes precedence over WithEnvironment and
// WithAPIVersion).
func WithBaseURL(u string) Option { return func(o *options) { o.baseURL = u } }

// WithLang sets the default lang header (default "tr").
func WithLang(lang string) Option { return func(o *options) { o.lang = lang } }

// WithPartnerToken sets the partner token issued for your integration. It seeds
// the default in-memory token store. Mutually exclusive with [WithTokenStore].
func WithPartnerToken(token string) Option { return func(o *options) { o.partnerToken = token } }

// WithTokenStore injects a custom token store, read on every request so a
// long-running process can rotate the credential without being rebuilt. Mutually
// exclusive with [WithPartnerToken].
func WithTokenStore(ts TokenStore) Option { return func(o *options) { o.tokenStore = ts } }

// WithHTTPClient injects a custom *http.Client.
func WithHTTPClient(c *http.Client) Option { return func(o *options) { o.httpClient = c } }

// WithTimeout sets the request timeout when no custom client is provided.
func WithTimeout(d time.Duration) Option { return func(o *options) { o.timeout = d } }

// NewClient builds a client. With no options it targets production v3 with an
// in-memory token store and a 30s timeout.
//
// It returns [ErrCredentialConflict] when both WithPartnerToken and
// WithTokenStore are supplied.
func NewClient(opts ...Option) (*Client, error) {
	o := options{environment: Production, apiVersion: V3, lang: "tr"}
	for _, opt := range opts {
		opt(&o)
	}

	if o.partnerToken != "" && o.tokenStore != nil {
		return nil, ErrCredentialConflict
	}

	base := o.baseURL
	if base == "" {
		base = apiRoots[o.environment] + "/" + string(o.apiVersion)
	}
	base = strings.TrimRight(base, "/")

	store := o.tokenStore
	if store == nil {
		store = NewInMemoryTokenStore(o.partnerToken)
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
		httpClient: httpClient,
		baseURL:    base,
		lang:       o.lang,
		tokenStore: store,
	}

	c := &Client{transport: tr}
	c.Doctors = &DoctorsService{tr}
	c.Slots = &SlotsService{tr}
	c.Appointments = &AppointmentsService{tr}
	c.Measures = &MeasuresService{tr}
	c.Laboratory = &LaboratoryService{tr}
	c.Diets = &DietsService{tr}
	return c, nil
}

// TokenStore returns the active token store. Write a newly issued partner token
// here to rotate the credential without rebuilding the client.
func (c *Client) TokenStore() TokenStore { return c.transport.tokenStore }

// RequestOptions configures a [Client.Do] call. The zero value (or a nil
// *RequestOptions) means a partner-authenticated request with no body and the
// client's default lang.
type RequestOptions struct {
	// Auth selects the authentication mode: "partner" or "public".
	// Empty defaults to "partner".
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
// as the typed methods, so default headers, the chosen auth mode (partner by
// default), response-envelope unwrapping and the typed error hierarchy all apply.
//
// method is one of GET, POST, PUT, DELETE; path is relative to the configured
// base URL with a leading slash (for example "/outher/branches"). Pass nil opts
// for a partner GET/DELETE with no body. It returns the unwrapped "data" payload
// as a json.RawMessage (unmarshal it into your own type) plus an error, exactly
// like the typed resource methods.
//
// Auth "public" reaches the handful of unauthenticated endpoints outside the
// partner surface, for example GET /general/getConfig for the city/district list.
//
// Prefer a typed resource method when one exists; reach for Do only for the
// endpoints the SDK does not cover yet.
func (c *Client) Do(ctx context.Context, method, path string, opts *RequestOptions) (json.RawMessage, error) {
	r := request{method: method, path: path, auth: authPartner}
	if opts != nil {
		r.auth = authModeFromString(opts.Auth)
		r.body = opts.Body
		r.lang = opts.Lang
	}
	return c.transport.do(ctx, r)
}

// authModeFromString maps the public string auth labels to the internal
// authMode. Unknown or empty values default to partner.
func authModeFromString(s string) authMode {
	if strings.EqualFold(s, "public") {
		return authPublic
	}
	return authPartner
}
