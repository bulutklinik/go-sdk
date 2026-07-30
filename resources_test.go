package bulutklinik_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

type recorded struct {
	method string
	path   string
	auth   string
	body   map[string]any
}

// partnerClient records every call a client makes, against the partner token "PT".
func partnerClient(t *testing.T) (*bk.Client, *[]recorded) {
	t.Helper()
	var calls []recorded
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		calls = append(calls, recorded{r.Method, r.URL.Path, r.Header.Get("Authorization"), body})
		_, _ = w.Write([]byte(`{"resultType":0,"data":null}`))
	}, partnerToken("PT"))
	return client, &calls
}

func TestEveryCallUsesThePartnerToken(t *testing.T) {
	client, calls := partnerClient(t)
	ctx := context.Background()

	if _, err := client.Doctors.Branches(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Measures.Last(ctx, bk.Patient{IdentityNumber: "12345678901"}); err != nil {
		t.Fatal(err)
	}

	for _, c := range *calls {
		if c.auth != "Bearer PT" {
			t.Errorf("%s %s auth = %q, want Bearer PT", c.method, c.path, c.auth)
		}
	}
}

func TestDiscoveryPaths(t *testing.T) {
	client, calls := partnerClient(t)
	ctx := context.Background()

	_, _ = client.Doctors.Locations(ctx)
	_, _ = client.Doctors.Detail(ctx, 42)
	_, _ = client.Laboratory.Catalog(ctx)
	_, _ = client.Laboratory.CatalogDetail(ctx, 18246)
	_, _ = client.Slots.Schedule(ctx, bk.ScheduleInput{DoctorID: 7, ScheduleDate: "2026-08-01"})

	want := []string{
		"/outher/locations",
		"/outher/doctorInfos/42",
		"/outher/laboratoryCatalog",
		"/outher/laboratoryCatalog/18246",
		"/outher/doctorSlots",
	}
	for i, w := range want {
		if (*calls)[i].path != w {
			t.Errorf("call %d path = %q, want %q", i, (*calls)[i].path, w)
		}
	}
}

func TestPatientRefStaysOutOfThePath(t *testing.T) {
	client, calls := partnerClient(t)
	ctx := context.Background()
	patient := bk.Patient{IdentityNumber: "12345678901"}

	_, _ = client.Diets.List(ctx, patient, 2)
	_, _ = client.Measures.List(ctx, patient, "glucose", 1, nil)
	_, _ = client.Laboratory.Results(ctx, patient, nil)

	// The identity number must never leak into a URL — it would land in access
	// logs, proxy logs and error breadcrumbs.
	for _, c := range *calls {
		if strings.Contains(c.path, "12345678901") {
			t.Errorf("identity number leaked into path %q", c.path)
		}
	}

	if (*calls)[0].path != "/outher/dietLists" {
		t.Errorf("diets path = %q", (*calls)[0].path)
	}
	if (*calls)[1].path != "/outher/measuresList/glucose" {
		t.Errorf("measures path = %q", (*calls)[1].path)
	}
	if got := (*calls)[0].body["patient"].(map[string]any)["identityNumber"]; got != "12345678901" {
		t.Errorf("patient ref not in body: %v", got)
	}
}

func TestLabResultIDRoundTrips(t *testing.T) {
	client, calls := partnerClient(t)
	ctx := context.Background()
	patient := bk.Patient{IdentityNumber: "12345678901"}

	_, _ = client.Laboratory.ResultDetail(ctx, patient, "1234-lab")
	if got := (*calls)[0].body["testId"]; got != "1234-lab" {
		t.Errorf("testId = %v, want 1234-lab", got)
	}

	_, _ = client.Laboratory.ResultDetail(ctx, patient, 1234)
	if got := (*calls)[1].body["testId"]; got != "1234" {
		t.Errorf("testId = %v, want \"1234\"", got)
	}
}

func TestMeasureWriteVerbsAndPaths(t *testing.T) {
	client, calls := partnerClient(t)
	ctx := context.Background()
	writePatient := bk.Patient{Name: "Ada", Surname: "Lovelace", PhoneNumber: "+905551112233"}
	ref := bk.Patient{IdentityNumber: "12345678901"}

	_, _ = client.Measures.AddList(ctx, writePatient, []map[string]any{
		{"type": "pulse", "date_time": "2026-06-17 09:00", "pulse": 72},
	})
	_, _ = client.Measures.Add(ctx, writePatient, "tension", map[string]any{
		"date_time": "2026-06-17 09:00", "hypertension": 120, "hypotension": 80,
	})
	_, _ = client.Measures.Update(ctx, ref, "tension", 9, map[string]any{
		"date_time": "2026-06-17 10:00", "hypertension": 125, "hypotension": 85,
	})
	_, _ = client.Measures.Delete(ctx, ref, "tension", 9)

	want := []struct{ method, path string }{
		{http.MethodPost, "/outher/measures"},
		{http.MethodPost, "/outher/measure/tension"},
		{http.MethodPut, "/outher/measure/tension"},
		{http.MethodDelete, "/outher/measure/tension"},
	}
	for i, w := range want {
		if (*calls)[i].method != w.method || (*calls)[i].path != w.path {
			t.Errorf("call %d = %s %s, want %s %s", i, (*calls)[i].method, (*calls)[i].path, w.method, w.path)
		}
	}

	// Measure fields are flattened alongside patient, matching the server shape.
	if got := (*calls)[1].body["hypertension"]; got != float64(120) {
		t.Errorf("hypertension not flattened: %v", got)
	}
	if got := (*calls)[3].body["id"]; got != float64(9) {
		t.Errorf("delete id = %v", got)
	}
}

func TestAppointmentLifecycle(t *testing.T) {
	client, calls := partnerClient(t)
	ctx := context.Background()
	user := bk.Patient{Name: "Ada", Surname: "Lovelace", PhoneNumber: "+905551112233"}

	_, _ = client.Appointments.Reserve(ctx, 1, 2, user)
	_, _ = client.Appointments.Create(ctx, "h", 5)
	_, _ = client.Appointments.List(ctx, "+905551112233", nil, "")
	_, _ = client.Appointments.CancelWithoutSlot(ctx, bk.AppointmentLookup{Hash: "h", OutherProcessID: 5})

	want := []struct{ method, path string }{
		{http.MethodPost, "/outher/reservation"},
		{http.MethodPost, "/outher/appointment"},
		{http.MethodPost, "/outher/appointments"},
		{http.MethodDelete, "/outher/appointmentWithoutSlot"},
	}
	for i, w := range want {
		if (*calls)[i].method != w.method || (*calls)[i].path != w.path {
			t.Errorf("call %d = %s %s, want %s %s", i, (*calls)[i].method, (*calls)[i].path, w.method, w.path)
		}
	}
	if got := (*calls)[0].body["slotId"]; got != float64(1) {
		t.Errorf("slotId = %v", got)
	}
}

func TestRemainingAppointmentEndpoints(t *testing.T) {
	client, calls := partnerClient(t)
	ctx := context.Background()
	user := bk.Patient{Name: "Ada", Surname: "Lovelace", PhoneNumber: "+905551112233"}

	_, _ = client.Appointments.CheckDoctor(ctx, 2, 0)
	_, _ = client.Appointments.ReserveWithoutAgreement(ctx, 1, 2, user)
	_, _ = client.Appointments.InstantReserve(ctx, user)
	_, _ = client.Appointments.CreateWithoutSlot(ctx, bk.AppointmentWithoutSlotInput{
		DoctorID: 2, StartDate: "2026-08-01 09:00", FinishDate: "2026-08-01 09:30", User: user,
	})
	_, _ = client.Appointments.Info(ctx, bk.AppointmentLookup{Hash: "h", OutherProcessID: 5})

	want := []string{
		"/outher/checkDoctor",
		"/outher/reservationWithoutAgreement",
		"/outher/instantReservation",
		"/outher/appointmentWithoutSlot",
		"/outher/appointmentInfo",
	}
	for i, w := range want {
		if (*calls)[i].path != w {
			t.Errorf("call %d path = %q, want %q", i, (*calls)[i].path, w)
		}
	}
}

func TestMeasuresGraphPathAndLegacyTeusanShape(t *testing.T) {
	client, calls := partnerClient(t)
	ctx := context.Background()

	_, _ = client.Measures.Graph(ctx, bk.Patient{PhoneNumber: "+905551112233"}, "weight", 3, nil, nil)
	if (*calls)[0].path != "/outher/measuresGraph/weight/3" {
		t.Errorf("graph path = %q", (*calls)[0].path)
	}

	_, _ = client.Measures.HealthInformation(ctx, "12345678901", "+905551112233",
		[]map[string]any{{"type": "pulse", "date_time": "2026-06-17 09:00", "pulse": 72}})
	if (*calls)[1].path != "/outher/healthInformation" {
		t.Errorf("healthInformation path = %q", (*calls)[1].path)
	}
	// No `patient` wrapper here — this endpoint predates that contract.
	if _, ok := (*calls)[1].body["patient"]; ok {
		t.Errorf("legacy endpoint must stay flat: %v", (*calls)[1].body)
	}
	if got := (*calls)[1].body["identity"]; got != "12345678901" {
		t.Errorf("identity = %v", got)
	}
}

func TestDietsAndLabDetailPaths(t *testing.T) {
	client, calls := partnerClient(t)
	ctx := context.Background()
	ref := bk.Patient{IdentityNumber: "12345678901"}

	_, _ = client.Diets.Detail(ctx, ref, 77)
	_, _ = client.Laboratory.CatalogDetail(ctx, 18246)

	if (*calls)[0].path != "/outher/diet" {
		t.Errorf("diet path = %q", (*calls)[0].path)
	}
	if got := (*calls)[0].body["listId"]; got != float64(77) {
		t.Errorf("listId = %v", got)
	}
	if (*calls)[1].path != "/outher/laboratoryCatalog/18246" {
		t.Errorf("catalogDetail path = %q", (*calls)[1].path)
	}
}
