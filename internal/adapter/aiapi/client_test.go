package aiapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"telegram-ai-bot/internal/domain/ai"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestEndpointMatchesOriginalSuffixRules(t *testing.T) {
	tests := []struct {
		base   string
		suffix string
		want   string
	}{
		{"http://127.0.0.1:12345/v1", "/chat/completions", "http://127.0.0.1:12345/v1/chat/completions"},
		{"http://127.0.0.1:12345/v1/", "/chat/completions", "http://127.0.0.1:12345/v1/chat/completions"},
		{"http://127.0.0.1:12345/v1/chat/completions", "/chat/completions", "http://127.0.0.1:12345/v1/chat/completions"},
		{"http://127.0.0.1:12345/v1/chat/completions/", "/chat/completions", "http://127.0.0.1:12345/v1/chat/completions/chat/completions"},
		{"http://127.0.0.1:12345/v1", "/models", "http://127.0.0.1:12345/v1/models"},
		{"http://127.0.0.1:12345/v1", "/chat/session", "http://127.0.0.1:12345/v1/chat/session"},
	}

	for _, tt := range tests {
		client := NewClient(tt.base, "key", nil, nil)
		if got := client.endpoint(tt.suffix); got != tt.want {
			t.Fatalf("endpoint(%q, %q)=%q, want %q", tt.base, tt.suffix, got, tt.want)
		}
	}
}

func TestModelsParsesBodyWithoutStatusCodeGate(t *testing.T) {
	client := NewClient("http://example.test/v1", "key", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"model-a"}]}`)),
		}, nil
	}), nil)

	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	if len(models) != 1 || models[0] != "model-a" {
		t.Fatalf("models=%v, want [model-a]", models)
	}
}

func TestStreamChatReturnsStreamWithoutStatusCodeGate(t *testing.T) {
	client := NewClient("http://example.test/v1", "key", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: [DONE]\n")),
		}, nil
	}), nil)

	stream, err := client.StreamChat(context.Background(), "session", ai.ChatRequest{Model: "smart", Stream: true})
	if err != nil {
		t.Fatalf("StreamChat returned error: %v", err)
	}
	defer stream.Close()

	got, err := stream.Next()
	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("token=%q, want hello", got)
	}
}
