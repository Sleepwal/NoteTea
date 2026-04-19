package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type BackendType string

const (
	BackendOllama BackendType = "ollama"
	BackendOpenAI BackendType = "openai"
)

const (
	DefaultBaseURL = "https://ollama.com"
	DefaultModel   = "qwen3.5:cloud"
	ChatEndpoint   = "/api/chat"
	OpenAIEndpoint = "/v1/chat/completions"
)

var AvailableModels = []string{
	"qwen3.5:cloud",
	"gpt-oss:120b",
	"gpt-oss:20b",
	"deepseek:v3.1:571b",
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

type ChatRequest struct {
	Model    string      `json:"model"`
	Messages []Message   `json:"messages"`
	Stream   bool        `json:"stream"`
	Options  ChatOptions `json:"options,omitempty"`
}

type ChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`

	TotalDuration   int64 `json:"total_duration,omitempty"`
	PromptEvalCount int   `json:"prompt_eval_count,omitempty"`
	EvalCount       int   `json:"eval_count,omitempty"`
}

type OpenAIChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason *string `json:"finish_reason"`
		Delta        *struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type ApiError struct {
	StatusCode int
	Message    string
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

type Client struct {
	BaseURL    string
	APIKey     string
	Model      string
	Backend    BackendType
	HTTPClient *http.Client
}

func DetectBackend(baseURL string) BackendType {
	if strings.Contains(baseURL, "ollama.com") {
		return BackendOllama
	}
	return BackendOpenAI
}

func NewClient() (*Client, error) {
	apiKey := os.Getenv("OLLAMA_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 OLLAMA_API_KEY 未设置，请先设置 API 密钥")
	}

	return &Client{
		BaseURL: DefaultBaseURL,
		APIKey:  apiKey,
		Model:   DefaultModel,
		Backend: BackendOllama,
		HTTPClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}, nil
}

func NewClientWithConfig(baseURL, apiKey, model string) *Client {
	backend := DetectBackend(baseURL)
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Backend: backend,
		HTTPClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

func (c *Client) SendChat(ctx context.Context, messages []Message) (io.ReadCloser, error) {
	if !strings.HasPrefix(c.BaseURL, "https://") {
		fmt.Fprintf(os.Stderr, "警告: BaseURL (%s) 未使用 HTTPS，API 密钥将以明文传输\n", c.BaseURL)
	}

	if c.Backend == BackendOpenAI {
		return c.sendOpenAIChat(ctx, messages)
	}
	return c.sendOllamaChat(ctx, messages)
}

func (c *Client) sendOllamaChat(ctx context.Context, messages []Message) (io.ReadCloser, error) {
	reqBody := ChatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   true,
		Options: ChatOptions{
			Temperature: 0.7,
			NumPredict:  4096,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+ChatEndpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	return resp.Body, nil
}

func (c *Client) sendOpenAIChat(ctx context.Context, messages []Message) (io.ReadCloser, error) {
	reqBody := OpenAIChatRequest{
		Model:       c.Model,
		Messages:    messages,
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+OpenAIEndpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	return resp.Body, nil
}

func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return &ApiError{StatusCode: statusCode, Message: "API 密钥无效或已过期"}
	case http.StatusNotFound:
		return &ApiError{StatusCode: statusCode, Message: fmt.Sprintf("模型 '%s' 不存在或端点不可用", c.Model)}
	case http.StatusTooManyRequests:
		return &ApiError{StatusCode: statusCode, Message: "请求频率超限，请稍后重试"}
	default:
		errMsg := string(body)
		if errMsg == "" {
			errMsg = "未知错误"
		}
		return &ApiError{StatusCode: statusCode, Message: errMsg}
	}
}

func ParseStream(reader io.ReadCloser, onToken func(ChatResponse)) error {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp ChatResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		onToken(resp)

		if resp.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取流式响应失败: %w", err)
	}

	return nil
}

func ParseOpenAIStream(reader io.ReadCloser, onToken func(content string, done bool, usage OpenAIChatResponse)) error {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		lineStr := string(line)
		if !strings.HasPrefix(lineStr, "data: ") {
			continue
		}

		data := strings.TrimPrefix(lineStr, "data: ")
		if data == "[DONE]" {
			onToken("", true, OpenAIChatResponse{})
			break
		}

		var resp OpenAIChatResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if len(resp.Choices) > 0 {
			choice := resp.Choices[0]
			content := ""
			if choice.Delta != nil {
				content = choice.Delta.Content
			}
			done := choice.FinishReason != nil && *choice.FinishReason == "stop"
			onToken(content, done, resp)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取流式响应失败: %w", err)
	}

	return nil
}
