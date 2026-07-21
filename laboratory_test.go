package bulutklinik_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

func TestLaboratoryResultsWithPage(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"foundTestsCount":0,"foundTests":[]}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	data, err := client.Laboratory.Results(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"foundTestsCount":0,"foundTests":[]}` {
		t.Errorf("data = %s", data)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s", gotMethod)
	}
	if gotPath != "/patients/userLabTestList/2" {
		t.Errorf("path = %s", gotPath)
	}
	if gotAuth != "Bearer abc" {
		t.Errorf("auth = %q", gotAuth)
	}
}

func TestLaboratoryResultsWithoutPage(t *testing.T) {
	var gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"foundTestsCount":0,"foundTests":[]}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	if _, err := client.Laboratory.Results(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/patients/userLabTestList" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestLaboratoryResultDetailStringID(t *testing.T) {
	var gotMethod, gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"id":"4821-lab"}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	if _, err := client.Laboratory.ResultDetail(context.Background(), "4821-lab"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s", gotMethod)
	}
	if gotPath != "/patients/userLabTestDetail/4821-lab" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestLaboratoryCatalog(t *testing.T) {
	var gotMethod, gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"test_groups":[]}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	if _, err := client.Laboratory.Catalog(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s", gotMethod)
	}
	if gotPath != "/patients/allLaboratoryTests" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestLaboratoryCatalogDetail(t *testing.T) {
	var gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"id":7}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	if _, err := client.Laboratory.CatalogDetail(context.Background(), "7"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/patients/laboratoryTestDetail/7" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestLaboratoryOrder(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"preOrderId":99}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	if _, err := client.Laboratory.Order(context.Background(), bk.LabOrderInput{
		TestID:       "12",
		AddressID:    "34",
		LaboratoryID: "56",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s", gotMethod)
	}
	if gotPath != "/patients/addNewLaboratoryTest" {
		t.Errorf("path = %s", gotPath)
	}
	if gotAuth != "Bearer abc" {
		t.Errorf("auth = %q", gotAuth)
	}
	want := map[string]any{
		"testId":       "12",
		"addressId":    "34",
		"laboratoryId": "56",
	}
	if !reflect.DeepEqual(gotBody, want) {
		t.Errorf("body = %#v", gotBody)
	}
}
