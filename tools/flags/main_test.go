package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pad40 mimics llama.cpp's to_string: alias column padded to 40, help at
// column 40 (or on the next line when the alias column is too long).
func pad40(name, help string) string { return fmt.Sprintf("%-40s%s", name, help) }

func cont40(text string) string { return strings.Repeat(" ", 40) + text }

// syntheticHelp is the same shape as a real llama-server --help dump.
var syntheticHelp = strings.Join([]string{
	"----- common params -----",
	"",
	pad40("-c,    --ctx-size N", "size of the prompt context (default: 0)"),
	cont40("(env: LLAMA_ARG_CTX_SIZE)"),
	pad40("--swa-full", "use full-size SWA cache (default: false)"),
	cont40("(env: LLAMA_ARG_SWA_FULL)"),
	pad40("--brand-new-flag FNAME", "brand new valued flag"),
	pad40("--new-bool, --no-new-bool", "brand new boolean flag"),
	pad40("--draft, --draft-n, --draft-max N", "the argument has been removed. use --spec-draft-n-max"),
	pad40("-ts,  --tensor-split N0,N1,N2,...", "fraction of the model to offload to each GPU"),
	pad40("--spec-type none,draft-simple,ngram-cache", ""),
	cont40("comma-separated list of speculative types (default: none)"),
	pad40("--rope-scaling {none,linear,yarn}", "RoPE frequency scaling method"),
	"",
	"----- sampling params -----",
	"",
	pad40("-t,    --threads, --thread-count N", "number of CPU threads to use during generation (default: -1)"),
}, "\n")

func TestParseHelpSynthetic(t *testing.T) {
	entries := parseHelp(syntheticHelp)
	want := map[string][]string{
		"--ctx-size":       {"--ctx-size"},
		"--swa-full":       {"--swa-full"},
		"--brand-new-flag": {"--brand-new-flag"},
		"--new-bool":       {"--new-bool", "--no-new-bool"},
		"--draft":          {"--draft", "--draft-n", "--draft-max"},
		"--tensor-split":   {"--tensor-split"},
		"--spec-type":      {"--spec-type"},
		"--rope-scaling":   {"--rope-scaling"},
		"--threads":        {"--threads", "--thread-count"},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(want), entries)
	}
	for _, e := range entries {
		w, ok := want[e.longFlags[0]]
		if !ok {
			t.Fatalf("unexpected entry %v", e.longFlags)
		}
		if strings.Join(e.longFlags, ",") != strings.Join(w, ",") {
			t.Fatalf("entry %s: aliases %v, want %v", e.longFlags[0], e.longFlags, w)
		}
	}
	byFirst := map[string]entry{}
	for _, e := range entries {
		byFirst[e.longFlags[0]] = e
	}
	if !strings.Contains(byFirst["--ctx-size"].help, "size of the prompt context") {
		t.Errorf("inline help lost: %q", byFirst["--ctx-size"].help)
	}
	if !strings.Contains(byFirst["--ctx-size"].help, "(env:") {
		t.Errorf("continuation lines lost: %q", byFirst["--ctx-size"].help)
	}
	if !byFirst["--draft"].removedStub() {
		t.Errorf("removed stub not detected: %q", byFirst["--draft"].help)
	}
	if byFirst["--new-bool"].removedStub() {
		t.Errorf("false removed-stub detection: %q", byFirst["--new-bool"].help)
	}
	if byFirst["--spec-type"].section != "common params" {
		t.Errorf("section tracking broken: %q", byFirst["--spec-type"].section)
	}
	// Dotted flag names are real (--fim-qwen-1.5b-default in b10815).
	dotted := parseHelp(pad40("--fim-qwen-1.5b-default", "use default Qwen 2.5 Coder 1.5B"))
	if len(dotted) != 1 || len(dotted[0].longFlags) != 1 || dotted[0].longFlags[0] != "--fim-qwen-1.5b-default" {
		t.Errorf("dotted alias misparsed: %+v", dotted)
	}
}

const synthProbes = `--ctx-size	VALUED
--swa-full	STANDALONE
--brand-new-flag	VALUED
--new-bool	STANDALONE
--no-new-bool	STANDALONE
--draft	REMOVED
--draft-n	REMOVED
--draft-max	REMOVED
--tensor-split	VALUED
--spec-type	VALUED
--rope-scaling	VALUED
--threads	VALUED
--thread-count	VALUED
`

