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

const (
	DefaultBaseURL = "https://ollama.com"
	DefaultModel   = "qwen3.5:cloud"
	ChatEndpoint   = "/api/chat"
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
	HTTPClient *http.Client
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
		HTTPClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}, nil
}

func NewClientWithConfig(baseURL, apiKey, model string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

func (c *Client) SendChat(ctx context.Context, messages []Message) (io.ReadCloser, error) {
	if !strings.HasPrefix(c.BaseURL, "https://") {
		fmt.Fprintf(os.Stderr, "警告: BaseURL (%s) 未使用 HTTPS，API 密钥将以明文传输\n", c.BaseURL)
	}

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

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, &ApiError{StatusCode: resp.StatusCode, Message: "API 密钥无效或已过期，请检查 OLLAMA_API_KEY 环境变量"}
		case http.StatusNotFound:
			return nil, &ApiError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("模型 '%s' 不存在，可用模型: gpt-oss:120b, gpt-oss:20b, deepseek:v3.1:571b", c.Model)}
		case http.StatusTooManyRequests:
			return nil, &ApiError{StatusCode: resp.StatusCode, Message: "请求频率超限，请稍后重试"}
		default:
			errMsg := string(body)
			if errMsg == "" {
				errMsg = "未知错误"
			}
			return nil, &ApiError{StatusCode: resp.StatusCode, Message: errMsg}
		}
	}

	return resp.Body, nil
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
