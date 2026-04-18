package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/agenttea/internal/api"
	"github.com/user/agenttea/internal/msg"
	"github.com/user/agenttea/internal/ui"
)

func (m AppModel) handleSend() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}

	m.inputHistory = append(m.inputHistory, input)
	m.historyIndex = -1
	m.currentInput = ""

	userMsg := ChatMessage{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)

	assistantMsg := ChatMessage{
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
		Streaming: true,
	}
	m.messages = append(m.messages, assistantMsg)

	m.apiMessages = m.buildAPIMessages()

	m.textarea.Reset()
	m.loading = true
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()

	return m, m.startChatRequest()
}

func (m AppModel) buildAPIMessages() []api.Message {
	apiMsgs := make([]api.Message, 0, len(m.messages))
	for _, chatMsg := range m.messages {
		if chatMsg.Role == "user" || (chatMsg.Role == "assistant" && !chatMsg.Streaming) {
			apiMsgs = append(apiMsgs, api.Message{
				Role:    chatMsg.Role,
				Content: chatMsg.Content,
			})
		}
	}
	return apiMsgs
}

func (m AppModel) startChatRequest() tea.Cmd {
	apiMsgs := m.apiMessages
	client := m.client

	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())

		reader, err := client.SendChat(ctx, apiMsgs)
		if err != nil {
			cancel()
			return msg.ApiErrorMsg{Err: err}
		}

		return msg.StreamStartMsg{
			Reader:    reader,
			CancelCtx: cancel,
		}
	}
}

func (m AppModel) readNextStreamToken() tea.Cmd {
	scanner := m.streamScan

	return func() tea.Msg {
		if scanner == nil {
			return msg.StreamDoneMsg{}
		}

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var resp api.ChatResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				continue
			}

			if resp.Done {
				return msg.StreamDoneMsg{
					TotalDuration:   resp.TotalDuration,
					PromptEvalCount: resp.PromptEvalCount,
					EvalCount:       resp.EvalCount,
				}
			}

			if resp.Message.Content != "" {
				return msg.StreamTokenMsg{Content: resp.Message.Content}
			}
		}

		if err := scanner.Err(); err != nil {
			return msg.ApiErrorMsg{Err: fmt.Errorf("读取流式响应失败: %w", err)}
		}

		return msg.StreamDoneMsg{}
	}
}

func (m *AppModel) cleanupStream() {
	if m.streamReader != nil {
		m.streamReader.Close()
		m.streamReader = nil
	}
	m.streamScan = nil
}

func (m *AppModel) handleStreamDone(message msg.StreamDoneMsg) {
	if len(m.messages) > 0 {
		last := &m.messages[len(m.messages)-1]
		last.Streaming = false
		var stats []string
		if message.PromptEvalCount > 0 {
			stats = append(stats, fmt.Sprintf("prompt: %d tokens", message.PromptEvalCount))
		}
		if message.EvalCount > 0 {
			stats = append(stats, fmt.Sprintf("completion: %d tokens", message.EvalCount))
		}
		if message.TotalDuration > 0 {
			duration := time.Duration(message.TotalDuration)
			stats = append(stats, fmt.Sprintf("耗时: %s", duration.Round(time.Millisecond)))
		}
		if len(stats) > 0 {
			last.Content += "\n\n" + ui.StatsStyle.Render(fmt.Sprintf("📊 %s", strings.Join(stats, " | ")))
		}
	}
	m.loading = false
	m.hasError = false
	m.cancelFunc = nil
	m.cleanupStream()
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
}

func (m *AppModel) handleApiError(message msg.ApiErrorMsg) {
	m.loading = false
	m.hasError = true
	m.cancelFunc = nil
	m.cleanupStream()
	if len(m.messages) > 0 {
		last := &m.messages[len(m.messages)-1]
		if last.Streaming {
			last.Streaming = false
			last.Content += fmt.Sprintf("\n\n[错误] %s\n按 r 键重试", message.Err.Error())
		}
	}
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
}

func (m AppModel) handleRetry() (tea.Model, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}

	last := &m.messages[len(m.messages)-1]
	if last.Role == "assistant" {
		m.messages = m.messages[:len(m.messages)-1]
	}

	if len(m.messages) == 0 {
		return m, nil
	}

	lastUserMsg := m.messages[len(m.messages)-1]
	if lastUserMsg.Role != "user" {
		return m, nil
	}

	assistantMsg := ChatMessage{
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
		Streaming: true,
	}
	m.messages = append(m.messages, assistantMsg)

	m.apiMessages = m.buildAPIMessages()
	m.loading = true
	m.hasError = false
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()

	return m, m.startChatRequest()
}
