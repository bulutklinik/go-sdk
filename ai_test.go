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

func TestSkinAnalyze(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"status":[{"id":1,"label":"nevus"}]}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	data, err := client.Skin.Analyze(context.Background(), []map[string]any{{"image": "BASE64", "branch_id": 42}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"status":[{"id":1,"label":"nevus"}]}` {
		t.Errorf("data = %s", data)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s", gotMethod)
	}
	if gotPath != "/patients/imageCheck" {
		t.Errorf("path = %s", gotPath)
	}
	if gotAuth != "Bearer abc" {
		t.Errorf("auth = %q", gotAuth)
	}
	want := map[string]any{"images": []any{map[string]any{"image": "BASE64", "branch_id": float64(42)}}}
	if !reflect.DeepEqual(gotBody, want) {
		t.Errorf("body = %#v", gotBody)
	}
}

func TestMealsAnalyzeWithOptionalFields(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"status":{"comment":"{}"}}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	grams := 300
	note := "az yağlı"
	if _, err := client.Meals.Analyze(context.Background(), bk.MealInput{
		Image:        "BASE64",
		PortionSize:  "custom",
		MealType:     "lunch",
		PortionGrams: &grams,
		Note:         &note,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s", gotMethod)
	}
	if gotPath != "/patients/imageAnalyzeMeal" {
		t.Errorf("path = %s", gotPath)
	}
	if gotAuth != "Bearer abc" {
		t.Errorf("auth = %q", gotAuth)
	}
	want := map[string]any{
		"image":         "BASE64",
		"portion_size":  "custom",
		"meal_type":     "lunch",
		"portion_grams": float64(300),
		"note":          "az yağlı",
	}
	if !reflect.DeepEqual(gotBody, want) {
		t.Errorf("body = %#v", gotBody)
	}
}

func TestMealsAnalyzeOmitsOptionalFields(t *testing.T) {
	var gotBody map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"resultType":0,"data":{"status":{"comment":"{}"}}}`))
	}, bk.WithTokenStore(bk.NewInMemoryTokenStore("abc", "")))

	if _, err := client.Meals.Analyze(context.Background(), bk.MealInput{
		Image:       "BASE64",
		PortionSize: "medium",
		MealType:    "snack",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{
		"image":        "BASE64",
		"portion_size": "medium",
		"meal_type":    "snack",
	}
	if !reflect.DeepEqual(gotBody, want) {
		t.Errorf("body = %#v", gotBody)
	}
}
