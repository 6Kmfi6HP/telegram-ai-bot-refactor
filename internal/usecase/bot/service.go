package bot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"telegram-ai-bot/internal/domain/ai"
	"telegram-ai-bot/internal/domain/telegram"
	"telegram-ai-bot/internal/pkg/media"
)

// Service orchestrates Telegram updates and AI chat use cases.
type Service struct {
	telegram   TelegramClient
	ai         AIClient
	models     ModelStore
	drafts     DraftIDGenerator
	botUser    string
	logger     Logger
	now        func() time.Time
	printToken func(string)
}

func NewService(telegramClient TelegramClient, aiClient AIClient, modelStore ModelStore, draftGenerator DraftIDGenerator, botUserName string, logger Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		telegram: telegramClient,
		ai:       aiClient,
		models:   modelStore,
		drafts:   draftGenerator,
		botUser:  botUserName,
		logger:   logger,
		now:      time.Now,
		printToken: func(value string) {
			fmt.Print(value)
		},
	}
}

func (s *Service) ProcessUpdate(ctx context.Context, body []byte) {
	startTime := s.now()
	s.logger.Printf("[Webhook 原始消息] %s", string(body))

	var update telegram.Update
	if err := json.Unmarshal(body, &update); err != nil {
		s.logger.Printf("解析更新失败: %v", err)
		return
	}

	if update.CallbackQuery != nil {
		s.processCallbackQuery(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}
	if update.Message.Chat == nil || update.Message.From == nil {
		return
	}
	if update.Message.Text == "" && len(update.Message.Photo) == 0 {
		return
	}

	if s.shouldIgnoreGroupMessage(update.Message) {
		return
	}

	messageText := strings.TrimSpace(update.Message.Text)
	if messageText == "" && len(update.Message.Photo) > 0 {
		messageText = strings.TrimSpace(update.Message.Caption)
	}

	s.logger.Printf("[收到消息] 用户ID: %d, 消息ID: %d, 文字: %s, 含图片: %v",
		update.Message.From.ID, update.Message.MessageID, messageText, len(update.Message.Photo) > 0)

	replyToID := getReplyToMessageID(update.Message.Chat.ID, update.Message.MessageID)

	if strings.HasPrefix(messageText, "/model@"+s.botUser) || strings.HasPrefix(messageText, "/model") {
		s.handleModelCommand(ctx, update, replyToID)
		return
	}
	if strings.HasPrefix(messageText, "/new@"+s.botUser) || strings.HasPrefix(messageText, "/new") {
		s.handleNewSessionCommand(ctx, update, replyToID)
		return
	}
	if isCommandForBot(messageText, "chat", s.botUser) {
		var ok bool
		messageText, ok = s.resolveChatCommandText(ctx, update.Message, messageText, replyToID)
		if !ok {
			return
		}
	}

	imageContent := s.buildImageContent(ctx, update.Message, messageText)

	isPrivateChat := update.Message.Chat.ID > 0
	initialMsgID := 0
	draftID := 0
	if isPrivateChat {
		draftID = s.drafts.Next()
	} else {
		msgID, err := s.telegram.SendMessage(ctx, update.Message.Chat.ID, "**🤖💭 正在处理中，请稍等…**", replyToID)
		if err != nil {
			s.logger.Printf("发送提示消息失败: %v", err)
		} else {
			initialMsgID = msgID
		}
	}

	messageText = s.applyReplyContext(update.Message, messageText)

	if !isPrivateChat && initialMsgID != 0 {
		_ = s.telegram.EditMessage(ctx, update.Message.Chat.ID, initialMsgID, "**🤖 正在思考中，请稍等...**")
	}

	messages := []ai.ChatMessage{}
	s.buildReplyChain(update.Message, &messages)
	if len(imageContent) > 0 {
		messages = append(messages, ai.ChatMessage{
			Role:    "user",
			Content: imageContent,
		})
	} else {
		messages = append(messages, ai.ChatMessage{
			Role:    "user",
			Content: messageText,
		})
	}

	sessionID := getSessionID(update.Message.From.ID, update.Message.Chat.ID)
	chatRequest := ai.ChatRequest{
		Model:    s.models.GetModel(sessionID),
		Messages: messages,
		Stream:   true,
	}

	stream, err := s.ai.StreamChat(ctx, sessionID, chatRequest)
	if err != nil {
		s.logger.Printf("%v", err)
		return
	}
	defer stream.Close()

	finalContent, err := s.consumeStream(ctx, update.Message.Chat.ID, initialMsgID, draftID, isPrivateChat, stream)
	if err != nil {
		s.logger.Printf("%v", err)
		return
	}

	s.finalizeResponse(ctx, update.Message.Chat.ID, update.Message.MessageID, replyToID, initialMsgID, isPrivateChat, finalContent)

	processTime := time.Since(startTime)
	s.logger.Printf("[处理完成] 耗时: %v", processTime)
}

func (s *Service) processCallbackQuery(ctx context.Context, query *telegram.CallbackQuery) {
	if query == nil || query.Message == nil || query.Message.Chat == nil || query.From == nil {
		return
	}

	data := strings.TrimSpace(query.Data)
	if !strings.HasPrefix(data, "model:") {
		return
	}

	modelName := strings.TrimSpace(strings.TrimPrefix(data, "model:"))
	if modelName == "" {
		s.telegram.AnswerCallbackQuery(ctx, query.ID)
		return
	}

	sessionID := getSessionID(query.From.ID, query.Message.Chat.ID)
	s.models.SetModel(sessionID, modelName)
	s.telegram.AnswerCallbackQuery(ctx, query.ID)

	selectedModel := s.models.GetModel(sessionID)
	_ = s.telegram.EditMessage(ctx, query.Message.Chat.ID, query.Message.MessageID,
		fmt.Sprintf("**✅ 模型已切换为：** `%s`", selectedModel))
	s.telegram.DeleteMessageAfter(query.Message.Chat.ID, query.Message.MessageID, 30*time.Second)
}

func (s *Service) shouldIgnoreGroupMessage(message *telegram.Message) bool {
	if message.Chat.ID >= 0 {
		return false
	}

	checkText := message.Text
	if checkText == "" && len(message.Photo) > 0 {
		checkText = message.Caption
	}
	trimmedText := strings.TrimSpace(checkText)
	isReplyToBot := message.ReplyToMessage != nil && message.ReplyToMessage.From != nil && message.ReplyToMessage.From.UserName == s.botUser
	isAllowedCommand := isCommandForBot(trimmedText, "chat", s.botUser) || isCommandForBot(trimmedText, "model", s.botUser) || isCommandForBot(trimmedText, "new", s.botUser)

	if !strings.Contains(checkText, "@"+s.botUser) && !isReplyToBot && !isAllowedCommand {
		s.logger.Printf("群组消息未包含 @%s 且不是回复机器人的消息，忽略回复", s.botUser)
		return true
	}
	return false
}

func (s *Service) resolveChatCommandText(ctx context.Context, message *telegram.Message, messageText string, replyToID int) (string, bool) {
	query := extractCommandArg(messageText)
	query = strings.ReplaceAll(query, "@"+s.botUser, "")
	query = strings.TrimSpace(query)
	if query != "" {
		return query, true
	}

	if message.ReplyToMessage == nil {
		_, _ = s.telegram.SendMessage(ctx, message.Chat.ID, "**用法：** /chat 你的问题", replyToID)
		return "", false
	}

	replyText := strings.TrimSpace(message.ReplyToMessage.Text)
	if replyText == "" {
		replyText = strings.TrimSpace(message.ReplyToMessage.Caption)
	}
	if replyText == "" {
		_, _ = s.telegram.SendMessage(ctx, message.Chat.ID, "**用法：** /chat 你的问题", replyToID)
		return "", false
	}

	if replyUserName := userDisplayName(message.ReplyToMessage.From, true); replyUserName != "" {
		replyText = fmt.Sprintf("[%s]: %s", replyUserName, replyText)
	}
	return replyText, true
}

func (s *Service) buildImageContent(ctx context.Context, message *telegram.Message, messageText string) []ai.ContentPart {
	photo := message.Photo
	if len(photo) == 0 && message.ReplyToMessage != nil && len(message.ReplyToMessage.Photo) > 0 {
		photo = message.ReplyToMessage.Photo
	}
	if len(photo) == 0 {
		return nil
	}

	largestPhoto := photo[len(photo)-1]
	s.logger.Printf("[图片消息] file_id: %s, 尺寸: %dx%d", largestPhoto.FileID, largestPhoto.Width, largestPhoto.Height)
	fileInfo, err := s.telegram.GetFile(ctx, largestPhoto.FileID)
	if err != nil {
		s.logger.Printf("获取图片文件信息失败: %v", err)
		return nil
	}

	imageData, err := s.telegram.DownloadFile(ctx, fileInfo.FilePath)
	if err != nil {
		s.logger.Printf("下载图片失败: %v", err)
		return nil
	}

	mimeType := media.MIMETypeFromExt(fileInfo.FilePath)
	base64Data := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
	s.logger.Printf("[图片消息] 已下载并编码为 base64 (%d bytes, %s)", len(imageData), mimeType)

	imgText := messageText
	if imgText == "" {
		imgText = "请描述这张图片"
	}
	return []ai.ContentPart{
		{Type: "text", Text: imgText},
		{Type: "image_url", ImageURL: &ai.ImageURLRef{URL: dataURL}},
	}
}

func (s *Service) applyReplyContext(message *telegram.Message, messageText string) string {
	if message.Chat.ID < 0 {
		return s.applyGroupReplyContext(message, messageText)
	}
	return s.applyPrivateReplyContext(message, messageText)
}

func (s *Service) applyGroupReplyContext(message *telegram.Message, messageText string) string {
	if message.ReplyToMessage == nil {
		return messageText
	}

	cleanText := strings.ReplaceAll(messageText, "@"+s.botUser, "")
	cleanText = strings.TrimSpace(cleanText)

	if message.ReplyToMessage.From != nil && message.ReplyToMessage.From.UserName == s.botUser {
		return messageText
	}

	replyMsg := message.ReplyToMessage
	replyContent := ""
	if replyMsg.Text != "" {
		replyUserName := userDisplayName(replyMsg.From, true)
		replyContent = fmt.Sprintf("[%s]: %s", replyUserName, replyMsg.Text)
	}

	if cleanText != "" {
		if replyContent != "" {
			messageText = fmt.Sprintf("%s\n\n用户补充：%s", replyContent, cleanText)
		} else {
			messageText = cleanText
		}
		s.logger.Printf("[群聊-使用被回复消息+用户描述] %s", messageText)
	} else if replyContent != "" {
		messageText = replyContent
		s.logger.Printf("[群聊-使用被回复消息作为问题] %s", messageText)
	}
	return messageText
}

func (s *Service) applyPrivateReplyContext(message *telegram.Message, messageText string) string {
	if message.ReplyToMessage == nil || message.ReplyToMessage.From == nil || message.ReplyToMessage.From.UserName != s.botUser {
		return messageText
	}

	cleanText := strings.TrimSpace(messageText)
	if cleanText == "" {
		return messageText
	}

	replyMsg := message.ReplyToMessage
	if replyMsg.Text != "" {
		messageText = fmt.Sprintf("用户补充：%s\n\n---以下是之前的对话---\n%s", cleanText, replyMsg.Text)
	} else {
		messageText = cleanText
	}
	s.logger.Printf("[私聊-回复机器人并补充] %s", messageText)
	return messageText
}

func (s *Service) consumeStream(ctx context.Context, chatID int, initialMsgID int, draftID int, isPrivateChat bool, stream ai.ChatStream) (string, error) {
	var contentBuilder strings.Builder
	lastUpdateTime := s.now()
	var lastSentContent string
	var draftStarted bool

	for {
		blockContent, err := stream.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if blockContent == "" {
			continue
		}

		contentBuilder.WriteString(blockContent)
		s.printToken(blockContent)

		timeSinceLastUpdate := time.Since(lastUpdateTime)
		currentContent := contentBuilder.String()

		if isPrivateChat {
			if !draftStarted {
				if err := s.telegram.SendMessageDraft(ctx, chatID, draftID, ""); err != nil {
					s.logger.Printf("发送草稿失败: %v", err)
				}
				draftStarted = true
				lastUpdateTime = s.now()
				lastSentContent = ""
			} else if timeSinceLastUpdate > 200*time.Millisecond && currentContent != lastSentContent {
				if err := s.telegram.SendMessageDraft(ctx, chatID, draftID, currentContent); err != nil {
					s.logger.Printf("更新草稿失败: %v", err)
				}
				lastUpdateTime = s.now()
				lastSentContent = currentContent
			}
		} else if initialMsgID != 0 && timeSinceLastUpdate > 1500*time.Millisecond && currentContent != lastSentContent {
			if err := s.telegram.EditMessage(ctx, chatID, initialMsgID, currentContent); err != nil {
				if !strings.Contains(err.Error(), "message is not modified") {
					s.logger.Printf("流式更新消息失败: %v", err)
				}
			}
			lastUpdateTime = s.now()
			lastSentContent = currentContent
		}
	}

	finalContent := strings.TrimSpace(contentBuilder.String())
	if endIdx := strings.Index(finalContent, "</think>"); endIdx != -1 {
		finalContent = strings.TrimSpace(finalContent[endIdx+len("</think>"):])
	}
	return finalContent, nil
}

func (s *Service) finalizeResponse(ctx context.Context, chatID int, sourceMessageID int, replyToID int, initialMsgID int, isPrivateChat bool, finalContent string) {
	if finalContent == "" {
		defaultReply := "**🤖 当前AI繁忙，请稍后重试！**"
		if isPrivateChat {
			_, _ = s.telegram.SendMessage(ctx, chatID, defaultReply, replyToID)
		} else if initialMsgID != 0 {
			allIDs, err := s.telegram.EditMessageAll(ctx, chatID, initialMsgID, defaultReply)
			if err != nil {
				_, _ = s.telegram.SendMessage(ctx, chatID, err.Error(), replyToID)
			} else {
				s.deleteMessagesAfter(chatID, sourceMessageID, allIDs, 3*time.Minute)
			}
		} else {
			_, _ = s.telegram.SendMessage(ctx, chatID, defaultReply, replyToID)
		}
		return
	}

	if isPrivateChat {
		_, _ = s.telegram.SendMessage(ctx, chatID, finalContent, replyToID)
		return
	}

	if initialMsgID != 0 {
		allIDs, err := s.telegram.EditMessageAll(ctx, chatID, initialMsgID, finalContent)
		if err != nil {
			_, _ = s.telegram.SendMessage(ctx, chatID, err.Error(), replyToID)
		} else {
			s.deleteMessagesAfter(chatID, sourceMessageID, allIDs, 3*time.Minute)
		}
		return
	}

	_, _ = s.telegram.SendMessage(ctx, chatID, finalContent, replyToID)
}

func (s *Service) deleteMessagesAfter(chatID int, sourceMessageID int, generatedIDs []int, delay time.Duration) {
	for _, id := range generatedIDs {
		s.telegram.DeleteMessageAfter(chatID, id, delay)
	}
	s.telegram.DeleteMessageAfter(chatID, sourceMessageID, delay)
}

func (s *Service) handleNewSessionCommand(ctx context.Context, update telegram.Update, replyToID int) {
	if update.Message == nil || update.Message.Chat == nil || update.Message.From == nil {
		return
	}

	sessionID := getSessionID(update.Message.From.ID, update.Message.Chat.ID)
	initialMsgID, err := s.telegram.SendMessage(ctx, update.Message.Chat.ID, "**🗑️ 正在清除会话上下文...**", replyToID)
	if err != nil {
		return
	}

	result, err := s.ai.ResetSession(ctx, sessionID)
	if err != nil {
		_ = s.telegram.EditMessage(ctx, update.Message.Chat.ID, initialMsgID,
			fmt.Sprintf("**❌ 清除会话失败：** %v", err))
		return
	}

	if !result.ParsedJSON {
		if result.Success {
			s.editSessionCleared(ctx, update, initialMsgID)
		} else {
			_ = s.telegram.EditMessage(ctx, update.Message.Chat.ID, initialMsgID,
				fmt.Sprintf("**❌ 清除会话失败**\n\n状态码：%d\n响应：%s", result.StatusCode, result.Body))
			s.deleteCommandMessagesAfter(update, initialMsgID, 30*time.Second)
		}
		return
	}

	if result.Success {
		s.editSessionCleared(ctx, update, initialMsgID)
	} else {
		_ = s.telegram.EditMessage(ctx, update.Message.Chat.ID, initialMsgID,
			fmt.Sprintf("**❌ 清除会话失败**\n\n%s", result.Message))
		s.deleteCommandMessagesAfter(update, initialMsgID, 30*time.Second)
	}
}

func (s *Service) editSessionCleared(ctx context.Context, update telegram.Update, initialMsgID int) {
	_ = s.telegram.EditMessage(ctx, update.Message.Chat.ID, initialMsgID,
		"**✅ 已创建新会话**\n\n会话上下文已清除，您可以开始新的对话了。")
	s.deleteCommandMessagesAfter(update, initialMsgID, 30*time.Second)
}

func (s *Service) deleteCommandMessagesAfter(update telegram.Update, initialMsgID int, delay time.Duration) {
	s.telegram.DeleteMessageAfter(update.Message.Chat.ID, initialMsgID, delay)
	s.telegram.DeleteMessageAfter(update.Message.Chat.ID, update.Message.MessageID, delay)
}

func (s *Service) handleModelCommand(ctx context.Context, update telegram.Update, replyToID int) {
	if update.Message == nil || update.Message.Chat == nil || update.Message.From == nil {
		return
	}

	sessionID := getSessionID(update.Message.From.ID, update.Message.Chat.ID)
	currentModel := s.models.GetModel(sessionID)

	initialMsgID, err := s.telegram.SendMessage(ctx, update.Message.Chat.ID, "**🤖 正在获取可用模型列表...**", replyToID)
	if err != nil {
		return
	}

	models, err := s.ai.Models(ctx)
	if err != nil {
		_ = s.telegram.EditMessage(ctx, update.Message.Chat.ID, initialMsgID,
			fmt.Sprintf("**❌ 获取模型列表失败：** %v\n\n当前使用模型：`%s`", err, currentModel))
		s.deleteCommandMessagesAfter(update, initialMsgID, 30*time.Second)
		return
	}

	if len(models) == 0 {
		_ = s.telegram.EditMessage(ctx, update.Message.Chat.ID, initialMsgID,
			fmt.Sprintf("**⚠️ 未获取到可用模型**\n\n当前使用模型：`%s`", currentModel))
		s.deleteCommandMessagesAfter(update, initialMsgID, 30*time.Second)
		return
	}

	var keyboard telegram.InlineKeyboardMarkup
	for _, model := range models {
		displayName := model
		if len(model) > 30 {
			displayName = model[:30] + "..."
		}
		row := []telegram.InlineKeyboardButton{{
			Text:         displayName,
			CallbackData: "model:" + model,
		}}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	}

	_ = s.telegram.EditMessageWithMarkup(ctx, update.Message.Chat.ID, initialMsgID,
		fmt.Sprintf("**📋 选择 AI 模型**\n\n当前使用：`%s`\n\n点击下方按钮切换模型：", currentModel), keyboard)
	s.deleteCommandMessagesAfter(update, initialMsgID, 30*time.Second)
}

func (s *Service) buildReplyChain(msg *telegram.Message, messages *[]ai.ChatMessage) {
	if msg == nil || msg.ReplyToMessage == nil {
		return
	}

	s.buildReplyChain(msg.ReplyToMessage, messages)

	replyMsg := msg.ReplyToMessage
	if replyMsg.Text == "" {
		return
	}

	role := "user"
	if replyMsg.From != nil && replyMsg.From.UserName == s.botUser {
		role = "assistant"
	}

	content := strings.TrimSpace(replyMsg.Text)
	if role == "user" {
		if userName := userDisplayName(replyMsg.From, false); userName != "" {
			content = fmt.Sprintf("[%s]: %s", userName, content)
		}
	}

	*messages = append(*messages, ai.ChatMessage{
		Role:    role,
		Content: content,
	})
}
