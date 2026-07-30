package bulutklinik_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

const tokensJSON = `{"resultType":0,"data":{"access_token":"AT","refresh_token":"RT"}}`

var authRef = bk.Patient{IdentityNumber: "12345678901"}

func TestConnectPostsPortalCredentialsAndStoresBothTokens(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	store := bk.NewInMemoryTokenStore("")
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_, _ = w.Write([]byte(tokensJSON))
	}, bk.WithTokenStore(store), bk.WithCredentials("cid", "csecret"))

	result, err := client.Auth.Connect(context.Background(), bk.ConnectInput{
		APIUserName:     "svc@app.bulutklinik",
		APIUserPassword: "hunter2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TwoFactorRequired {
		t.Error("did not expect a 2FA challenge")
	}
	if gotPath != "/general/connectApi" {
		t.Errorf("path = %q", gotPath)
	}
	// The login call is public — it is what produces the credential.
	if gotAuth != "" {
		t.Errorf("connect should not send Authorization, got %q", gotAuth)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(gotBody), &body)
	for k, want := range map[string]string{
		"apiClientId": "cid", "apiSecretKey": "csecret",
		"apiUserName": "svc@app.bulutklinik", "apiUserPassword": "hunter2", "loginMode": "email",
	} {
		if body[k] != want {
			t.Errorf("body[%q] = %v, want %q", k, body[k], want)
		}
	}
	if store.Token() != "AT" || store.RefreshToken() != "RT" {
		t.Errorf("tokens = %q / %q", store.Token(), store.RefreshToken())
	}
}

func TestConnectSurfacesTwoFactorChallenge(t *testing.T) {
	store := bk.NewInMemoryTokenStore("")
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"response":"BLOB"}}`))
	}, bk.WithTokenStore(store), bk.WithCredentials("cid", "csecret"))

	result, err := client.Auth.Connect(context.Background(), bk.ConnectInput{
		APIUserName: "svc", APIUserPassword: "p",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TwoFactorRequired || result.TwoFactorResponse != "BLOB" {
		t.Errorf("result = %+v", result)
	}
	if store.Token() != "" {
		t.Errorf("no token should be stored, got %q", store.Token())
	}
}

func TestConnectRequiresClientCredentials(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(tokensJSON))
	}, partnerToken("PT"))

	_, err := client.Auth.Connect(context.Background(), bk.ConnectInput{
		APIUserName: "svc", APIUserPassword: "p",
	})
	if !errors.Is(err, bk.ErrMissingClientCredentials) {
		t.Errorf("want ErrMissingClientCredentials, got %v", err)
	}
}

func TestRefreshesOnceThenRetries(t *testing.T) {
	dataCalls := 0
	var lastAuth, refreshBody string
	store := bk.NewInMemoryTokenStorePair("AT", "RT")
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/general/refreshApi" {
			raw, _ := io.ReadAll(r.Body)
			refreshBody = string(raw)
			_, _ = w.Write([]byte(`{"resultType":0,"data":{"access_token":"AT2","refresh_token":"RT2"}}`))
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
	}, bk.WithTokenStore(store), bk.WithCredentials("cid", "csecret"))

	data, err := client.Measures.Last(context.Background(), authRef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("data = %s", data)
	}
	if store.Token() != "AT2" || store.RefreshToken() != "RT2" {
		t.Errorf("tokens = %q / %q", store.Token(), store.RefreshToken())
	}
	if lastAuth != "Bearer AT2" {
		t.Errorf("retry auth = %q", lastAuth)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(refreshBody), &body)
	if body["refreshToken"] != "RT" || body["clientId"] != "cid" || body["clientSecretKey"] != "csecret" {
		t.Errorf("refresh body = %s", refreshBody)
	}
}

func TestRetriesAtMostOnceAndClearsOnFailedRefresh(t *testing.T) {
	refreshCalls := 0
	store := bk.NewInMemoryTokenStorePair("AT", "RT")
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/general/refreshApi" {
			refreshCalls++
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"resultType":1}`))
			return
		}
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"resultType":4}`))
	}, bk.WithTokenStore(store), bk.WithCredentials("cid", "csecret"))

	_, err := client.Measures.Last(context.Background(), authRef)
	if !errors.Is(err, bk.ErrAuthentication) {
		t.Fatalf("want ErrAuthentication, got %v", err)
	}
	if refreshCalls != 1 {
		t.Errorf("refreshCalls = %d, want 1", refreshCalls)
	}
	if store.Token() != "" {
		t.Errorf("store not cleared: %q", store.Token())
	}
}

func TestNoRefreshAttemptWithoutARefreshToken(t *testing.T) {
	refreshCalls := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/general/refreshApi" {
			refreshCalls++
		}
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"resultType":4}`))
	}, partnerToken("AT"))

	_, err := client.Doctors.Branches(context.Background())
	if !errors.Is(err, bk.ErrAuthentication) {
		t.Fatalf("want ErrAuthentication, got %v", err)
	}
	if refreshCalls != 0 {
		t.Errorf("refreshCalls = %d, want 0", refreshCalls)
	}
}

// legacyStore is a store written against spec 1.0.x: access token only.
type legacyStore struct{ token string }

func (s *legacyStore) Token() string     { return s.token }
func (s *legacyStore) SetToken(t string) { s.token = t }
func (s *legacyStore) Clear()            { s.token = "" }

func TestStoreWithoutRefreshSupportStillRefreshesInMemory(t *testing.T) {
	dataCalls := 0
	store := &legacyStore{}
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/general/connectApi":
			_, _ = w.Write([]byte(tokensJSON))
			return
		case "/general/refreshApi":
			_, _ = w.Write([]byte(`{"resultType":0,"data":{"access_token":"AT2"}}`))
			return
		}
		dataCalls++
		if dataCalls == 1 {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"resultType":4}`))
			return
		}
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"ok":true}}`))
	}, bk.WithTokenStore(store), bk.WithCredentials("cid", "csecret"))

	ctx := context.Background()
	if _, err := client.Auth.Connect(ctx, bk.ConnectInput{APIUserName: "svc", APIUserPassword: "p"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Measures.Last(ctx, authRef); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.token != "AT2" {
		t.Errorf("token = %q, want AT2", store.token)
	}
}

func TestDisconnectSendsAnEmptyBodyAndClears(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	store := bk.NewInMemoryTokenStorePair("AT", "RT")
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"resultType":0,"data":null}`))
	}, bk.WithTokenStore(store))

	if err := client.Auth.Disconnect(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/general/disconnectApi" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "Bearer AT" {
		t.Errorf("auth = %q", gotAuth)
	}
	// The device-cleanup fields are deliberately not sent: the server's `device`
	// mapping has no default branch.
	if gotBody != "{}" {
		t.Errorf("body = %q, want {}", gotBody)
	}
	if store.Token() != "" || store.RefreshToken() != "" {
		t.Errorf("store not cleared: %q / %q", store.Token(), store.RefreshToken())
	}
}
