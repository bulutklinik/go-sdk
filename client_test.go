package bulutklinik_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

func TestDoDefaultsToPartner(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"ok":true}}`))
	}, partnerToken("PT"))

	data, err := client.Do(context.Background(), "GET", "/outher/customEndpoint", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("data = %s", data)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/outher/customEndpoint" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "Bearer PT" {
		t.Errorf("auth = %q", gotAuth)
	}
}

func TestDoPublicPOST(t *testing.T) {
	var gotAuth, gotMethod, gotBody string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"id":7}}`))
	}, partnerToken("PT"))

	data, err := client.Do(context.Background(), "POST", "/general/somePublicEndpoint", &bk.RequestOptions{
		Auth: "public",
		Body: map[string]any{"foo": "bar"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"id":7}` {
		t.Errorf("data = %s", data)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q", gotMethod)
	}
	if gotAuth != "" {
		t.Errorf("public request should omit Authorization, got %q", gotAuth)
	}
	if gotBody != `{"foo":"bar"}` {
		t.Errorf("body = %s", gotBody)
	}
}
