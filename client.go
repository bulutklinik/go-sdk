package bulutklinik

import (
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
	return c
}

// TokenStore returns the active token store.
func (c *Client) TokenStore() TokenStore { return c.transport.tokenStore }
