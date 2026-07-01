package httpserver

import (
	"context"
	"io"
	"log"
	"net/http"
)

// UpdateProcessor handles a raw Telegram update body.
type UpdateProcessor interface {
	ProcessUpdate(ctx context.Context, body []byte)
}

// Logger is the logging capability used by the HTTP transport.
type Logger interface {
	Printf(format string, v ...interface{})
}

// WebhookHandler acknowledges Telegram webhooks immediately and processes them asynchronously.
type WebhookHandler struct {
	processor UpdateProcessor
	logger    Logger
}

func NewWebhookHandler(processor UpdateProcessor, logger Logger) *WebhookHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &WebhookHandler{processor: processor, logger: logger}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "不允许的方法", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Printf("读取请求体失败: %v", err)
		http.Error(w, "内部错误", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	go h.processor.ProcessUpdate(context.Background(), body)
}
