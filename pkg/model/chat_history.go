package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type ChatSummary struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	Preview   string    `json:"preview"`
	MsgCount  int       `json:"msg_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ChatsDir() string {
	return filepath.Join(GollamaDir(), "chats")
}

func chatPath(id string) string {
	return filepath.Join(ChatsDir(), id+".json")
}

func SaveChat(session *ChatSession) error {
	EnsureDir(ChatsDir())
	session.UpdatedAt = time.Now()
	if session.ID == "" {
		session.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		session.CreatedAt = session.UpdatedAt
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
		summaries = append(summaries, ChatSummary{
			ID:        session.ID,
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
