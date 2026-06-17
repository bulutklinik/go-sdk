package bulutklinik_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

func newTestClient(t *testing.T, handler http.HandlerFunc, opts ...bk.Option) (*bk.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	all := append([]bk.Option{bk.WithBaseURL(srv.URL)}, opts...)
	return bk.NewClient(all...), srv
}

func TestQuickSearchSuccess(t *testing.T) {
	var gotAuth, gotLang, gotPath, gotBody string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotLang = r.Header.Get("lang")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"searchedDoctors":[]}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	data, err := client.Doctors.QuickSearch(context.Background(), "kardiyo", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"searchedDoctors":[]}` {
		t.Errorf("data = %s", data)
	}
	if gotPath != "/patients/quickSearch" {
		t.Errorf("path = %s", gotPath)
	}
	if gotAuth != "Bearer abc" {
		t.Errorf("auth = %q", gotAuth)
	}
	if gotLang != "tr" {
		t.Errorf("lang = %q", gotLang)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(gotBody), &body)
	if body["searchText"] != "kardiyo" || body["listType"] != nil || body["location"] != nil {
		t.Errorf("body = %s", gotBody)
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
		}, bk.WithTokenStore(bk.NewInMemoryTokenStore("a", "")))

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
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("a", "")))

	_, err := client.Doctors.Branches(context.Background())
	var apiErr *bk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.RetryAfter == nil || *apiErr.RetryAfter != 30 {
		t.Errorf("retryAfter = %v", apiErr.RetryAfter)
	}
}

func TestRefreshAndRetry(t *testing.T) {
	dataCalls := 0
	var lastAuth string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/general/refreshApi" {
			_, _ = w.Write([]byte(`{"resultType":0,"data":{"access_token":"new","refresh_token":"r2"}}`))
			return
		}
		lastAuth = r.Header.Get("Authorization")
		dataCalls++
		if dataCalls == 1 {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"resultType":4}`))
			return
		}
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"ok":true}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("old", "r")), bk.WithCredentials("c", "s"))

	data, err := client.Measures.Last(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("data = %s", data)
	}
	if client.TokenStore().AccessToken() != "new" {
		t.Errorf("token not refreshed: %q", client.TokenStore().AccessToken())
	}
	if lastAuth != "Bearer new" {
		t.Errorf("retry auth = %q", lastAuth)
	}
}

func TestLogoutClearsStore(t *testing.T) {
	store := bk.NewInMemoryTokenStore("a", "r")
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"resultType":2,"errorMessage":"logged out"}`))
	}, bk.WithTokenStore(store))

	_, err := client.Measures.Last(context.Background())
	if !errors.Is(err, bk.ErrAuthentication) {
		t.Errorf("want ErrAuthentication, got %v", err)
	}
	if store.AccessToken() != "" {
		t.Errorf("store not cleared")
	}
}

func TestTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // closed: requests fail at the transport layer
	client := bk.NewClient(bk.WithBaseURL(srv.URL), bk.WithTokenStore(bk.NewInMemoryTokenStore("a", "")))

	_, err := client.Doctors.Branches(context.Background())
	if !errors.Is(err, bk.ErrTransport) {
		t.Errorf("want ErrTransport, got %v", err)
	}
}

func TestMeasurePathAndPartner(t *testing.T) {
	var paths, auths []string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		auths = append(auths, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"resultType":0,"data":null}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("a", "")), bk.WithPartnerToken("PT"))

	gt := 0
	if _, err := client.Measures.List(context.Background(), "glucose", 1, &gt); err != nil {
		t.Fatal(err)
	}
	if paths[0] != "/patients/userMeasuresList/glucose/1/0" {
		t.Errorf("path = %s", paths[0])
	}

	if _, err := client.Measures.PartnerHealthInformation(context.Background(), "", "5551112233",
		[]map[string]any{{"type": "pulse", "date_time": "2026-06-17 09:00", "pulse": 72}}); err != nil {
		t.Fatal(err)
	}
	if auths[len(auths)-1] != "Bearer PT" {
		t.Errorf("partner auth = %q", auths[len(auths)-1])
	}
}
