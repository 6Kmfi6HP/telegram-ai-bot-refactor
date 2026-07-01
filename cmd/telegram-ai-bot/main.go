package main

import (
	"context"
	"log"

	"telegram-ai-bot/internal/app"
	"telegram-ai-bot/internal/config"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	application := app.New(cfg, log.Default())
	if err := application.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
