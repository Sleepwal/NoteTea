package ui

import (
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/user/agenttea/internal/logger"
)

var codeBlockRe = regexp.MustCompile("(?s)```\\w*\\n(.*?)```")

func ExtractCodeBlocks(text string) []string {
	matches := codeBlockRe.FindAllStringSubmatch(text, -1)
	var blocks []string
	for _, match := range matches {
		if len(match) >= 2 {
			blocks = append(blocks, strings.TrimSpace(match[1]))
		}
	}
	return blocks
}

func CopyToClipboard(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		logger.Error("复制到剪贴板失败: %v", err)
		return err
	}
	logger.Info("已复制到剪贴板, 长度: %d", len(text))
	return nil
}

func CopyLastCodeBlock(text string) (bool, string) {
	blocks := ExtractCodeBlocks(text)
	if len(blocks) == 0 {
		return false, ""
	}
	last := blocks[len(blocks)-1]
	if err := CopyToClipboard(last); err != nil {
		return false, ""
	}
	preview := last
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}
	return true, preview
}
