package bulutklinik_test

import (
	"context"
	"net/http"
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

func TestDietsListWithPage(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"foundDietsCount":0,"foundDiets":[]}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	data, err := client.Diets.List(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"foundDietsCount":0,"foundDiets":[]}` {
		t.Errorf("data = %s", data)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s", gotMethod)
	}
	if gotPath != "/patients/dietLists/3" {
		t.Errorf("path = %s", gotPath)
	}
	if gotAuth != "Bearer abc" {
		t.Errorf("auth = %q", gotAuth)
	}
}

func TestDietsListWithoutPage(t *testing.T) {
	var gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"foundDietsCount":0,"foundDiets":[]}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	if _, err := client.Diets.List(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/patients/dietLists" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestDietsDetail(t *testing.T) {
	var gotMethod, gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"resultType":0,"data":[]}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	if _, err := client.Diets.Detail(context.Background(), "55"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s", gotMethod)
	}
	if gotPath != "/patients/diet/55" {
		t.Errorf("path = %s", gotPath)
	}
}
