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
	authBearer
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

type transport struct {
	httpClient   *http.Client
	baseURL      string
	lang         string
	clientID     string
	clientSecret string
	partnerToken string
	tokenStore   TokenStore
	refreshMu    sync.Mutex
}

func (t *transport) do(ctx context.Context, r request) (json.RawMessage, error) {
	return t.send(ctx, r, false)
}

func (t *transport) send(ctx context.Context, r request, isRetry bool) (json.RawMessage, error) {
	staleAccess := ""
	if r.auth == authBearer {
		staleAccess = t.tokenStore.AccessToken()
	}

	status, env, retryAfter, err := t.dispatch(ctx, r)
	if err != nil {
		return nil, err
	}

	if status >= 200 && status < 300 && env.ResultType != nil && *env.ResultType == 0 {
		return env.Data, nil
	}

	expired := status == http.StatusUnauthorized || (env.ResultType != nil && *env.ResultType == 4)
	if r.auth == authBearer && expired && !isRetry && t.tryRefresh(ctx, staleAccess) {
		return t.send(ctx, r, true)
	}

	if env.ResultType != nil && *env.ResultType == 2 {
		t.tokenStore.Clear()
	}

	return nil, t.toError(r, status, env, retryAfter)
}

func (t *transport) dispatch(ctx context.Context, r request) (int, envelope, string, error) {
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
	switch r.auth {
	case authBearer:
		if tok := t.tokenStore.AccessToken(); tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
	case authPartner:
		if t.partnerToken != "" {
			req.Header.Set("Authorization", "Bearer "+t.partnerToken)
		}
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

// tryRefresh performs a single token refresh. It is concurrency-safe: callers
// pass the access token that failed; if another goroutine already refreshed it,
// this returns true without issuing a second refresh.
func (t *transport) tryRefresh(ctx context.Context, staleAccess string) bool {
	t.refreshMu.Lock()
	defer t.refreshMu.Unlock()

	if staleAccess != "" && t.tokenStore.AccessToken() != staleAccess {
		return true
	}

	refreshToken := t.tokenStore.RefreshToken()
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
		t.tokenStore.Clear()
		return false
	}

	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(env.Data, &tokens); err != nil || tokens.AccessToken == "" {
		t.tokenStore.Clear()
		return false
	}
	newRefresh := tokens.RefreshToken
	if newRefresh == "" {
		newRefresh = refreshToken
	}
	t.tokenStore.SetTokens(tokens.AccessToken, newRefresh)
	return true
}

func (t *transport) refresh(ctx context.Context) error {
	if !t.tryRefresh(ctx, "") {
		return newAPIError(http.MethodPost, "/general/refreshApi", "token refresh failed", http.StatusUnauthorized, nil, nil, nil, nil)
	}
	return nil
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
