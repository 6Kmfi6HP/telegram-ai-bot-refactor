package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"telegram-ai-bot/internal/config"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRunChecksWebhookAfterTelegramStartup(t *testing.T) {
	var calls []string
	application := New(config.Config{
		TelegramToken: "token",
		Port:          8080,
		APIKey:        "key",
		DefaultModel:  "smart",
		APIURL:        "http://ai.test/v1",
		ReadTimeout:   time.Minute,
		WriteTimeout:  time.Minute,
		IdleTimeout:   time.Minute,
	}, log.New(io.Discard, "", 0))
	application.telegramHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.URL.Path)
		switch {
		case strings.HasSuffix(req.URL.Path, "/getMe"):
			return jsonResponse(`{"ok":true,"result":{"id":1,"username":"mybot","first_name":"Bot"}}`), nil
		case strings.HasSuffix(req.URL.Path, "/setMyCommands"):
			return jsonResponse(`{"ok":true,"result":true}`), nil
		default:
			return nil, fmt.Errorf("unexpected Telegram call: %s", req.URL.Path)
		}
	})}

	err := application.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "-webhook") {
		t.Fatalf("Run error=%v, want missing -webhook", err)
	}

	want := []string{"/bottoken/getMe", "/bottoken/setMyCommands"}
	if strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls=%v, want %v", calls, want)
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
