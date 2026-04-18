package model

import (
	"testing"
	"time"

	"github.com/user/agenttea/internal/api"
)

func TestBuildAPIMessages(t *testing.T) {
	m := AppModel{
		messages: []ChatMessage{
			{Role: "user", Content: "hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "hi there", Timestamp: time.Now(), Streaming: false},
			{Role: "user", Content: "how are you?", Timestamp: time.Now()},
			{Role: "assistant", Content: "", Timestamp: time.Now(), Streaming: true},
		},
	}

	apiMsgs := m.buildAPIMessages()

	if len(apiMsgs) != 3 {
		t.Fatalf("len(apiMsgs) = %d, want 3", len(apiMsgs))
	}
	if apiMsgs[0].Role != "user" || apiMsgs[0].Content != "hello" {
		t.Errorf("apiMsgs[0] = {%q, %q}, want {user, hello}", apiMsgs[0].Role, apiMsgs[0].Content)
	}
	if apiMsgs[1].Role != "assistant" || apiMsgs[1].Content != "hi there" {
		t.Errorf("apiMsgs[1] = {%q, %q}, want {assistant, hi there}", apiMsgs[1].Role, apiMsgs[1].Content)
	}
	if apiMsgs[2].Role != "user" || apiMsgs[2].Content != "how are you?" {
		t.Errorf("apiMsgs[2] = {%q, %q}, want {user, how are you?}", apiMsgs[2].Role, apiMsgs[2].Content)
	}
}

func TestBuildAPIMessagesEmpty(t *testing.T) {
	m := AppModel{messages: []ChatMessage{}}
	apiMsgs := m.buildAPIMessages()
	if len(apiMsgs) != 0 {
		t.Errorf("len(apiMsgs) = %d, want 0", len(apiMsgs))
	}
}

func TestBuildAPIMessagesExcludesStreaming(t *testing.T) {
	m := AppModel{
		messages: []ChatMessage{
			{Role: "user", Content: "hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "partial...", Timestamp: time.Now(), Streaming: true},
		},
	}

	apiMsgs := m.buildAPIMessages()

	if len(apiMsgs) != 1 {
		t.Fatalf("len(apiMsgs) = %d, want 1 (streaming assistant should be excluded)", len(apiMsgs))
	}
	if apiMsgs[0].Role != "user" {
		t.Errorf("apiMsgs[0].Role = %q, want %q", apiMsgs[0].Role, "user")
	}
}

func TestNewAppModel(t *testing.T) {
	client := &api.Client{
		BaseURL: "https://ollama.com",
		APIKey:  "test-key",
		Model:   "qwen3.5:cloud",
	}

	m := NewAppModel(client)

	if m.focused != FocusInput {
		t.Errorf("m.focused = %d, want %d", m.focused, FocusInput)
	}
	if m.loading {
		t.Error("m.loading = true, want false")
	}
	if m.showHelp {
		t.Error("m.showHelp = true, want false")
	}
	if m.hasError {
		t.Error("m.hasError = true, want false")
	}
	if m.historyIndex != -1 {
		t.Errorf("m.historyIndex = %d, want -1", m.historyIndex)
	}
	if m.client.Model != "qwen3.5:cloud" {
		t.Errorf("m.client.Model = %q, want %q", m.client.Model, "qwen3.5:cloud")
	}
}
