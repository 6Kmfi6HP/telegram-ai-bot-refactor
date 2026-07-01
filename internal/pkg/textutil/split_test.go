package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitTextSafelyPreservesUTF8(t *testing.T) {
	text := strings.Repeat("你好", 100)
	chunks := SplitTextSafely(text, 17)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	for _, chunk := range chunks {
		if !utf8.ValidString(chunk) {
			t.Fatalf("invalid UTF-8 chunk: %q", chunk)
		}
		if len(chunk) > 17 {
			t.Fatalf("chunk exceeds max bytes: got %d", len(chunk))
		}
	}
	if strings.Join(chunks, "") != text {
		t.Fatal("chunks do not reconstruct original text")
	}
}

func TestSplitTextSafelyPrefersNewline(t *testing.T) {
	text := "first line\nsecond line\nthird line"
	chunks := SplitTextSafely(text, 24)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0] != "first line\nsecond line\n" {
		t.Fatalf("unexpected first chunk: %q", chunks[0])
	}
}
