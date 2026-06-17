package bulutklinik_test

import (
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

func TestInMemoryTokenStore(t *testing.T) {
	store := bk.NewInMemoryTokenStore("a", "r")
	if store.AccessToken() != "a" || store.RefreshToken() != "r" {
		t.Fatalf("seed failed: %q %q", store.AccessToken(), store.RefreshToken())
	}

	store.SetTokens("a2", "r2")
	if store.AccessToken() != "a2" || store.RefreshToken() != "r2" {
		t.Fatalf("set failed: %q %q", store.AccessToken(), store.RefreshToken())
	}

	store.Clear()
	if store.AccessToken() != "" || store.RefreshToken() != "" {
		t.Fatalf("clear failed: %q %q", store.AccessToken(), store.RefreshToken())
	}
}
