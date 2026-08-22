package model

import (
	"reflect"
	"testing"
)

// These tests exercise the typed flag model (P5-T3) directly: ParseFlags,
// Args, Merge, and the standalone-flag derivation helpers. The behavior
// contract is pinned by flags_test.go (the P0-T4 snapshots); these tests
// document the model's semantics.

func TestParseFlagsForms(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string // expected Args() output
	}{
		{
			name: "key value form",
			in:   []string{"--ctx-size", "4096", "--temp", "0.7"},
			want: []string{"--ctx-size", "4096", "--temp", "0.7"},
		},
		{
			name: "key equals value form",
			in:   []string{"--ctx-size=4096", "--temp=0.7"},
			want: []string{"--ctx-size", "4096", "--temp", "0.7"},
		},
		{
			name: "mixed forms",
			in:   []string{"--ctx-size=4096", "--temp", "0.7", "--flash-attn=on"},
			want: []string{"--ctx-size", "4096", "--temp", "0.7", "--flash-attn", "on"},
		},
		{
			name: "standalone flags take no value",
			in:   []string{"--verbose", "--mlock", "--ctx-size", "4096"},
			want: []string{"--verbose", "--mlock", "--ctx-size", "4096"},
		},
		{
			name: "derived no-form is standalone",
			in:   []string{"--no-verbose", "--ctx-size", "4096"},
			want: []string{"--no-verbose", "--ctx-size", "4096"},
		},
		{
			name: "valued flag not made standalone by its no-form",
			// --flash-attn is valued even though --no-flash-attn is standalone.
			in:   []string{"--flash-attn", "on"},
			want: []string{"--flash-attn", "on"},
		},
		{
			name: "leading orphan dropped",
			in:   []string{"4096", "--ctx-size", "4096"},
			want: []string{"--ctx-size", "4096"},
		},
		{
			name: "orphan after standalone dropped",
			in:   []string{"--verbose", "true", "--temp", "0.7"},
			want: []string{"--verbose", "--temp", "0.7"},
		},
		{
			name: "consecutive values dropped",
			in:   []string{"--ctx-size", "4096", "8192"},
			want: []string{"--ctx-size", "4096"},
		},
		{
			name: "flag whose next token is a flag takes no value",
			in:   []string{"--foo", "--bar"},
			want: []string{"--foo", "--bar"},
		},
		{
			name: "empty input",
			in:   []string{},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFlags(tt.in).Args()
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseFlags(%v).Args() = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseFlagsDuplicateKeyLastWins(t *testing.T) {
	// A key repeated keeps its first-seen position but the last value.
	got := ParseFlags([]string{"--temp", "0.5", "--ctx-size", "1024", "--temp", "0.9"}).Args()
	want := []string{"--temp", "0.9", "--ctx-size", "1024"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeValuedOverride(t *testing.T) {
	base := ParseFlags([]string{"--temp", "0.7", "--ctx-size", "2048"})
	over := ParseFlags([]string{"--temp", "0.9"})
	got := base.Merge(over).Args()
	want := []string{"--ctx-size", "2048", "--temp", "0.9"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeAppendsNewFlags(t *testing.T) {
	base := ParseFlags([]string{"--temp", "0.7", "--ctx-size", "2048"})
	over := ParseFlags([]string{"--flash-attn", "off"})
	got := base.Merge(over).Args()
	want := []string{"--temp", "0.7", "--ctx-size", "2048", "--flash-attn", "off"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeStandaloneSetUnset(t *testing.T) {
	// --no-verbose in override unsets a base --verbose.
	base := ParseFlags([]string{"--verbose", "--ctx-size", "2048"})
	over := ParseFlags([]string{"--no-verbose"})
	got := base.Merge(over).Args()
	want := []string{"--ctx-size", "2048", "--no-verbose"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	// Reverse: --verbose in override unsets a base --no-verbose.
	base2 := ParseFlags([]string{"--no-verbose", "--ctx-size", "2048"})
	over2 := ParseFlags([]string{"--verbose"})
	got2 := base2.Merge(over2).Args()
	want2 := []string{"--ctx-size", "2048", "--verbose"}
	if !reflect.DeepEqual(got2, want2) {
		t.Fatalf("got %v, want %v", got2, want2)
	}
}

func TestMergeStandaloneSameKey(t *testing.T) {
	// Same standalone key in both: deduped, appears once (from override).
	base := ParseFlags([]string{"--verbose"})
	over := ParseFlags([]string{"--verbose"})
	got := base.Merge(over).Args()
	want := []string{"--verbose"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestStandaloneFlagDerivation(t *testing.T) {
	// Direct map membership.
	if !IsStandaloneFlag("--verbose") {
		t.Error("--verbose should be standalone")
	}
	if !IsStandaloneFlag("--no-flash-attn") {
		t.Error("--no-flash-attn should be standalone")
	}
	// Derived: --no-X is standalone when --X is.
	if !IsStandaloneFlag("--no-verbose") {
		t.Error("--no-verbose should be standalone (derived from --verbose)")
	}
	// NOT derived in reverse: --flash-attn stays valued.
	if IsStandaloneFlag("--flash-attn") {
		t.Error("--flash-attn should NOT be standalone")
	}
	// Not a flag at all.
	if IsStandaloneFlag("verbose") {
		t.Error("bare word should NOT be standalone")
	}
}

func TestStandaloneCounterpart(t *testing.T) {
	tests := []struct{ in, want string }{
		{"--no-verbose", "--verbose"},
		{"--verbose", "--no-verbose"},
		{"--no-flash-attn", "--flash-attn"},
		{"not-a-flag", ""},
	}
	for _, tt := range tests {
		if got := standaloneCounterpart(tt.in); got != tt.want {
			t.Errorf("standaloneCounterpart(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
