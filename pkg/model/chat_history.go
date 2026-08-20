package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatSession struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type ChatSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	Preview   string    `json:"preview"`
	MsgCount  int       `json:"msg_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ChatsDir() string {
	return filepath.Join(GollamaDir(), "chats")
}

// ValidChatID reports whether id is a safe chat session identifier.
// Chat ids are filesystem names under ~/.gollama/chats/, so anything
// beyond [A-Za-z0-9_-] (up to 64 chars) is rejected.
var chatIDRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func ValidChatID(id string) bool {
	return chatIDRe.MatchString(id)
}

func chatPath(id string) string {
	// Defense in depth: reject any id that could escape the chats dir,
	// even if a caller bypasses ValidChatID.
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") || !ValidChatID(id) {
		return filepath.Join(ChatsDir(), ".invalid")
	}
	return filepath.Join(ChatsDir(), id+".json")
}

func SaveChat(session *ChatSession) error {
	EnsureDir(ChatsDir())
	session.UpdatedAt = time.Now()
	if session.ID == "" {
		session.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		session.CreatedAt = session.UpdatedAt
	}
	// Auto-generate title from first user message if empty
	if session.Title == "" {
		for _, m := range session.Messages {
			if m.Role == "user" && m.Content != "" {
				session.Title = m.Content
				if len(session.Title) > 60 {
					session.Title = session.Title[:60] + "…"
				}
				break
			}
		}
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	path := chatPath(session.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func LoadChat(id string) (*ChatSession, error) {
	data, err := os.ReadFile(chatPath(id))
	if err != nil {
		return nil, err
	}
	var session ChatSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func DeleteChat(id string) error {
	return os.Remove(chatPath(id))
}

func ListChats() ([]ChatSummary, error) {
	EnsureDir(ChatsDir())
	entries, err := os.ReadDir(ChatsDir())
	if err != nil {
		return nil, err
	}
	var summaries []ChatSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		data, err := os.ReadFile(chatPath(id))
		if err != nil {
			continue
		}
		var session ChatSession
		if json.Unmarshal(data, &session) != nil {
			continue
		}
		preview := ""
		for _, m := range session.Messages {
			if m.Role == "user" && m.Content != "" {
				preview = m.Content
				if len(preview) > 80 {
					preview = preview[:80] + "…"
				}
				break
			}
		}
		if preview == "" {
			preview = "(empty)"
		}
		title := session.Title
		if title == "" {
			title = preview
		}
		summaries = append(summaries, ChatSummary{
			ID:        session.ID,
			Title:     title,
			Model:     session.Model,
			Preview:   preview,
			MsgCount:  len(session.Messages),
			CreatedAt: session.CreatedAt,
			UpdatedAt: session.UpdatedAt,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})
	return summaries, nil
}

var chatMu sync.Mutex

func SaveChatConcurrent(session *ChatSession) error {
	chatMu.Lock()
	defer chatMu.Unlock()
	return SaveChat(session)
}
