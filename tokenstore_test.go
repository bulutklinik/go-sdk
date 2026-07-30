package bulutklinik_test

import (
	"testing"

	bk "github.com/bulutklinik/go-sdk"
)

func TestInMemoryTokenStore(t *testing.T) {
	store := bk.NewInMemoryTokenStore("a")
	if store.Token() != "a" {
		t.Fatalf("seed failed: %q", store.Token())
	}

	store.SetToken("b")
	if store.Token() != "b" {
		t.Fatalf("set failed: %q", store.Token())
	}

	store.Clear()
	if store.Token() != "" {
		t.Fatalf("clear failed: %q", store.Token())
	}
}

func TestInMemoryTokenStoreDefaultsToEmpty(t *testing.T) {
	if got := bk.NewInMemoryTokenStore("").Token(); got != "" {
		t.Errorf("token = %q, want empty", got)
	}
}
