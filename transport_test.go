package bulutklinik_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

// newTestClient wires a client at a local test server. Unless an option
// overrides it, the credential is the partner token "PT".
func newTestClient(t *testing.T, handler http.HandlerFunc, opts ...bk.Option) (*bk.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	all := append([]bk.Option{bk.WithBaseURL(srv.URL)}, opts...)
	client, err := bk.NewClient(all...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client, srv
}

func partnerToken(token string) bk.Option { return bk.WithTokenStore(bk.NewInMemoryTokenStore(token)) }

func TestSearchSuccess(t *testing.T) {
	var gotAuth, gotLang, gotPath, gotBody string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotLang = r.Header.Get("lang")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"foundDoctors":[]}}`))
	}, partnerToken("PT"))

	data, err := client.Doctors.Search(context.Background(),
		map[string]any{"withFreeText": "kardiyoloji"}, 1, []string{"slot"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"foundDoctors":[]}` {
		t.Errorf("data = %s", data)
	}
	if gotPath != "/outher/search" {
		t.Errorf("path = %s", gotPath)
	}
	if gotAuth != "Bearer PT" {
		t.Errorf("auth = %q", gotAuth)
	}
	if gotLang != "tr" {
		t.Errorf("lang = %q", gotLang)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(gotBody), &body)
	if body["currentPage"] != float64(1) {
		t.Errorf("body = %s", gotBody)
	}
}

func TestAPIVersionSelectsTheBaseURL(t *testing.T) {
	for _, c := range []struct {
		version bk.APIVersion
		want    string
	}{
		{bk.V3, "/api/v3/outher/branches"},
		{bk.V4, "/api/v4/outher/branches"},
	} {
		client, err := bk.NewClient(
			bk.WithEnvironment(bk.Test),
			bk.WithAPIVersion(c.version),
			bk.WithPartnerToken("PT"),
			// A transport that fails immediately, so nothing leaves the machine
			// while still reporting the URL that was built.
			bk.WithHTTPClient(&http.Client{Transport: failingTransport{}}),
		)
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}

		_, err = client.Doctors.Branches(context.Background())
		var tErr *bk.TransportError
		if !errors.As(err, &tErr) {
			t.Fatalf("%s: want *TransportError, got %v", c.version, err)
		}
		if !strings.Contains(tErr.Err.Error(), c.want) {
			t.Errorf("%s: built URL %v, want it to contain %q", c.version, tErr.Err, c.want)
		}
	}
}

// failingTransport refuses every request, echoing the URL in the error.
type failingTransport struct{}

func (failingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New("refused: " + r.URL.String())
}

func TestPartnerTokenAndTokenStoreConflict(t *testing.T) {
	_, err := bk.NewClient(bk.WithPartnerToken("PT"), bk.WithTokenStore(bk.NewInMemoryTokenStore("OTHER")))
	if !errors.Is(err, bk.ErrCredentialConflict) {
		t.Errorf("want ErrCredentialConflict, got %v", err)
	}
}

func TestMissingTokenFailsBeforeDispatch(t *testing.T) {
	dispatched := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		dispatched++
		_, _ = w.Write([]byte(`{"resultType":0,"data":null}`))
	}, partnerToken(""))

	_, err := client.Doctors.Branches(context.Background())
	if !errors.Is(err, bk.ErrAuthentication) {
		t.Errorf("want ErrAuthentication, got %v", err)
	}
	if dispatched != 0 {
		t.Errorf("dispatched %d requests, want 0", dispatched)
	}
}

func TestTokenIsReadFromStoreOnEveryCall(t *testing.T) {
	var auths []string
	store := bk.NewInMemoryTokenStore("first")
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"resultType":0,"data":null}`))
	}, bk.WithTokenStore(store))

	_, _ = client.Doctors.Branches(context.Background())
	store.SetToken("second")
	_, _ = client.Doctors.Branches(context.Background())

	if auths[0] != "Bearer first" || auths[1] != "Bearer second" {
		t.Errorf("auths = %v", auths)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		status   int
		body     string
		sentinel error
	}{
		{422, `{"resultType":1,"errorType":"validation"}`, bk.ErrValidation},
		{404, `{"resultType":1,"errorType":1,"errorMessage":"Bilinmeyen"}`, bk.ErrNotFound}, // numeric errorType
		{403, `{"resultType":1}`, bk.ErrAuthorization},
		{429, `{"resultType":1}`, bk.ErrRateLimit},
	}
	for _, c := range cases {
		c := c
		client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
			_, _ = w.Write([]byte(c.body))
		}, partnerToken("PT"))

		_, err := client.Doctors.Branches(context.Background())
		if !errors.Is(err, c.sentinel) {
			t.Errorf("status %d: want %v, got %v", c.status, c.sentinel, err)
		}
		if !errors.Is(err, bk.ErrAPI) {
			t.Errorf("status %d: should also match ErrAPI", c.status)
		}
	}
}

func TestRateLimitRetryAfter(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"resultType":1}`))
	}, partnerToken("PT"))

	_, err := client.Doctors.Branches(context.Background())
	var apiErr *bk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.RetryAfter == nil || *apiErr.RetryAfter != 30 {
		t.Errorf("retryAfter = %v", apiErr.RetryAfter)
	}
}

func TestExpiredTokenIsNotRetried(t *testing.T) {
	attempts := 0
	store := bk.NewInMemoryTokenStore("expired")
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"resultType":4,"errorMessage":"You must log in."}`))
	}, bk.WithTokenStore(store))

	_, err := client.Measures.Last(context.Background(), bk.Patient{IdentityNumber: "12345678901"})
	if !errors.Is(err, bk.ErrAuthentication) {
		t.Fatalf("want ErrAuthentication, got %v", err)
	}
	if !strings.Contains(err.Error(), "cannot refresh it") {
		t.Errorf("message should say what to do: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry)", attempts)
	}
	// An expired token is kept: the caller may want to inspect it while
	// installing the replacement. Only a revoked one is cleared.
	if store.Token() != "expired" {
		t.Errorf("token = %q, want it kept", store.Token())
	}
}

func TestLogoutClearsStore(t *testing.T) {
	store := bk.NewInMemoryTokenStore("revoked")
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"resultType":2,"errorMessage":"logged out"}`))
	}, bk.WithTokenStore(store))

	_, err := client.Measures.Last(context.Background(), bk.Patient{IdentityNumber: "12345678901"})
	if !errors.Is(err, bk.ErrAuthentication) {
		t.Errorf("want ErrAuthentication, got %v", err)
	}
	if store.Token() != "" {
		t.Errorf("store not cleared")
	}
}

func TestTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // closed: requests fail at the transport layer
	client, err := bk.NewClient(bk.WithBaseURL(srv.URL), bk.WithPartnerToken("PT"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Doctors.Branches(context.Background())
	if !errors.Is(err, bk.ErrTransport) {
		t.Errorf("want ErrTransport, got %v", err)
	}
}
