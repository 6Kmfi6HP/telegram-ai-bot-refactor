package bot

import "telegram-ai-bot/internal/domain/telegram"

// DefaultCommands returns the Telegram command menu preserved from the original bot.
func DefaultCommands() []telegram.BotCommand {
	return []telegram.BotCommand{
		{
			Command:     "chat",
			Description: "💬 快捷对话 - /chat 你的问题",
		},
		{
			Command:     "new",
			Description: "🆕 创建新会话 - 清除当前会话上下文",
		},
		{
			Command:     "model",
			Description: "🤖 选择AI模型 - 切换不同的AI模型",
		},
	}
}
