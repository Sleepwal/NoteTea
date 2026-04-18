package ui

import (
	"github.com/charmbracelet/glamour"
)

var renderer *glamour.TermRenderer

func init() {
	var err error
	renderer, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		panic(err)
	}
}

func RenderMarkdown(text string) string {
	rendered, err := renderer.Render(text)
	if err != nil {
		return text
	}
	return rendered
}
