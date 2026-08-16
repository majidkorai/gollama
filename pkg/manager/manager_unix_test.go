//go:build !windows

package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/majidkorai/gollama/pkg/model"
)

// setupFakeModel installs a throwaway HOME with a dummy model blob, an index
// entry for "fake-model", and a llama-server binary that just sleeps.
func setupFakeModel(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := filepath.Join(home, ".gollama", "bin")
	modelsDir := filepath.Join(home, ".gollama", "models")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(binDir, "llama-server")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\nsleep 30\n"), 0755); err != nil {
		t.Fatal(err)
	}
	blob := filepath.Join(modelsDir, "fake.gguf")
	if err := os.WriteFile(blob, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	idx := map[string]model.ModelInfo{
		"fake-model": {Name: "fake-model", ShortName: "fake-model", BlobPath: blob, Size: 4},
	}
	model.SaveIndex(idx)
}

// A failed start must not poison the port: a stale (stopped) entry should be
// replaced so the next launch on the same port can proceed.
func TestStartReusesStaleSlot(t *testing.T) {
	setupFakeModel(t)
	m := &Manager{
		instances: map[int]*Instance{
			8123: {Port: 8123, Status: "stopped", Model: "fake-model"},
		},
		nextPort: 9000,
	}
	inst, err := m.Start("fake-model", 8123, nil, false, nil)
	if err != nil {
		t.Fatalf("Start on stale slot: %v", err)
	}
	t.Cleanup(func() { m.Stop(inst.Port) })
	if inst.Port != 8123 {
		t.Fatalf("port = %d, want 8123 (stale slot reused)", inst.Port)
	}
	if inst.Status != "running" {
		t.Fatalf("status = %q, want running", inst.Status)
	}
}
