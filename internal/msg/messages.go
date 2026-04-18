package msg

import (
	"context"
	"io"
)

type StreamTokenMsg struct {
	Content string
}

type StreamDoneMsg struct {
	TotalDuration   int64
	PromptEvalCount int
	EvalCount       int
}

type StreamStartMsg struct {
	Reader    io.ReadCloser
	CancelCtx context.CancelFunc
}

type ApiErrorMsg struct {
	Err error
}

type SendRequestMsg struct{}

type CancelRequestMsg struct{}

type ClearHistoryMsg struct{}

type NewChatMsg struct{}

type ToggleHelpMsg struct{}

type FocusChangeMsg struct {
	Focused string
}
