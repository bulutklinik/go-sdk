// Command livecheck is a read-only smoke test against the Bulutklinik test
// environment (apitest). It doubles as an end-to-end usage example.
//
// Run: go run ./examples/livecheck
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
	client := bk.NewClient(
		bk.WithEnvironment(bk.Test),
		bk.WithCredentials(
			env("BK_CLIENT_ID", "96b630b3-f62a-4e67-b33c-b58802dca5af"),
			env("BK_CLIENT_SECRET", "KPgmEavOSomEl8mQu1ZZMoyZaVXBSuuKxrrzMAkX"),
		),
	)
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

	total++
	login, err := client.Auth.Connect(ctx, bk.ConnectInput{
		APIUserName:     env("BK_USERNAME", "hackathon@bulutklinik.test"),
		APIUserPassword: env("BK_PASSWORD", "Hackathon2026"),
		LoginMode:       "email",
	})
	if err != nil {
		fmt.Printf("ERR auth.connect: %v\n", err)
	} else {
		pass++
		fmt.Printf("OK  auth.connect (twoFactor=%v, tokenStored=%v)\n", login.TwoFactorRequired, client.TokenStore().AccessToken() != "")
	}

	if b := step("doctors.branches", func() (json.RawMessage, error) { return client.Doctors.Branches(ctx) }); b != nil {
		fmt.Printf("    branches=%d\n", countArray(b))
	}
	if l := step("doctors.locations", func() (json.RawMessage, error) { return client.Doctors.Locations(ctx) }); l != nil {
		fmt.Printf("    locations=%d\n", countArray(l))
	}
	step("doctors.quickSearch", func() (json.RawMessage, error) { return client.Doctors.QuickSearch(ctx, "kardiyo", "interview", "") })
	step("doctors.search", func() (json.RawMessage, error) {
		return client.Doctors.Search(ctx, bk.SearchInput{
			SearchParams: map[string]any{"withFreeText": "kardiyoloji"},
			OrderParams:  []string{"slot"},
			OtherParams:  []string{"isInterviewable"},
			CurrentPage:  1,
			PerPageLimit: 10,
		})
	})

	doctorID := env("BK_DOCTOR_ID", "8282")
	if d := step("doctors.detail", func() (json.RawMessage, error) { return client.Doctors.Detail(ctx, doctorID, nil) }); d != nil {
		fmt.Printf("    detailKeys=%d\n", countMap(d))
	}
	if sl := step("slots.schedule", func() (json.RawMessage, error) {
		return client.Slots.Schedule(ctx, bk.ScheduleInput{DoctorID: doctorID, ListType: "interview"})
	}); sl != nil {
		fmt.Printf("    slotDays=%d\n", countMap(sl))
	}
	if last := step("measures.last", func() (json.RawMessage, error) { return client.Measures.Last(ctx) }); last != nil {
		fmt.Printf("    measuresLastKeys=%d\n", countMap(last))
	}
	step("auth.disconnect", func() (json.RawMessage, error) { return nil, client.Auth.Disconnect(ctx) })

	fmt.Printf("\nSUMMARY: %d/%d steps OK\n", pass, total)
}
