// Command livecheck is a read-only smoke test against the Bulutklinik test
// environment (apitest). It doubles as an end-to-end usage example.
//
// Needs a partner token issued for a test company with the apiouther scope:
//
//	BK_PARTNER_TOKEN=... go run ./examples/livecheck
//
// Unlike the patient surface there is no shared test credential — the token is
// per-integration. Steps that touch a patient need one that exists inside the
// token's own company; set BK_PATIENT_TCKN or BK_PATIENT_PHONE to run them.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	bk "github.com/bulutklinik/go-sdk"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func countArray(raw json.RawMessage) int {
	var a []any
	_ = json.Unmarshal(raw, &a)
	return len(a)
}

func countMap(raw json.RawMessage) int {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return len(m)
}

func main() {
	partnerToken := os.Getenv("BK_PARTNER_TOKEN")
	if partnerToken == "" {
		fmt.Fprintln(os.Stderr, "BK_PARTNER_TOKEN is required.")
		os.Exit(2)
	}

	client, err := bk.NewClient(
		bk.WithEnvironment(bk.Test),
		bk.WithAPIVersion(bk.APIVersion(env("BK_API_VERSION", "v3"))),
		bk.WithPartnerToken(partnerToken),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	ctx := context.Background()

	pass, total := 0, 0
	step := func(name string, fn func() (json.RawMessage, error)) json.RawMessage {
		total++
		data, err := fn()
		if err != nil {
			detail := ""
			var apiErr *bk.APIError
			if errors.As(err, &apiErr) {
				detail = fmt.Sprintf(" [http=%d resultType=%v errorType=%v]", apiErr.HTTPStatus, apiErr.ResultType, apiErr.ErrorType)
			}
			fmt.Printf("ERR %s: %v%s\n", name, err, detail)
			return nil
		}
		pass++
		fmt.Printf("OK  %s\n", name)
		return data
	}

	// Scope-only steps: these prove the token and base URL without any patient.
	if b := step("doctors.branches", func() (json.RawMessage, error) { return client.Doctors.Branches(ctx) }); b != nil {
		fmt.Printf("    branches=%d\n", countArray(b))
	}
	if l := step("doctors.locations", func() (json.RawMessage, error) { return client.Doctors.Locations(ctx) }); l != nil {
		fmt.Printf("    locations=%d\n", countArray(l))
	}
	if c := step("laboratory.catalog", func() (json.RawMessage, error) { return client.Laboratory.Catalog(ctx) }); c != nil {
		fmt.Printf("    catalog=%d\n", countArray(c))
	}
	step("doctors.search", func() (json.RawMessage, error) {
		return client.Doctors.Search(ctx, map[string]any{"withFreeText": "kardiyoloji"}, 1, []string{"slot"})
	})

	doctorID := env("BK_DOCTOR_ID", "8282")
	if d := step("doctors.detail", func() (json.RawMessage, error) { return client.Doctors.Detail(ctx, doctorID) }); d != nil {
		fmt.Printf("    detailKeys=%d\n", countMap(d))
	}
	step("appointments.checkDoctor", func() (json.RawMessage, error) {
		return client.Appointments.CheckDoctor(ctx, doctorID, 0)
	})
	if sl := step("slots.schedule", func() (json.RawMessage, error) {
		return client.Slots.Schedule(ctx, bk.ScheduleInput{DoctorID: doctorID})
	}); sl != nil {
		fmt.Printf("    slotDays=%d\n", countMap(sl))
	}

	// Patient-scoped steps. A TCKN that works on the patient surface will not
	// necessarily resolve here: the patient must exist in the token's company.
	var patient bk.Patient
	switch {
	case os.Getenv("BK_PATIENT_TCKN") != "":
		patient = bk.Patient{IdentityNumber: os.Getenv("BK_PATIENT_TCKN")}
	case os.Getenv("BK_PATIENT_PHONE") != "":
		patient = bk.Patient{PhoneNumber: os.Getenv("BK_PATIENT_PHONE")}
	}

	if patient != (bk.Patient{}) {
		if last := step("measures.last", func() (json.RawMessage, error) { return client.Measures.Last(ctx, patient) }); last != nil {
			fmt.Printf("    measuresLastKeys=%d\n", countMap(last))
		}
		step("diets.list", func() (json.RawMessage, error) { return client.Diets.List(ctx, patient, nil) })
		step("laboratory.results", func() (json.RawMessage, error) { return client.Laboratory.Results(ctx, patient, nil) })
	} else {
		fmt.Println("--  skipped patient-scoped steps (set BK_PATIENT_TCKN or BK_PATIENT_PHONE)")
	}

	fmt.Printf("\nSUMMARY: %d/%d steps OK\n", pass, total)
}
