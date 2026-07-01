# Telegram AI Bot

Refactored Go Telegram AI Bot using a Standard Go Project Layout style and Clean Architecture layering. The project uses only the Go standard library.

## Runtime configuration

Configuration uses the original command-line flags.

| Flag | Required | Default | Description |
| --- | --- | --- | --- |
| `-token` | yes | - | Telegram Bot token |
| `-webhook` | yes, checked after Telegram startup | - | Public webhook URL passed to `setWebhook` |
| `-port` | no | `8080` | HTTP webhook server port |
| `-key` | no | `12345` | Bearer token for the OpenAI-compatible API |
| `-model` | no | `smart` | Default model used until a session selects another model |
| `-url` | no | `http://127.0.0.1:12345/v1` | Base OpenAI-compatible API URL |

## Run

```sh
go run ./cmd/telegram-ai-bot \
  -token '123456:telegram-token' \
  -webhook 'https://example.com/webhook' \
  -key 'your-api-key' \
  -url 'http://127.0.0.1:12345/v1' \
  -model 'smart' \
  -port 8080
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
internal/config/              CLI flag configuration
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
- Startup order: validate API key and token, call `getMe`, set command menu, then require `-webhook` before `setWebhook`.
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

## License

No license has been selected yet. Until a `LICENSE` file is added, all rights are reserved by the project owner.
