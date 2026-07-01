package bot

import (
	"testing"

	"telegram-ai-bot/internal/domain/telegram"
)

func TestUserDisplayNamePreservesOriginalPreference(t *testing.T) {
	user := &telegram.User{FirstName: "Alice", UserName: "alice_handle"}

	if got := userDisplayName(user, true); got != "@alice_handle" {
		t.Fatalf("prefer @ display=%q, want @alice_handle", got)
	}
	if got := userDisplayName(user, false); got != "Alice" {
		t.Fatalf("plain display=%q, want Alice", got)
	}
	if got := userDisplayName(&telegram.User{UserName: "alice_handle"}, false); got != "alice_handle" {
		t.Fatalf("username-only plain display=%q, want alice_handle", got)
	}
}
