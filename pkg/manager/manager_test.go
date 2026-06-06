package manager

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestList(t *testing.T) {
	m := NewManager()
	list := m.List()
	if list == nil {
		t.Fatal("List() returned nil")
	}
}

func TestStopNonExistent(t *testing.T) {
	m := NewManager()
	err := m.Stop(99999)
	if err == nil {
		t.Error("expected error when stopping non-existent instance")
	}
}

func TestUpdateTokensNonExistent(t *testing.T) {
	m := NewManager()
	m.UpdateTokens(99999, 42.5)
}

func TestUpdateTokens(t *testing.T) {
	m := NewManager()
	m.UpdateTokens(0, 10.5)
}

func TestNewManagerMultiple(t *testing.T) {
	m1 := NewManager()
	m2 := NewManager()
	if m1 == nil || m2 == nil {
		t.Fatal("NewManager() returned nil")
	}
}
