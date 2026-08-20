package model

import (
	"path/filepath"
	"testing"
)

func TestPresetsSaveAndList(t *testing.T) {
	tmp := t.TempDir()
	p := &Presets{
		path: filepath.Join(tmp, "presets.json"),
		data: make(map[string][]string),
	}

	err := p.Save("test", []string{"--ctx-size", "4096", "--temp", "0.7"})
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	list := p.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 preset, got %d", len(list))
	}
	if list["test"] == nil {
		t.Fatal("expected 'test' preset")
	}
	if len(list["test"]) != 4 {
		t.Fatalf("expected 4 flag tokens, got %d", len(list["test"]))
	}
}

func TestPresetsDelete(t *testing.T) {
	tmp := t.TempDir()
	p := &Presets{
		path: filepath.Join(tmp, "presets.json"),
		data: make(map[string][]string),
	}

	p.Save("a", []string{"--ctx-size", "2048"})
	p.Save("b", []string{"--temp", "0.5"})
	p.Delete("a")

	list := p.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 preset after delete, got %d", len(list))
	}
	if list["b"] == nil {
		t.Fatal("expected 'b' preset to remain")
	}
}

func TestPresetsPersistence(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "presets.json")

	p1 := &Presets{path: path, data: make(map[string][]string)}
	p1.Save("x", []string{"--flag", "val"})

	p2 := &Presets{path: path, data: make(map[string][]string)}
	p2.load()

	list := p2.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 preset after reload, got %d", len(list))
	}
}

func TestPresetsFile(t *testing.T) {
	setTestHome(t)

	path := PresetsFile()
	if !filepath.IsAbs(path) {
		t.Errorf("PresetsFile() = %q, should be absolute", path)
	}
}
