package model

import (
	"os"
	"reflect"
	"testing"
)

// These tests snapshot the CURRENT flag-handling behavior. The flag system is
// a pile of positional heuristics (see ROBUSTNESS_PLAN.md P5-T3) and the
// snapshot is the acceptance gate for the typed flag-model rewrite.

func TestSanitizeFlags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "clean flags pass through",
			in:   []string{"--ctx-size", "4096", "--flash-attn", "on"},
			want: []string{"--ctx-size", "4096", "--flash-attn", "on"},
		},
		{
			name: "leading orphan value dropped",
			in:   []string{"4096", "--ctx-size", "4096"},
			want: []string{"--ctx-size", "4096"},
		},
		{
			name: "orphan value after a standalone flag dropped",
			in:   []string{"--verbose", "true", "--temp", "0.7"},
			want: []string{"--verbose", "--temp", "0.7"},
		},
		{
			name: "consecutive values dropped",
			in:   []string{"--ctx-size", "4096", "8192"},
			want: []string{"--ctx-size", "4096"},
		},
		{
			name: "empty stays empty",
			in:   []string{},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFlags(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("sanitizeFlags(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestProfileFlags(t *testing.T) {
	cfg := &Config{
		DefaultFlags:  []string{"--host", "0.0.0.0", "--temp", "0.7", "--ctx-size", "2048"},
		ProxyDefaults: []string{"--temp", "0.7", "--ctx-size", "2048"},
		Profiles: map[string]Profile{
			"override":   {Flags: []string{"--temp", "0.9"}},
			"extend":     {Flags: []string{"--temp", "0.9", "--flash-attn", "off"}},
			"standalone": {Flags: []string{"--no-verbose"}},
		},
	}

	t.Run("unknown profile falls back to proxy defaults", func(t *testing.T) {
		want := []string{"--temp", "0.7", "--ctx-size", "2048"}
		if got := cfg.ProfileFlags("missing"); !reflect.DeepEqual(got, want) {
			t.Fatalf("ProfileFlags(missing) = %v, want %v", got, want)
		}
	})

	t.Run("profile value override drops the base flag and value", func(t *testing.T) {
		want := []string{"--ctx-size", "2048", "--temp", "0.9"}
		if got := cfg.ProfileFlags("override"); !reflect.DeepEqual(got, want) {
			t.Fatalf("ProfileFlags(override) = %v, want %v", got, want)
		}
	})

	t.Run("profile adds new flags", func(t *testing.T) {
		want := []string{"--ctx-size", "2048", "--temp", "0.9", "--flash-attn", "off"}
		if got := cfg.ProfileFlags("extend"); !reflect.DeepEqual(got, want) {
			t.Fatalf("ProfileFlags(extend) = %v, want %v", got, want)
		}
	})

	t.Run("standalone flags set/unset their counterpart (P5-T3)", func(t *testing.T) {
		// P5-T3 fix: a standalone flag in the profile unsets its --no-/bare
		// counterpart from the base. Effective behavior is unchanged — the
		// profile flag always came last, and llama-server takes the last —
		// but the redundant base flag is no longer emitted.
		cfg2 := &Config{
			ProxyDefaults: []string{"--verbose"},
			Profiles:      map[string]Profile{"standalone": {Flags: []string{"--no-verbose"}}},
		}
		want := []string{"--no-verbose"}
		if got := cfg2.ProfileFlags("standalone"); !reflect.DeepEqual(got, want) {
			t.Fatalf("ProfileFlags(standalone) = %v, want %v", got, want)
		}
		// And the reverse direction: --verbose unsets a base --no-verbose.
		cfg3 := &Config{
			ProxyDefaults: []string{"--no-verbose"},
			Profiles:      map[string]Profile{"standalone": {Flags: []string{"--verbose"}}},
		}
		want3 := []string{"--verbose"}
		if got := cfg3.ProfileFlags("standalone"); !reflect.DeepEqual(got, want3) {
			t.Fatalf("ProfileFlags(standalone) = %v, want %v", got, want3)
		}
	})
}

func TestProxyFlagsFallback(t *testing.T) {
	cfg := &Config{DefaultFlags: []string{"--temp", "0.7"}}
	if got := cfg.ProxyFlags(); !reflect.DeepEqual(got, []string{"--temp", "0.7"}) {
		t.Fatalf("ProxyFlags() = %v, want default_flags fallback", got)
	}
	cfg.ProxyDefaults = []string{"--ctx-size", "4096"}
	if got := cfg.ProxyFlags(); !reflect.DeepEqual(got, []string{"--ctx-size", "4096"}) {
		t.Fatalf("ProxyFlags() = %v, want proxy_defaults", got)
	}
}

func TestLoadConfigSanitizesFlags(t *testing.T) {
	setTestHome(t)
	// Simulate a hand-edited config containing orphaned values.
	raw := `{"default_flags":["4096","--ctx-size","4096","--verbose","true"],"profiles":{},"idle_ttl":0}`
	EnsureDir(GollamaDir())
	if err := os.WriteFile(ConfigFile(), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := LoadConfig()
	want := []string{"--ctx-size", "4096", "--verbose"}
	if !reflect.DeepEqual(cfg.DefaultFlags, want) {
		t.Fatalf("DefaultFlags = %v, want %v", cfg.DefaultFlags, want)
	}
}
