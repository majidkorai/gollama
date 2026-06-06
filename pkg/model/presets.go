package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Presets struct {
	mu   sync.RWMutex
	path string
	data map[string][]string
}

var globalPresets *Presets

func PresetsFile() string {
	return filepath.Join(GollamaDir(), "presets.json")
}

func GetPresets() *Presets {
	if globalPresets == nil {
		globalPresets = &Presets{
			path: PresetsFile(),
			data: make(map[string][]string),
		}
		globalPresets.load()
	}
	return globalPresets
}

func (p *Presets) load() {
	p.mu.Lock()
	defer p.mu.Unlock()
	data, err := os.ReadFile(p.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &p.data)
	if p.data == nil {
		p.data = make(map[string][]string)
	}
}

func (p *Presets) save() {
	data, _ := json.MarshalIndent(p.data, "", "  ")
	os.MkdirAll(filepath.Dir(p.path), 0755)
	tmp := p.path + ".tmp"
	os.WriteFile(tmp, data, 0644)
	os.Rename(tmp, p.path)
}

func (p *Presets) List() map[string][]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make(map[string][]string, len(p.data))
	for k, v := range p.data {
		flags := make([]string, len(v))
		copy(flags, v)
		result[k] = flags
	}
	return result
}

func (p *Presets) Save(name string, flags []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := make([]string, len(flags))
	copy(cp, flags)
	p.data[name] = cp
	return p.saveLocked()
}

func (p *Presets) Delete(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.data, name)
	p.saveLocked()
}

func (p *Presets) saveLocked() error {
	data, err := json.MarshalIndent(p.data, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(p.path), 0755)
	tmp := p.path + ".tmp"
	os.WriteFile(tmp, data, 0644)
	os.Rename(tmp, p.path)
	return nil
}
