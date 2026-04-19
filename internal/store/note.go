// Package store 提供数据持久化功能，包括对话和笔记的存储与读取。
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Note 表示一条笔记，使用 Markdown 文件存储在 ~/.agenttea/notes/ 目录下。
// 文件格式为 YAML frontmatter + Markdown 正文，可直接用外部编辑器打开。
//
// 文件示例:
//
//	---
//	id: "1710000000000"
//	title: "Go 并发编程笔记"
//	tags:
//	  - "go"
//	  - "并发"
//	created_at: "2025-01-01T10:00:00Z"
//	updated_at: "2025-01-01T10:00:00Z"
//	---
//
//	# Go 并发编程笔记
//	正文内容...
type Note struct {
	ID        string    `json:"id"`         // 笔记唯一标识，使用创建时间的毫秒时间戳
	Title     string    `json:"title"`      // 笔记标题
	Content   string    `json:"content"`    // Markdown 正文内容
	Tags      []string  `json:"tags"`       // 标签列表，用于分类和检索
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 最后更新时间，每次保存时自动刷新
}

// noteDir 返回笔记存储目录路径 (~/.agenttea/notes/)。
func noteDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".agenttea", "notes"), nil
}

// ensureNoteDir 确保笔记存储目录存在，若不存在则递归创建。
// 目录权限为 0700（仅所有者可读写执行）。
func ensureNoteDir() error {
	dir, err := noteDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

// NewNote 创建一条新笔记。ID 使用当前毫秒时间戳生成。
// 若 title 为空，默认设为"未命名笔记"。
func NewNote(title string) *Note {
	now := time.Now()
	tags := []string{}
	if title == "" {
		title = "未命名笔记"
	}
	return &Note{
		ID:        fmt.Sprintf("%d", now.UnixMilli()),
		Title:     title,
		Content:   "",
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SaveNote 将笔记保存为 Markdown 文件。
// 文件格式：YAML frontmatter（元数据）+ 空行 + Markdown 正文。
// 每次保存时自动更新 UpdatedAt 时间戳。
// 文件权限为 0600（仅所有者可读写）。
func SaveNote(note *Note) error {
	if err := ensureNoteDir(); err != nil {
		return err
	}

	note.UpdatedAt = time.Now()

	// 构建 YAML frontmatter
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %q\n", note.ID))
	sb.WriteString(fmt.Sprintf("title: %q\n", note.Title))
	if len(note.Tags) > 0 {
		sb.WriteString("tags:\n")
		for _, tag := range note.Tags {
			sb.WriteString(fmt.Sprintf("  - %q\n", tag))
		}
	} else {
		sb.WriteString("tags: []\n")
	}
	sb.WriteString(fmt.Sprintf("created_at: %q\n", note.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated_at: %q\n", note.UpdatedAt.Format(time.RFC3339)))
	sb.WriteString("---\n\n")
	sb.WriteString(note.Content)

	dir, err := noteDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, note.ID+".md")

	if err := os.WriteFile(path, []byte(sb.String()), 0600); err != nil {
		return fmt.Errorf("写入笔记文件失败: %w", err)
	}

	return nil
}

// LoadNote 根据 ID 读取笔记。返回 nil 表示笔记不存在。
func LoadNote(id string) (*Note, error) {
	dir, err := noteDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取笔记文件失败: %w", err)
	}

	return parseNoteContent(data)
}

// parseNoteContent 解析 Markdown 文件内容为 Note 结构体。
// 支持两种格式：
//  1. 带 YAML frontmatter 的标准格式（--- 包裹的元数据头 + 正文）
//  2. 纯 Markdown 格式（无 frontmatter，标题默认为"未命名笔记"）
func parseNoteContent(data []byte) (*Note, error) {
	content := string(data)

	// 无 frontmatter 的纯 Markdown 文件
	if !strings.HasPrefix(content, "---\n") {
		return &Note{
			Title:   "未命名笔记",
			Content: content,
			Tags:    []string{},
		}, nil
	}

	// 查找 frontmatter 结束标记
	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx == -1 {
		return &Note{
			Title:   "未命名笔记",
			Content: content,
			Tags:    []string{},
		}, nil
	}

	frontmatter := content[4 : 4+endIdx]
	body := content[4+endIdx+5:]

	note := &Note{
		Tags: []string{},
	}

	// 逐行解析 frontmatter 中的键值对
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "id:") {
			note.ID = unquote(strings.TrimSpace(strings.TrimPrefix(line, "id:")))
		} else if strings.HasPrefix(line, "title:") {
			note.Title = unquote(strings.TrimSpace(strings.TrimPrefix(line, "title:")))
		} else if strings.HasPrefix(line, "created_at:") {
			val := unquote(strings.TrimSpace(strings.TrimPrefix(line, "created_at:")))
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				note.CreatedAt = t
			}
		} else if strings.HasPrefix(line, "updated_at:") {
			val := unquote(strings.TrimSpace(strings.TrimPrefix(line, "updated_at:")))
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				note.UpdatedAt = t
			}
		} else if strings.HasPrefix(line, "- ") {
			// YAML 列表项，用于解析 tags 数组
			tag := unquote(strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			note.Tags = append(note.Tags, tag)
		}
	}

	note.Content = strings.TrimPrefix(body, "\n")

	if note.Title == "" {
		note.Title = "未命名笔记"
	}

	return note, nil
}

// unquote 去除字符串两端的双引号。
// 用于解析 YAML frontmatter 中的带引号值，如 title: "笔记标题" → 笔记标题。
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// ListNotes 列出所有笔记，按更新时间倒序排列（最近更新的排在前面）。
// 若笔记目录不存在则返回空切片而非错误。
func ListNotes() ([]Note, error) {
	dir, err := noteDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取笔记目录失败: %w", err)
	}

	var notes []Note
	for _, entry := range entries {
		// 仅处理 .md 文件，跳过目录和其他格式
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		// 文件名去掉 .md 后缀即为笔记 ID
		id := entry.Name()[:len(entry.Name())-3]
		note, err := LoadNote(id)
		if err != nil || note == nil {
			continue
		}
		if note.ID == "" {
			note.ID = id
		}
		notes = append(notes, *note)
	}

	// 按更新时间倒序排列
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].UpdatedAt.After(notes[j].UpdatedAt)
	})

	return notes, nil
}

// DeleteNote 根据 ID 删除笔记文件。
func DeleteNote(id string) error {
	dir, err := noteDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, id+".md")
	return os.Remove(path)
}
