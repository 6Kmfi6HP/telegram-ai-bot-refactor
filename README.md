# Telegram AI Bot

Refactored Go Telegram AI Bot using a Standard Go Project Layout style and Clean Architecture layering. The project uses only the Go standard library.

## Runtime configuration

Configuration is loaded from environment variables.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `TELEGRAM_BOT_TOKEN` | yes | - | Telegram Bot token |
| `TELEGRAM_WEBHOOK_URL` | yes | - | Public webhook URL passed to `setWebhook` |
| `PORT` | no | `8080` | HTTP webhook server port |
| `AI_API_KEY` | no | `12345` | Bearer token for the OpenAI-compatible API |
| `AI_MODEL` | no | `smart` | Default model used until a session selects another model |
| `AI_API_URL` | no | `http://127.0.0.1:12345/v1` | Base OpenAI-compatible API URL |
| `HTTP_READ_TIMEOUT` | no | `60s` | HTTP server read timeout |
| `HTTP_WRITE_TIMEOUT` | no | `60s` | HTTP server write timeout |
| `HTTP_IDLE_TIMEOUT` | no | `120s` | HTTP server idle timeout |

## Run

```sh
export TELEGRAM_BOT_TOKEN='123456:telegram-token'
export TELEGRAM_WEBHOOK_URL='https://example.com/webhook'
export AI_API_KEY='your-api-key'
export AI_API_URL='http://127.0.0.1:12345/v1'
export AI_MODEL='smart'
export PORT='8080'

go run ./cmd/telegram-ai-bot
```

## Verify

```sh
go test ./...
go build ./...
go vet ./...
```

## Layout

```text
cmd/telegram-ai-bot/          executable entrypoint
internal/app/                 application wiring/bootstrap
internal/config/              environment configuration
internal/domain/ai/           AI request/response domain types
internal/domain/telegram/     Telegram domain/API payload types
internal/usecase/bot/         bot orchestration use case and ports
internal/usecase/session/     in-memory session model store
internal/adapter/telegramapi/ Telegram Bot API adapter
internal/adapter/aiapi/       OpenAI-compatible API adapter
internal/adapter/idgen/       atomic draft ID generator
internal/transport/httpserver/webhook HTTP webhook transport
internal/pkg/media/           MIME type helper
internal/pkg/textutil/        UTF-8 safe Telegram message splitter
```

## Preserved behavior

- Telegram webhook handling with immediate HTTP 200 acknowledgement and asynchronous update processing.
- Group-chat filtering unless the bot is mentioned, replied to, or called through `/chat`, `/new`, or `/model`.
- `/chat` command argument extraction and reply-message fallback.
- `/new` AI session reset using `DELETE /chat/session` and `X-Session-Id`.
- `/model` model list retrieval from `/models`, inline keyboard selection, and per-session model storage.
- Telegram Markdown send/edit with plain-text fallback.
- Telegram 4096-byte message splitting with UTF-8 boundary protection.
- Private-chat streaming through `sendMessageDraft`; group-chat streaming through periodic `editMessageText`.
- Image handling through Telegram `getFile`, file download, MIME detection, base64 data URL construction, and multimodal OpenAI-compatible message content.
- Reply chain reconstruction into user/assistant chat messages.
- Final `</think>` stripping before sending the answer.
