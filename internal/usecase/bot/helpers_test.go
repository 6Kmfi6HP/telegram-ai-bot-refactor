package bot

import "testing"

func TestGetSessionID(t *testing.T) {
	if got := getSessionID(10, 20); got != "10" {
		t.Fatalf("private session=%q, want 10", got)
	}
	if got := getSessionID(10, -20); got != "10-20" {
		t.Fatalf("group session=%q, want 10-20", got)
	}
}

func TestIsCommandForBot(t *testing.T) {
	if !isCommandForBot("/chat hello", "chat", "mybot") {
		t.Fatal("expected plain command to match")
	}
	if !isCommandForBot("/chat@mybot hello", "chat", "mybot") {
		t.Fatal("expected addressed command to match")
	}
	if isCommandForBot("/chat@other hello", "chat", "mybot") {
		t.Fatal("unexpected command match for other bot")
	}
}

func TestExtractCommandArg(t *testing.T) {
	if got := extractCommandArg("/chat hello world"); got != "hello world" {
		t.Fatalf("arg=%q, want hello world", got)
	}
	if got := extractCommandArg("/chat"); got != "" {
		t.Fatalf("arg=%q, want empty", got)
	}
}
