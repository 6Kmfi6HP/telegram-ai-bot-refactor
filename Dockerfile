FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o /out/telegram-ai-bot ./cmd/telegram-ai-bot

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 1000 bot

WORKDIR /app

COPY --from=builder /out/telegram-ai-bot /app/telegram-ai-bot
RUN chmod +x /app/telegram-ai-bot

USER bot

EXPOSE 8080

ENTRYPOINT ["/app/telegram-ai-bot"]
