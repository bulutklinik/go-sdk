package bulutklinik

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
// There is no silent refresh: a partner token is issued out of band and cannot
// be renewed from here, so an expired one (401 / resultType 4) surfaces as an
// [ErrAuthentication] failure instead of being retried.
type transport struct {
	httpClient *http.Client
	baseURL    string
	lang       string
	tokenStore TokenStore
}

func (t *transport) do(ctx context.Context, r request) (json.RawMessage, error) {
	status, env, retryAfter, err := t.dispatch(ctx, r)
	if err != nil {
		return nil, err
	}

	if status >= 200 && status < 300 && env.ResultType != nil && *env.ResultType == 0 {
		return env.Data, nil
	}

	// A revoked token is worth forgetting; an expired one is not, since the
	// caller may want to inspect it while installing a replacement.
	if env.ResultType != nil && *env.ResultType == 2 {
		t.tokenStore.Clear()
	}

	return nil, t.toError(r, status, env, retryAfter)
}

func (t *transport) dispatch(ctx context.Context, r request) (int, envelope, string, error) {
	var token string
	if r.auth == authPartner {
		token = t.tokenStore.Token()
		if token == "" {
			// Dispatching anyway would only come back as an opaque 401.
			return 0, envelope{}, "", newAPIError(r.method, r.path,
				"no partner token configured", http.StatusUnauthorized,
				nil, nil, nil, nil)
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
