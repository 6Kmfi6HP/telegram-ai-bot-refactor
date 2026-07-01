package telegramapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"telegram-ai-bot/internal/domain/telegram"
	"telegram-ai-bot/internal/pkg/textutil"
)

const (
	baseURL          = "https://api.telegram.org/bot%s/%s"
	fileBaseURL      = "https://api.telegram.org/file/bot%s/%s"
	maxMessageLength = 4096
)

// HTTPDoer is the subset of http.Client used by the Telegram adapter.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a zero-dependency Telegram Bot API client for the bot's required endpoints.
type Client struct {
	token      string
	httpClient HTTPDoer
	mutex      sync.Mutex
}

func NewClient(token string, httpClient HTTPDoer) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		token:      token,
		httpClient: httpClient,
	}
}

func (c *Client) makeRequest(ctx context.Context, method string, endpoint string, params interface{}) (*telegram.APIResponse, error) {
	requestURL := fmt.Sprintf(baseURL, c.token, endpoint)
	var body io.Reader

	if params != nil {
		jsonData, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("编码参数失败: %v", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	if params != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.mutex.Lock()
	resp, err := c.httpClient.Do(req)
	c.mutex.Unlock()
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResp telegram.APIResponse
	if err := json.Unmarshal(responseBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	if !apiResp.Ok {
		return &apiResp, fmt.Errorf("API错误: %s", apiResp.Description)
	}
	return &apiResp, nil
}

func (c *Client) SetWebhook(ctx context.Context, webhookURL string) error {
	params := map[string]interface{}{"url": webhookURL}
	_, err := c.makeRequest(ctx, http.MethodPost, "setWebhook", params)
	return err
}

func (c *Client) DeleteWebhook(ctx context.Context) error {
	_, err := c.makeRequest(ctx, http.MethodPost, "deleteWebhook", nil)
	return err
}

func (c *Client) GetMe(ctx context.Context) (*telegram.User, error) {
	apiResp, err := c.makeRequest(ctx, http.MethodGet, "getMe", nil)
	if err != nil {
		return nil, fmt.Errorf("获取机器人信息失败: %v", err)
	}
	var user telegram.User
	if err := json.Unmarshal(apiResp.Result, &user); err != nil {
		return nil, fmt.Errorf("解析机器人信息失败: %v", err)
	}
	return &user, nil
}

func (c *Client) SetMyCommands(ctx context.Context, commands []telegram.BotCommand) error {
	params := map[string]interface{}{"commands": commands}
	_, err := c.makeRequest(ctx, http.MethodPost, "setMyCommands", params)
	if err != nil {
		return fmt.Errorf("设置命令菜单失败: %v", err)
	}
	return nil
}

func (c *Client) EditMessage(ctx context.Context, chatID int, messageID int, text string) error {
	_, err := c.EditMessageAll(ctx, chatID, messageID, text)
	return err
}

func (c *Client) EditMessageAll(ctx context.Context, chatID int, messageID int, text string) ([]int, error) {
	msgIDs := []int{messageID}

	if len(text) > maxMessageLength {
		chunks := textutil.SplitTextSafely(text, maxMessageLength)
		if len(chunks) == 0 {
			return msgIDs, fmt.Errorf("分割文本失败")
		}

		firstChunkParams := map[string]interface{}{
			"chat_id":    chatID,
			"message_id": messageID,
			"text":       chunks[0],
			"parse_mode": "Markdown",
		}
		if _, err := c.makeRequest(ctx, http.MethodPost, "editMessageText", firstChunkParams); err != nil {
			delete(firstChunkParams, "parse_mode")
			_, _ = c.makeRequest(ctx, http.MethodPost, "editMessageText", firstChunkParams)
		}

		lastMessageID := messageID
		for i := 1; i < len(chunks); i++ {
			msgID, err := c.sendSingleMessage(ctx, chatID, chunks[i], lastMessageID)
			if err != nil {
				return msgIDs, err
			}
			msgIDs = append(msgIDs, msgID)
			lastMessageID = msgID
		}
		return msgIDs, nil
	}

	params := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	_, err := c.makeRequest(ctx, http.MethodPost, "editMessageText", params)
	if err != nil {
		if strings.Contains(err.Error(), "Too Many Requests") {
			return msgIDs, err
		}
		log.Printf("使用Markdown格式编辑消息失败: %v，尝试使用纯文本格式", err)
		delete(params, "parse_mode")
		_, err = c.makeRequest(ctx, http.MethodPost, "editMessageText", params)
	}
	return msgIDs, err
}

func (c *Client) SendMessage(ctx context.Context, chatID int, text string, replyToMessageID int) (int, error) {
	msgIDs, err := c.SendMessageAll(ctx, chatID, text, replyToMessageID)
	if err != nil {
		return 0, err
	}
	if len(msgIDs) == 0 {
		return 0, nil
	}
	return msgIDs[len(msgIDs)-1], nil
}

func (c *Client) SendMessageAll(ctx context.Context, chatID int, text string, replyToMessageID int) ([]int, error) {
	if len(text) <= maxMessageLength {
		msgID, err := c.sendSingleMessage(ctx, chatID, text, replyToMessageID)
		if err != nil {
			return nil, err
		}
		return []int{msgID}, nil
	}

	var msgIDs []int
	isFirstMessage := true
	chunks := textutil.SplitTextSafely(text, maxMessageLength)

	for _, chunk := range chunks {
		replyTo := replyToMessageID
		if !isFirstMessage {
			replyTo = 0
		}

		msgID, err := c.sendSingleMessage(ctx, chatID, chunk, replyTo)
		if err != nil {
			return msgIDs, err
		}
		msgIDs = append(msgIDs, msgID)
		isFirstMessage = false
	}

	return msgIDs, nil
}

func (c *Client) SendMessageWithMarkup(ctx context.Context, chatID int, text string, replyToMessageID int, replyMarkup interface{}) (int, error) {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	if replyToMessageID != 0 {
		params["reply_to_message_id"] = replyToMessageID
	}
	if replyMarkup != nil {
		params["reply_markup"] = replyMarkup
	}
	apiResp, err := c.makeRequest(ctx, http.MethodPost, "sendMessage", params)
	if err != nil {
		if strings.Contains(err.Error(), "Too Many Requests") {
			return 0, err
		}
		delete(params, "parse_mode")
		apiResp, err = c.makeRequest(ctx, http.MethodPost, "sendMessage", params)
		if err != nil {
			return 0, err
		}
	}
	var sentMsg telegram.Message
	if err := json.Unmarshal(apiResp.Result, &sentMsg); err != nil {
		return 0, fmt.Errorf("解析发送消息响应失败: %v", err)
	}
	return sentMsg.MessageID, nil
}

func (c *Client) EditMessageWithMarkup(ctx context.Context, chatID int, messageID int, text string, replyMarkup interface{}) error {
	if len(text) > maxMessageLength {
		return c.EditMessage(ctx, chatID, messageID, text)
	}
	params := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	if replyMarkup != nil {
		params["reply_markup"] = replyMarkup
	}
	_, err := c.makeRequest(ctx, http.MethodPost, "editMessageText", params)
	if err != nil {
		if strings.Contains(err.Error(), "Too Many Requests") {
			return err
		}
		delete(params, "parse_mode")
		_, err = c.makeRequest(ctx, http.MethodPost, "editMessageText", params)
	}
	return err
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID string) {
	if callbackQueryID == "" {
		return
	}
	params := map[string]interface{}{"callback_query_id": callbackQueryID}
	_, _ = c.makeRequest(ctx, http.MethodPost, "answerCallbackQuery", params)
}

func (c *Client) SendMessageDraft(ctx context.Context, chatID int, draftID int, text string) error {
	params := map[string]interface{}{
		"chat_id":  chatID,
		"draft_id": draftID,
		"text":     text,
	}
	_, err := c.makeRequest(ctx, http.MethodPost, "sendMessageDraft", params)
	return err
}

func (c *Client) GetFile(ctx context.Context, fileID string) (*telegram.File, error) {
	params := map[string]interface{}{"file_id": fileID}
	apiResp, err := c.makeRequest(ctx, http.MethodPost, "getFile", params)
	if err != nil {
		return nil, err
	}
	var file telegram.File
	if err := json.Unmarshal(apiResp.Result, &file); err != nil {
		return nil, fmt.Errorf("解析 getFile 响应失败: %v", err)
	}
	return &file, nil
}

func (c *Client) DownloadFile(ctx context.Context, filePath string) ([]byte, error) {
	_ = ctx
	requestURL := fmt.Sprintf(fileBaseURL, c.token, filePath)
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %v", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) DeleteMessage(ctx context.Context, chatID int, messageID int) error {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	_, err := c.makeRequest(ctx, http.MethodPost, "deleteMessage", params)
	return err
}

func (c *Client) DeleteMessageAfter(chatID int, messageID int, delay time.Duration) {
	time.AfterFunc(delay, func() {
		if err := c.DeleteMessage(context.Background(), chatID, messageID); err != nil {
			log.Printf("自动删除消息失败 (chat=%d, msg=%d): %v", chatID, messageID, err)
		}
	})
}

func (c *Client) sendSingleMessage(ctx context.Context, chatID int, text string, replyToMessageID int) (int, error) {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	if replyToMessageID != 0 {
		params["reply_to_message_id"] = replyToMessageID
	}
	apiResp, err := c.makeRequest(ctx, http.MethodPost, "sendMessage", params)
	if err != nil {
		if strings.Contains(err.Error(), "Too Many Requests") {
			return 0, err
		}
		log.Printf("使用Markdown格式发送消息失败: %v，尝试使用纯文本格式", err)
		delete(params, "parse_mode")
		apiResp, err = c.makeRequest(ctx, http.MethodPost, "sendMessage", params)
		if err != nil {
			return 0, err
		}
	}
	var sentMsg telegram.Message
	if err := json.Unmarshal(apiResp.Result, &sentMsg); err != nil {
		return 0, fmt.Errorf("解析发送消息响应失败: %v", err)
	}
	return sentMsg.MessageID, nil
}
