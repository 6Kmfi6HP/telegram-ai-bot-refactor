package bot

import (
	"context"
	"time"

	"telegram-ai-bot/internal/domain/ai"
	"telegram-ai-bot/internal/domain/telegram"
)

// TelegramClient is the Telegram port required by the bot use case.
type TelegramClient interface {
	SendMessage(ctx context.Context, chatID int, text string, replyToMessageID int) (int, error)
	EditMessage(ctx context.Context, chatID int, messageID int, text string) error
	EditMessageAll(ctx context.Context, chatID int, messageID int, text string) ([]int, error)
	EditMessageWithMarkup(ctx context.Context, chatID int, messageID int, text string, replyMarkup interface{}) error
	AnswerCallbackQuery(ctx context.Context, callbackQueryID string)
	SendMessageDraft(ctx context.Context, chatID int, draftID int, text string) error
	GetFile(ctx context.Context, fileID string) (*telegram.File, error)
	DownloadFile(ctx context.Context, filePath string) ([]byte, error)
	DeleteMessageAfter(chatID int, messageID int, delay time.Duration)
}

// AIClient is the AI provider port required by the bot use case.
type AIClient interface {
	StreamChat(ctx context.Context, sessionID string, chatRequest ai.ChatRequest) (ai.ChatStream, error)
	ResetSession(ctx context.Context, sessionID string) (ai.SessionResetResult, error)
	Models(ctx context.Context) ([]string, error)
}

// ModelStore stores a selected AI model per session.
type ModelStore interface {
	SetModel(sessionID string, model string)
	GetModel(sessionID string) string
}

// DraftIDGenerator generates Telegram draft IDs.
type DraftIDGenerator interface {
	Next() int
}

// Logger is the logging capability used by the use case.
type Logger interface {
	Printf(format string, v ...interface{})
}
