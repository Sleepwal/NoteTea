package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func noteDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".agenttea", "notes"), nil
}

func ensureNoteDir() error {
	dir, err := noteDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

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

func SaveNote(note *Note) error {
	if err := ensureNoteDir(); err != nil {
		return err
	}

	note.UpdatedAt = time.Now()

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

func parseNoteContent(data []byte) (*Note, error) {
	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return &Note{
			Title:   "未命名笔记",
			Content: content,
			Tags:    []string{},
		}, nil
	}

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

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

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
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

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

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].UpdatedAt.After(notes[j].UpdatedAt)
	})

	return notes, nil
}

func DeleteNote(id string) error {
	dir, err := noteDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, id+".md")
	return os.Remove(path)
}
