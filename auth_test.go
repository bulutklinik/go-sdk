package bulutklinik_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

func TestConnectStoresTokensAndFillsCredentials(t *testing.T) {
	store := bk.NewInMemoryTokenStore("", "")
	var gotBody string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"access_token":"t","refresh_token":"r","password_policy":{}}}`))
	}, bk.WithTokenStore(store), bk.WithCredentials("c", "s"))

	result, err := client.Auth.Connect(context.Background(), bk.ConnectInput{
		APIUserName: "u", APIUserPassword: "p", LoginMode: "email",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TwoFactorRequired {
		t.Error("expected TwoFactorRequired = false")
	}
	if store.AccessToken() != "t" || store.RefreshToken() != "r" {
		t.Errorf("tokens not stored: %q %q", store.AccessToken(), store.RefreshToken())
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(gotBody), &body)
	if body["apiClientId"] != "c" || body["apiSecretKey"] != "s" || body["loginMode"] != "email" {
		t.Errorf("body = %s", gotBody)
	}
}

func TestConnectTwoFactorChallenge(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"response":"BLOB"}}`))
	}, bk.WithCredentials("c", "s"))

	result, err := client.Auth.Connect(context.Background(), bk.ConnectInput{
		APIUserName: "u", APIUserPassword: "p", LoginMode: "email",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TwoFactorRequired || result.TwoFactorResponse != "BLOB" {
		t.Errorf("result = %+v", result)
	}
}

func TestDisconnectClearsStoreOnError(t *testing.T) {
	store := bk.NewInMemoryTokenStore("a", "r")
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"resultType":1,"errorMessage":"fail"}`))
	}, bk.WithTokenStore(store))

	if err := client.Auth.Disconnect(context.Background()); err == nil {
		t.Error("expected an error")
	}
	if store.AccessToken() != "" {
		t.Error("store not cleared")
	}
}
