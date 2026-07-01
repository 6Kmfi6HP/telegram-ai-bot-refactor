package session

import "testing"

func TestMemoryModelStore(t *testing.T) {
	store := NewMemoryModelStore("smart")
	if got := store.GetModel("missing"); got != "smart" {
		t.Fatalf("default model=%q, want smart", got)
	}
	store.SetModel("user-1", "gpt-test")
	if got := store.GetModel("user-1"); got != "gpt-test" {
		t.Fatalf("selected model=%q, want gpt-test", got)
	}
}
