package aiapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"telegram-ai-bot/internal/domain/ai"
)

// HTTPDoer is the subset of http.Client used by the AI adapter.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client talks to an OpenAI-compatible API.
type Client struct {
	apiURL     string
	apiKey     string
	httpClient HTTPDoer
	logger     Logger
}

// Logger is the logging capability used by the adapter.
type Logger interface {
	Printf(format string, v ...interface{})
}

func NewClient(apiURL string, apiKey string, httpClient HTTPDoer, logger Logger) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Client{
		apiURL:     apiURL,
		apiKey:     apiKey,
		httpClient: httpClient,
		logger:     logger,
	}
}

func (c *Client) StreamChat(ctx context.Context, sessionID string, chatRequest ai.ChatRequest) (ai.ChatStream, error) {
	requestBody, err := json.Marshal(chatRequest)
	if err != nil {
		return nil, fmt.Errorf("编码请求失败: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/chat/completions"), bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-Session-Id", sessionID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求AI API失败: %v", err)
	}

	scanner := bufio.NewScanner(resp.Body)
	return &sseChatStream{
		body:    resp.Body,
		scanner: scanner,
		logger:  c.logger,
	}, nil
}

func (c *Client) ResetSession(ctx context.Context, sessionID string) (ai.SessionResetResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint("/chat/session"), nil)
	if err != nil {
		return ai.SessionResetResult{}, err
	}
	req.Header.Set("X-Session-Id", sessionID)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ai.SessionResetResult{}, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	result := ai.SessionResetResult{
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
	}

	var parsed struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		result.ParsedJSON = false
		result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
		return result, nil
	}

	result.ParsedJSON = true
	result.Success = parsed.Success
	result.Message = parsed.Message
	return result, nil
}

func (c *Client) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("/models"), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		c.logger.Printf("[获取模型列表原始响应] %s", string(body))
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, model := range result.Data {
		models = append(models, model.ID)
	}
	return models, nil
}

func (c *Client) endpoint(suffix string) string {
	endpointURL := c.apiURL
	if !strings.HasSuffix(endpointURL, suffix) {
		if strings.HasSuffix(endpointURL, "/") {
			endpointURL += strings.TrimPrefix(suffix, "/")
		} else {
			endpointURL += suffix
		}
	}
	return endpointURL
}

type sseChatStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	logger  Logger
}

func (s *sseChatStream) Next() (string, error) {
	for s.scanner.Scan() {
		line := s.scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		jsonData := strings.TrimPrefix(line, "data: ")
		if jsonData == "[DONE]" {
			return "", io.EOF
		}

		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(jsonData), &streamResp); err != nil {
			s.logger.Printf("解析流式响应失败: %v", err)
			continue
		}
		if len(streamResp.Choices) == 0 {
			continue
		}
		blockContent := streamResp.Choices[0].Delta.Content
		if blockContent == "" {
			continue
		}
		return blockContent, nil
	}

	if err := s.scanner.Err(); err != nil {
		return "", fmt.Errorf("读取流式响应失败: %v", err)
	}
	return "", io.EOF
}

func (s *sseChatStream) Close() error {
	return s.body.Close()
}
