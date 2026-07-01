package bot

import (
	"fmt"
	"strings"

	"telegram-ai-bot/internal/domain/telegram"
)

func getReplyToMessageID(chatID int, messageID int) int {
	if chatID > 0 {
		return 0
	}
	return messageID
}

func getSessionID(fromID int, chatID int) string {
	sessionID := fmt.Sprintf("%d", fromID)
	if chatID < 0 {
		sessionID = fmt.Sprintf("%d%d", fromID, chatID)
	}
	return sessionID
}

func isCommandForBot(text string, command string, botUserName string) bool {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/"+command) {
		return false
	}
	tokenEnd := len(text)
	if idx := strings.IndexAny(text, " \n\t"); idx != -1 {
		tokenEnd = idx
	}
	token := text[:tokenEnd]
	return token == "/"+command || token == "/"+command+"@"+botUserName
}

func extractCommandArg(text string) string {
	text = strings.TrimSpace(text)
	tokenEnd := len(text)
	if idx := strings.IndexAny(text, " \n\t"); idx != -1 {
		tokenEnd = idx
	}
	if tokenEnd >= len(text) {
		return ""
	}
	return strings.TrimSpace(text[tokenEnd:])
}

func messageChatID(msg *telegram.Message) int {
	if msg == nil || msg.Chat == nil {
		return 0
	}
	return msg.Chat.ID
}

func messageFromID(msg *telegram.Message) int {
	if msg == nil || msg.From == nil {
		return 0
	}
	return msg.From.ID
}

func userDisplayName(user *telegram.User, preferUsernameAt bool) string {
	if user == nil {
		return ""
	}
	if preferUsernameAt && user.UserName != "" {
		return "@" + user.UserName
	}
	if user.FirstName != "" {
		return user.FirstName
	}
	if user.UserName != "" {
		if preferUsernameAt {
			return "@" + user.UserName
		}
		return user.UserName
	}
	return ""
}
