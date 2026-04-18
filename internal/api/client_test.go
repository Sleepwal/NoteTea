package api

import (
	"encoding/json"
	"testing"
)

func TestAvailableModels(t *testing.T) {
	if len(AvailableModels) == 0 {
		t.Error("AvailableModels is empty")
	}

	found := false
	for _, m := range AvailableModels {
		if m == DefaultModel {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DefaultModel %q not found in AvailableModels", DefaultModel)
	}
}

func TestApiError(t *testing.T) {
	err := &ApiError{StatusCode: 401, Message: "unauthorized"}
	expected := "API error (status 401): unauthorized"
	if err.Error() != expected {
		t.Errorf("ApiError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestChatRequestSerialization(t *testing.T) {
	req := ChatRequest{
		Model: "qwen3.5:cloud",
		Messages: []Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi there"},
		},
		Stream: true,
		Options: ChatOptions{
			Temperature: 0.7,
			NumPredict:  4096,
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed ChatRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed.Model != req.Model {
		t.Errorf("parsed.Model = %q, want %q", parsed.Model, req.Model)
	}
	if len(parsed.Messages) != 2 {
		t.Fatalf("len(parsed.Messages) = %d, want 2", len(parsed.Messages))
	}
	if parsed.Messages[0].Role != "user" {
		t.Errorf("parsed.Messages[0].Role = %q, want %q", parsed.Messages[0].Role, "user")
	}
	if !parsed.Stream {
		t.Error("parsed.Stream = false, want true")
	}
}

func TestChatResponseParsing(t *testing.T) {
	jsonData := `{"model":"qwen3.5:cloud","message":{"role":"assistant","content":"hello"},"done":false,"total_duration":1000000000,"prompt_eval_count":10,"eval_count":20}`

	var resp ChatResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if resp.Model != "qwen3.5:cloud" {
		t.Errorf("resp.Model = %q, want %q", resp.Model, "qwen3.5:cloud")
	}
	if resp.Message.Content != "hello" {
		t.Errorf("resp.Message.Content = %q, want %q", resp.Message.Content, "hello")
	}
	if resp.Done {
		t.Error("resp.Done = true, want false")
	}
	if resp.PromptEvalCount != 10 {
		t.Errorf("resp.PromptEvalCount = %d, want 10", resp.PromptEvalCount)
	}
	if resp.EvalCount != 20 {
		t.Errorf("resp.EvalCount = %d, want 20", resp.EvalCount)
	}
}