// TestParseEndToEnd runs parse mode in a sandbox with a minimal catalog, and
// checks the generated blocks plus idempotency (a second run must be a no-op).
func TestParseEndToEnd(t *testing.T) {
	dir := t.TempDir()
	mustWrite := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("pkg/ui/web/app.js", `// head
var commonFlags = [
  '--ctx-size','--swa-full','--tensor-split',
];
var standaloneFlags = {
  '--swa-full':1,
};
var flagHints = {
  '--ctx-size': 'context size in tokens',
  '--swa-full': 'use full-size SWA cache',
  '--tensor-split': 'GPU split proportions, e.g. 3,1',
};
// tail
`)
	mustWrite("pkg/model/model.go", `package model

var standaloneFlags = map[string]bool{
	"--swa-full": true,
}
`)
	mustWrite("help.txt", syntheticHelp)
	mustWrite("probes.tsv", synthProbes)

	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if err := cmdParse("help.txt", "probes.tsv", false, false); err != nil {
		t.Fatal(err)
	}
	js, _ := os.ReadFile("pkg/ui/web/app.js")
	goFile, _ := os.ReadFile("pkg/model/model.go")
	s, g := string(js), string(goFile)

	for _, want := range []string{"'--brand-new-flag'", "'--new-bool'", "'--no-new-bool'", blockStart, blockEnd} {
		if !strings.Contains(s, want) {
			t.Errorf("app.js missing %q", want)
		}
	}
	for _, gone := range []string{"'--draft'", "'--draft-max'"} {
		if strings.Contains(s, gone) {
			t.Errorf("app.js should not contain removed stub %q", gone)
		}
	}
	if !strings.Contains(s, "'--ctx-size': 'context size in tokens'") {
		t.Errorf("hand-tuned hint not preserved:\n%s", s)
	}
	if !strings.Contains(s, "'--brand-new-flag': 'brand new valued flag'") {
		t.Errorf("derived hint missing:\n%s", s)
	}
	if !strings.Contains(g, "\"--new-bool\": true") || !strings.Contains(g, "\"--no-new-bool\": true") {
		t.Errorf("go standalone set missing new-bool pair:\n%s", g)
	}
	if strings.Contains(g, "\"--ctx-size\": true") {
		t.Errorf("valued flag wrongly in standalone set:\n%s", g)
	}
	before := s + "|" + g
	if err := cmdParse("help.txt", "probes.tsv", false, false); err != nil {
		t.Fatal(err)
	}
	js2, _ := os.ReadFile("pkg/ui/web/app.js")
	go2, _ := os.ReadFile("pkg/model/model.go")
	if string(js2)+"|"+string(go2) != before {
		os.WriteFile("/tmp/idem-js1.js", []byte(s), 0644)
		os.WriteFile("/tmp/idem-js2.js", js2, 0644)
		os.WriteFile("/tmp/idem-go1.go", []byte(g), 0644)
		os.WriteFile("/tmp/idem-go2.go", go2, 0644)
		t.Errorf("second parse run is not idempotent (diffs in /tmp/idem-*)")
	}
}

// TestParseShrinkGuard verifies that a catalog shrinking by more than half
// is refused without --force and allowed with it.
func TestParseShrinkGuard(t *testing.T) {
	dir := t.TempDir()
	mustWrite := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("pkg/ui/web/app.js", `var commonFlags = [
  '--a-1','--a-2','--a-3',
  '--a-4','--a-5',
];
var standaloneFlags = {
  '--a-1':1,
};
var flagHints = {
  '--a-1': 'x',
  '--a-2': 'x',
  '--a-3': 'x',
  '--a-4': 'x',
  '--a-5': 'x',
};
`)
	mustWrite("pkg/model/model.go", "package model\n\nvar standaloneFlags = map[string]bool{\n\t\"--a-1\": true,\n}\n")
	mustWrite("help.txt", pad40("--tiny-only N", "the only flag")+"\n")
	mustWrite("probes.tsv", "--tiny-only\tVALUED\n")

	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if err := cmdParse("help.txt", "probes.tsv", false, false); err == nil {
		t.Fatalf("shrink without --force must be refused")
	}
	if err := cmdParse("help.txt", "probes.tsv", true, false); err != nil {
		t.Fatal(err)
	}
}

