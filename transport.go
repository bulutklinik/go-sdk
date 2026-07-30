package bulutklinik

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
)

type authMode int

const (
	authPublic authMode = iota
	authPartner
)

type request struct {
	method string
	path   string
	auth   authMode
	body   any
	// lang optionally overrides the client's default lang header for this one
	// request. Empty means "use the transport default".
	lang string
}

type envelope struct {
	ResultType   *int            `json:"resultType"`
	ErrorType    any             `json:"errorType"`
	ErrorMessage string          `json:"errorMessage"`
	Data         json.RawMessage `json:"data"`
}

// transport builds requests, unwraps the response envelope and maps failures to
// typed errors.
//
// On a 401 / resultType 4 it refreshes once and retries the original request;
// the error surfaces only when there is no refresh token or the refresh fails.
// Concurrent refreshes are serialised so simultaneous failures do not stampede.
type transport struct {
	httpClient   *http.Client
	baseURL      string
	lang         string
	clientID     string
	clientSecret string
	tokenStore   TokenStore
	refreshMu    sync.Mutex
	// fallbackRefresh holds the refresh token when the injected store cannot.
	fallbackMu      sync.RWMutex
	fallbackRefresh string
}

// setTokens persists a freshly minted pair.
func (t *transport) setTokens(access, refresh string) {
	t.tokenStore.SetToken(access)
	if rs, ok := t.tokenStore.(RefreshTokenStore); ok {
		rs.SetRefreshToken(refresh)
		return
	}
	t.fallbackMu.Lock()
	defer t.fallbackMu.Unlock()
	t.fallbackRefresh = refresh
}

func (t *transport) refreshToken() string {
	if rs, ok := t.tokenStore.(RefreshTokenStore); ok {
		return rs.RefreshToken()
	}
	t.fallbackMu.RLock()
	defer t.fallbackMu.RUnlock()
	return t.fallbackRefresh
}

func (t *transport) clearTokens() {
	t.fallbackMu.Lock()
	t.fallbackRefresh = ""
	t.fallbackMu.Unlock()
	t.tokenStore.Clear()
}

func (t *transport) do(ctx context.Context, r request) (json.RawMessage, error) {
	return t.send(ctx, r, false)
}

func (t *transport) send(ctx context.Context, r request, isRetry bool) (json.RawMessage, error) {
	staleAccess := ""
	if r.auth == authPartner {
		staleAccess = t.tokenStore.Token()
	}

	status, env, retryAfter, err := t.dispatch(ctx, r)
	if err != nil {
		return nil, err
	}

	if status >= 200 && status < 300 && env.ResultType != nil && *env.ResultType == 0 {
		return env.Data, nil
	}

	expired := status == http.StatusUnauthorized || (env.ResultType != nil && *env.ResultType == 4)
	if r.auth == authPartner && expired && !isRetry && t.tryRefresh(ctx, staleAccess) {
		return t.send(ctx, r, true)
	}

	// A revoked session is worth forgetting; a merely expired access token is
	// not, since the caller may want to inspect it.
	if env.ResultType != nil && *env.ResultType == 2 {
		t.clearTokens()
	}

	return nil, t.toError(r, status, env, retryAfter)
}

// tryRefresh performs a single token refresh. Callers pass the access token that
// failed; if another goroutine already refreshed it, this returns true without
// issuing a second refresh.
func (t *transport) tryRefresh(ctx context.Context, staleAccess string) bool {
	t.refreshMu.Lock()
	defer t.refreshMu.Unlock()

	if staleAccess != "" && t.tokenStore.Token() != staleAccess {
		return true
	}

	refreshToken := t.refreshToken()
	if refreshToken == "" || t.clientID == "" || t.clientSecret == "" {
		return false
	}

	status, env, _, err := t.dispatch(ctx, request{
		method: http.MethodPost,
		path:   "/general/refreshApi",
		auth:   authPublic,
		body: map[string]any{
			"refreshToken":    refreshToken,
			"clientId":        t.clientID,
			"clientSecretKey": t.clientSecret,
		},
	})
	if err != nil {
		return false
	}
	if status < 200 || status >= 300 || env.ResultType == nil || *env.ResultType != 0 {
		t.clearTokens()
		return false
	}

	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(env.Data, &tokens); err != nil || tokens.AccessToken == "" {
		t.clearTokens()
		return false
	}
	newRefresh := tokens.RefreshToken
	if newRefresh == "" {
		newRefresh = refreshToken
	}
	t.setTokens(tokens.AccessToken, newRefresh)
	return true
}

// refresh forces a refresh, reporting failure as an error.
func (t *transport) refresh(ctx context.Context) error {
	if !t.tryRefresh(ctx, "") {
		return newAPIError(http.MethodPost, "/general/refreshApi", "token refresh failed",
			http.StatusUnauthorized, nil, nil, nil, nil)
	}
	return nil
}

func (t *transport) dispatch(ctx context.Context, r request) (int, envelope, string, error) {
	var token string
	if r.auth == authPartner {
		token = t.tokenStore.Token()
		if token == "" {
			// Dispatching anyway would only come back as an opaque 401.
			return 0, envelope{}, "", newAPIError(r.method, r.path,
				"no access token available: call Auth.Connect, or build the client with WithPartnerToken",
				http.StatusUnauthorized, nil, nil, nil, nil)
		}
	}

	var body io.Reader
	hasBody := r.body != nil && r.method != http.MethodGet
	if hasBody {
		encoded, err := json.Marshal(r.body)
		if err != nil {
			return 0, envelope{}, "", &TransportError{Message: fmt.Sprintf("bulutklinik: encode %s %s: %v", r.method, r.path, err), Err: err}
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, r.method, t.baseURL+r.path, body)
	if err != nil {
		return 0, envelope{}, "", &TransportError{Message: fmt.Sprintf("bulutklinik: build request %s %s: %v", r.method, r.path, err), Err: err}
	}
	req.Header.Set("Accept", "application/json")
	lang := r.lang
	if lang == "" {
		lang = t.lang
	}
	req.Header.Set("lang", lang)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return 0, envelope{}, "", &TransportError{Message: fmt.Sprintf("bulutklinik: %s %s: %v", r.method, r.path, err), Err: err}
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var env envelope
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &env)
	}
	return resp.StatusCode, env, resp.Header.Get("Retry-After"), nil
}

func (t *transport) toError(r request, status int, env envelope, retryAfter string) *APIError {
	message := env.ErrorMessage
	if message == "" {
		message = "request failed"
	}
	var ra *int
	if retryAfter != "" {
		if n, err := strconv.Atoi(retryAfter); err == nil {
			ra = &n
		}
	}
	return newAPIError(r.method, r.path, message, status, env.ResultType, env.ErrorType, env.Data, ra)
}

// strOrNil returns nil for an empty string so it serializes to JSON null.
func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}
