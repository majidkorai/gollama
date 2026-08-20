package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChatRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	session := &ChatSession{
		Model: "fake-1b",
		Messages: []ChatMessage{
			{Role: "user", Content: "hello there"},
			{Role: "assistant", Content: "hi!"},
		},
	}
	if err := SaveChat(session); err != nil {
		t.Fatalf("SaveChat: %v", err)
	}
	if session.ID == "" {
		t.Fatal("SaveChat did not assign an ID")
	}
	if session.Title != "hello there" {
		t.Fatalf("Title = %q, want auto-title from first user message", session.Title)
	}

	loaded, err := LoadChat(session.ID)
	if err != nil {
		t.Fatalf("LoadChat: %v", err)
	}
	if loaded.Model != "fake-1b" || len(loaded.Messages) != 2 {
		t.Fatalf("round-trip mismatch: %+v", loaded)
	}
	if loaded.Messages[0].Content != "hello there" || loaded.Messages[1].Content != "hi!" {
		t.Fatalf("messages mismatch: %+v", loaded.Messages)
	}
}

func TestChatTitleTruncation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	long := strings.Repeat("a", 100)
	session := &ChatSession{Messages: []ChatMessage{{Role: "user", Content: long}}}
	if err := SaveChat(session); err != nil {
		t.Fatal(err)
	}
	if len(session.Title) != 63 || !strings.HasSuffix(session.Title, "…") {
		t.Fatalf("Title = %q (len %d), want 60 bytes + ellipsis (63 bytes)", session.Title, len(session.Title))
	}
}

func TestListChatsSkipsCorruptFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := SaveChat(&ChatSession{ID: "1", Title: "one", Messages: []ChatMessage{{Role: "user", Content: "first"}}}); err != nil {
		t.Fatal(err)
	}
	if err := SaveChat(&ChatSession{ID: "2", Title: "two", Messages: []ChatMessage{{Role: "user", Content: "second"}}}); err != nil {
		t.Fatal(err)
	}
	// A corrupt file must be skipped, not fatal.
	if err := os.WriteFile(filepath.Join(ChatsDir(), "corrupt.json"), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	chats, err := ListChats()
	if err != nil {
		t.Fatalf("ListChats: %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("ListChats = %d entries, want 2 (corrupt file should be skipped)", len(chats))
	}
	ids := map[string]bool{}
	for _, c := range chats {
		ids[c.ID] = true
	}
	if !ids["1"] || !ids["2"] {
		t.Fatalf("missing chats: %v", ids)
	}
}

func TestListChatsPreviewTruncation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	long := strings.Repeat("b", 100)
	if err := SaveChat(&ChatSession{ID: "p", Messages: []ChatMessage{{Role: "user", Content: long}}}); err != nil {
		t.Fatal(err)
	}
	chats, err := ListChats()
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 {
		t.Fatalf("ListChats = %d entries, want 1", len(chats))
	}
	if len(chats[0].Preview) != 83 || !strings.HasSuffix(chats[0].Preview, "…") {
		t.Fatalf("Preview = %q (len %d), want 80 bytes + ellipsis (83 bytes)", chats[0].Preview, len(chats[0].Preview))
	}
}

func TestDeleteChat(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := SaveChat(&ChatSession{ID: "d", Messages: []ChatMessage{{Role: "user", Content: "x"}}}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteChat("d"); err != nil {
		t.Fatalf("DeleteChat: %v", err)
	}
	if _, err := LoadChat("d"); err == nil {
		t.Fatal("LoadChat after delete should fail")
	}
	if err := DeleteChat("nope"); err == nil {
		t.Fatal("expected error deleting a missing chat")
	}
}

func TestChatPathTraversal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // isolate on Windows too
	base := ChatsDir()
	if p := chatPath("normal-id"); p != filepath.Join(base, "normal-id.json") {
		t.Errorf("chatPath(normal-id) = %q, want %q", p, filepath.Join(base, "normal-id.json"))
	}
	// Defense in depth: any id that could escape the chats dir maps to an
	// inert path, never outside ChatsDir().
	for _, id := range []string{"../etc/passwd", "..\\evil", "a/b", "a..b", "", ".x", "a b"} {
		if p := chatPath(id); p != filepath.Join(base, ".invalid") {
			t.Errorf("chatPath(%q) = %q, want %q", id, p, filepath.Join(base, ".invalid"))
		}
	}
}
