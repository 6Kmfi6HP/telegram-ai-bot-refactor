package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"telegram-ai-bot/internal/adapter/aiapi"
	"telegram-ai-bot/internal/adapter/idgen"
	"telegram-ai-bot/internal/adapter/telegramapi"
	"telegram-ai-bot/internal/config"
	"telegram-ai-bot/internal/transport/httpserver"
	"telegram-ai-bot/internal/usecase/bot"
	"telegram-ai-bot/internal/usecase/session"
)

// App wires adapters, use cases and transports.
type App struct {
	cfg                config.Config
	logger             *log.Logger
	telegramHTTPClient *http.Client
	aiHTTPClient       *http.Client
	telegram           *telegramapi.Client
	ai                 *aiapi.Client
	modelStore         *session.MemoryModelStore
	draftIDGen         *idgen.AtomicDraftGenerator
}

func New(cfg config.Config, logger *log.Logger) *App {
	if logger == nil {
		logger = log.Default()
	}
	return &App{
		cfg:                cfg,
		logger:             logger,
		telegramHTTPClient: &http.Client{},
		aiHTTPClient:       http.DefaultClient,
	}
}

func (a *App) Run(ctx context.Context) error {
	a.telegram = telegramapi.NewClient(a.cfg.TelegramToken, a.telegramHTTPClient)
	a.ai = aiapi.NewClient(a.cfg.APIURL, a.cfg.APIKey, a.aiHTTPClient, a.logger)
	a.modelStore = session.NewMemoryModelStore(a.cfg.DefaultModel)
	a.draftIDGen = idgen.NewAtomicDraftGenerator()

	botInfo, err := a.telegram.GetMe(ctx)
	if err != nil {
		return err
	}
	a.logger.Printf("成功获取机器人信息，用户名：%s", botInfo.UserName)

	if err := a.telegram.SetMyCommands(ctx, bot.DefaultCommands()); err != nil {
		a.logger.Printf("设置命令菜单失败: %v", err)
	} else {
		a.logger.Printf("成功设置命令菜单")
	}

	if a.cfg.WebhookURL == "" {
		return fmt.Errorf("请通过-webhook参数设置webhook URL")
	}
	if err := a.telegram.SetWebhook(ctx, a.cfg.WebhookURL); err != nil {
		return fmt.Errorf("设置webhook失败: %v", err)
	}
	a.logger.Printf("已设置webhook: %s", a.cfg.WebhookURL)

	botService := bot.NewService(a.telegram, a.ai, a.modelStore, a.draftIDGen, botInfo.UserName, a.logger)
	handler := httpserver.NewWebhookHandler(botService, a.logger)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", a.cfg.Port),
		ReadTimeout:  a.cfg.ReadTimeout,
		WriteTimeout: a.cfg.WriteTimeout,
		IdleTimeout:  a.cfg.IdleTimeout,
		Handler:      handler,
	}

	a.logger.Printf("启动HTTP服务器在端口%d...", a.cfg.Port)
	return server.ListenAndServe()
}
