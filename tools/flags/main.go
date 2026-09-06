// Command flags regenerates gollama's llama.cpp flag catalog from the live
// binary's --help output plus probe results, so the catalog in
// pkg/ui/web/app.js and pkg/model/model.go stays complete and accurate.
//
// Why: the flag list used to be hand-maintained against an old build, and
// llama.cpp flags churn every release (added, removed, renamed). The
// llama-server binary's own --help output is the authoritative, complete
// list of every parameter (common, sampling, speculative, and server
// sections, including locally-patched flags), and running the flag through
// the binary tells us whether it is standalone (boolean), valued, invalid,
// or a removed stub.
//
// Usage (three steps):
//
//	# 1. Dump the help from the binary (on the machine that has it):
//	llama-server --help > help.txt
//	
//	# 2. Probe every long flag against that binary (same machine):
//	GOOS=linux GOARCH=amd64 go build -o flags-probe ./tools/flags
//	scp flags-probe help.txt <host>:... && ssh <host> './flags-probe probe help.txt /path/to/llama-server' > probes.tsv
//	
//	# 3. Regenerate the catalog blocks (repo root):
//	go run ./tools/flags parse help.txt probes.tsv
//
// Probe protocol — for each argument group (every alias on one --help line
// is the same argument, so one probe covers all its aliases), the binary is
// run with the flag plus --list-devices, which prints its device list and
// exits immediately at parse time:
//
//	A: F --list-devices
//	  "Available devices"                 -> F parsed as boolean, probe flag ran: STANDALONE
//	  "error: invalid argument: F"        -> F does not exist in this build: INVALID
//	  "the argument has been removed"     -> F is a removed stub: REMOVED
//	  anything else (e.g. F consumed --list-devices as its value) -> probe B
//	B: F __gollama_probe__ --list-devices
//	  "Available devices"                 -> value accepted: VALUED
//	  "error while handling argument F"   -> flag exists, probe value rejected: VALUED
//	  "error: invalid argument: __gollama_probe__" -> F took no value (orphan rejected): STANDALONE
//
// Immediate-handler flags (they exit while parsing, before the probe flag is
// seen) are classified statically: --list-devices and --cache-list are
// STANDALONE; --help/--usage/--version/--completion-bash are SKIPPED
// (meta flags, never part of a serving config).

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

const probeToken = "__gollama_probe__"

// ── Flag kinds ──────────────────────────────────────────────────────────────

type kind int

const (
	kindUnknown kind = iota
	kindStandalone
	kindValued
	kindInvalid
	kindRemoved
	kindSkipped
)

func (k kind) String() string {
	switch k {
	case kindStandalone:
		return "STANDALONE"
	case kindValued:
		return "VALUED"
	case kindInvalid:
		return "INVALID"
	case kindRemoved:
		return "REMOVED"
	case kindSkipped:
		return "SKIPPED"
	}
	return "UNKNOWN"
}

// probeOverrides classifies flags statically: immediate-handler flags (they
// exit during argument parsing, so a probe never reaches the marker) and
// meta flags that must not enter the catalog.
var probeOverrides = map[string]kind{
	"--help":            kindSkipped,
	"--usage":           kindSkipped,
	"--version":         kindSkipped,
	"--completion-bash": kindSkipped,
	"--list-devices":    kindStandalone,
	"--cache-list":      kindStandalone,
}

// ── --help dump parsing ─────────────────────────────────────────────────────

// entry is one argument definition from the --help dump. Every alias on the
// first line belongs to the same argument (llama.cpp prints args and their
// --no- negation on one line), so all aliases share one probe result.
type entry struct {
	longFlags []string // long aliases in listed order
	help      string   // full help text, lines joined
	section   string
}

const helpColumn = 40 // llama.cpp pads the alias column to 40 (common/arg.cpp to_string)

// aliasRe extracts an alias token. The dot is included because some flag
// names carry one (--fim-qwen-1.5b-default).
var aliasRe = regexp.MustCompile(`^(-{1,2}[A-Za-z0-9.-]+)`)

// parseHelp parses a llama-server --help dump into entries.
//
// Format contract (verified against common/arg.cpp to_string, b10815):
// the alias+hint column is padded to exactly 40 — but only when it is
// <=37 chars; longer columns push the help to the next line. So an entry
// line either is entirely aliases (len<=40, or len>40 with a non-space at
// index 39) or carries inline help starting at index 40, and the pad
// guarantees a space at index 39 exactly in that second case.
func parseHelp(text string) []entry {
	lines := strings.Split(text, "\n")
	var entries []entry
	var cur *entry
	section := ""
	flush := func() {
		if cur != nil {
			entries = append(entries, *cur)
			cur = nil
		}
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "-----") {
			flush()
			section = strings.Trim(strings.TrimSpace(line), "- ")
			continue
		}
		if len(line) > 1 && line[0] == '-' && (line[1] == '-' || isIdentChar(line[1])) {
			flush()
			flagPart, inlineHelp := line, ""
			if len(line) > helpColumn && line[helpColumn-1] == ' ' {
				flagPart, inlineHelp = line[:helpColumn], strings.TrimSpace(line[helpColumn:])
			}
			cur = &entry{help: inlineHelp, section: section}
			for _, a := range longAliases(flagPart) {
				cur.longFlags = append(cur.longFlags, a)
			}
			continue
		}
		if cur == nil {
			continue
		}
		if t := strings.TrimSpace(line); t != "" {
			cur.help = strings.TrimSpace(cur.help + " " + t)
		}
	}
	flush()
	return entries
}

func isIdentChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// longAliases extracts the long (--name) aliases from the alias column of a
// help line. The trailing value hint on the last alias (" N", " FNAME", ...)
// and hint fragments broken out by commas ("N1", "...", "yarn}") are ignored
// because only real aliases start with "-".
func longAliases(flagPart string) []string {
	var out []string
	for _, tok := range strings.Split(flagPart, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if m := aliasRe.FindString(tok); m != "" && strings.HasPrefix(m, "--") {
			out = append(out, m)
		}
	}
	return out
}

// removedStub reports whether the help marks this argument as removed
// (llama.cpp keeps a no-op stub that rejects the flag at parse time).
func (e entry) removedStub() bool {
	return strings.Contains(e.help, "the argument has been removed")
}

func (e entry) isSkipped() bool {
	if len(e.longFlags) == 0 {
		return true
	}
	for _, f := range e.longFlags {
		if k, ok := probeOverrides[f]; !ok || k != kindSkipped {
			return false
		}
	}
	return true
}

// ── probe mode ──────────────────────────────────────────────────────────────

func runProbe(binary string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

// classify probes one flag name against the binary (see the package comment
// for the protocol). It returns the kind plus the raw output for UNKNOWNs.
func classify(binary, flag string) (kind, string) {
	if k, ok := probeOverrides[flag]; ok {
		return k, "(static)"
	}
	a := runProbe(binary, flag, "--list-devices")
	switch {
	case strings.Contains(a, "Available devices"):
		return kindStandalone, ""
	case strings.Contains(a, "error: invalid argument: "+flag):
		return kindInvalid, ""
	case strings.Contains(a, "the argument has been removed"):
		return kindRemoved, ""
	}
	b := runProbe(binary, flag, probeToken, "--list-devices")
	switch {
	case strings.Contains(b, "Available devices"):
		return kindValued, ""
	case strings.Contains(b, "error while handling argument \""+flag+"\""):
		return kindValued, ""
	case strings.Contains(b, "error: invalid argument: "+probeToken):
		return kindStandalone, ""
	case strings.Contains(b, "error: invalid argument: "+flag):
		return kindInvalid, ""
	case strings.Contains(b, "the argument has been removed"):
		return kindRemoved, ""
	}
	return kindUnknown, a + "\n--- probe B output ---\n" + b
}

func cmdProbe(helpFile, binary string) error {
	helpText, err := os.ReadFile(helpFile)
	if err != nil {
		return err
	}
	entries := parseHelp(string(helpText))
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for i, e := range entries {
		if e.isSkipped() {
			for _, f := range e.longFlags {
				fmt.Fprintf(w, "%s\t%s\n", f, probeOverrides[f])
			}
			continue
		}
		if e.removedStub() {
			for _, f := range e.longFlags {
				fmt.Fprintf(w, "%s\t%s\n", f, kindRemoved)
			}
			fmt.Fprintf(os.Stderr, "[%d/%d] %s REMOVED\n", i+1, len(entries), e.longFlags[0])
			continue
		}
		k, out := classify(binary, e.longFlags[0])
		if k == kindUnknown {
			fmt.Fprintf(os.Stderr, "[%d/%d] %s UNKNOWN — manual review:\n%s\n", i+1, len(entries), e.longFlags[0], out)
		} else {
			fmt.Fprintf(os.Stderr, "[%d/%d] %s %s\n", i+1, len(entries), e.longFlags[0], k)
		}
		for _, f := range e.longFlags {
			fmt.Fprintf(w, "%s\t%s\n", f, k)
		}
	}
	return nil
}

// ── parse mode (catalog generation) ─────────────────────────────────────────

// Catalog blocks are wrapped in markers so re-runs replace exactly the
// generated region. The first run (pre-marker) falls back to regex matches.
const (
	blockStart = "// BEGIN generated flag catalog (tools/flags) — do not edit by hand"
	blockEnd   = "// END generated flag catalog"
)

var (
	reCommonFlags = regexp.MustCompile(`(?ms)^var commonFlags = \[.*?\n\];`)
	reStandalone  = regexp.MustCompile(`(?ms)^var standaloneFlags = \{.*?\n\};`)
	reHints       = regexp.MustCompile(`(?ms)^var flagHints = \{.*?\n\};`)
	reGoStandalone = regexp.MustCompile(`(?ms)^var standaloneFlags = map\[string\]bool\{.*?\n\}`)
	reFlagName    = regexp.MustCompile(`'(--[a-z0-9.-]+)'`)
	reHintPair    = regexp.MustCompile(`'(--[a-z0-9-]+)'\s*:\s*'((?:[^'\\]|\\.)*)'`)
)

// cmdParse regenerates the catalog. force permits a catalog that shrinks by
// more than half (a sanity tripwire against running with the wrong help
// dump or probes); dryRun prints the summary without writing any file.
func cmdParse(helpFile, probesFile string, force, dryRun bool) error {
	helpText, err := os.ReadFile(helpFile)
	if err != nil {
		return err
	}
	probes, err := os.ReadFile(probesFile)
	if err != nil {
		return err
	}
	kindByFlag := map[string]kind{}
	for _, line := range strings.Split(string(probes), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		switch k := parts[1]; k {
		case "STANDALONE":
			kindByFlag[parts[0]] = kindStandalone
		case "VALUED":
			kindByFlag[parts[0]] = kindValued
		case "INVALID":
			kindByFlag[parts[0]] = kindInvalid
		case "REMOVED":
			kindByFlag[parts[0]] = kindRemoved
		case "SKIPPED":
			kindByFlag[parts[0]] = kindSkipped
		}
	}

	// Current catalog (alias continuity + hand-tuned hint preservation).
	js, err := os.ReadFile("pkg/ui/web/app.js")
	if err != nil {
		return fmt.Errorf("run from the repo root: %w", err)
	}
	existingFlags := map[string]bool{}
	if m := reCommonFlags.FindString(string(js)); m != "" {
		for _, f := range reFlagName.FindAllStringSubmatch(m, -1) {
			existingFlags[f[1]] = true
		}
	}
	existingHints := map[string]string{}
	if m := reHints.FindString(string(js)); m != "" {
		for _, p := range reHintPair.FindAllStringSubmatch(m, -1) {
			existingHints[p[1]] = p[2]
		}
	}

	entries := parseHelp(string(helpText))
	newCatalog := map[string]string{} // flag -> hint
	newStandalone := map[string]bool{}
	var dropped, unknown, added []string

	for _, e := range entries {
		if e.isSkipped() {
			continue
		}
		k := kindByFlag[e.longFlags[0]]
		if e.removedStub() {
			k = kindRemoved
		}
		switch k {
		case kindInvalid, kindRemoved:
			for _, f := range e.longFlags {
				if existingFlags[f] {
					dropped = append(dropped, f+" ("+k.String()+")")
				}
			}
			continue
		case kindUnknown:
			unknown = append(unknown, e.longFlags[0]+" (no probe result)")
			continue
		case kindStandalone:
			for _, f := range e.longFlags {
				newStandalone[f] = true
			}
		}
		// Catalog names: every long alias of the group that is already in the
		// catalog (keeps existing profile flag names stable); otherwise the
		// first listed alias (llama.cpp lists the primary name first).
		var names []string
		for _, f := range e.longFlags {
			if existingFlags[f] {
				names = append(names, f)
			}
		}
		if len(names) == 0 {
			names = []string{e.longFlags[0]}
		}
		// Hint: hand-tuned hint for any alias of the group wins; otherwise
		// derive from the help text.
		hint := ""
		for _, f := range append(append([]string{}, names...), e.longFlags...) {
			if h, ok := existingHints[f]; ok && h != "" {
				hint = h
				break
			}
		}
		if hint == "" {
			hint = deriveHint(e.help)
		}
		for _, n := range names {
			if !existingFlags[n] {
				added = append(added, n)
			}
			newCatalog[n] = hint
		}
	}

	// Sanity: flags in the current catalog that vanished from the help dump.
	for f := range existingFlags {
		if _, ok := kindByFlag[f]; !ok {
			dropped = append(dropped, f+" (no longer in --help)")
		}
	}

	// Shrink tripwire: a healthy refresh changes the catalog by a handful of
	// flags; a >50% shrink almost always means the wrong help dump or probes
	// file (or a cwd mistake). Refuse unless forced.
	if len(existingFlags) > 0 && len(newCatalog)*2 < len(existingFlags) && !force {
		return fmt.Errorf("refusing to shrink catalog from %d to %d flags without --force (check the help dump, probes file, and cwd)",
			len(existingFlags), len(newCatalog))
	}

	// ── summary (printed even in dry-run) ──
	fmt.Printf("catalog: %d flags (was %d) — %d added, %d dropped%s\n", len(newCatalog), len(existingFlags), len(added), len(dropped), drySuffix(dryRun))
	for _, f := range sorted(added) {
		fmt.Printf("  + %s\n", f)
	}
	for _, f := range sorted(dropped) {
		fmt.Printf("  - %s\n", f)
	}
	fmt.Printf("standalone set: %d flags\n", len(newStandalone))
	if len(unknown) > 0 {
		fmt.Printf("WARNING — not cataloged (no probe result):\n")
		for _, f := range sorted(unknown) {
			fmt.Printf("  ? %s\n", f)
		}
	}
	if dryRun {
		fmt.Println("dry run — no files written")
		return nil
	}

	// ── render app.js blocks ──
	catalog := sorted(strKeys(newCatalog))
	common := "var commonFlags = [" + wrapList(catalog) + "\n];"
	standalone := "var standaloneFlags = {\n" + joinEntries(sorted(boolKeys(newStandalone)), func(f string) string { return "  '" + f + "':1" }) + "\n};"
	hints := "var flagHints = {\n" + joinEntries(catalog, func(f string) string { return "  '" + f + "': '" + newCatalog[f] + "'" }) + "\n};"

	generated := blockStart + "\n" +
		"// Derived from the live llama-server binary's --help output plus per-flag\n" +
		"// probes (see tools/flags). To refresh after a llama.cpp upgrade:\n" +
		"//   llama-server --help > help.txt && <flags-probe> probe help.txt <binary> > probes.tsv\n" +
		"//   go run ./tools/flags parse help.txt probes.tsv\n" +
		common + "\n" + standalone + "\n" + hints + "\n" + blockEnd

	jsOut := string(js)
	if start, end := markerRegion(jsOut); start >= 0 {
		jsOut = jsOut[:start] + generated + jsOut[end:]
	} else {
		// First run: no markers yet — replace the three blocks in place.
		m := reCommonFlags.FindStringIndex(jsOut)
		if m == nil || reStandalone.FindStringIndex(jsOut) == nil || reHints.FindStringIndex(jsOut) == nil {
			return fmt.Errorf("could not locate flag catalog blocks in app.js")
		}
		// The three blocks are contiguous: replace from the start of
		// commonFlags to the end of flagHints.
		start := m[0]
		hm := reHints.FindStringIndex(jsOut)
		jsOut = jsOut[:start] + generated + jsOut[hm[1]:]
	}
	if err := os.WriteFile("pkg/ui/web/app.js", []byte(jsOut), 0644); err != nil {
		return err
	}

	// ── render model.go block ──
	goFile := "pkg/model/model.go"
	goJS, err := os.ReadFile(goFile)
	if err != nil {
		return err
	}
	// The END marker sits inside the map literal, right before its closing
	// brace, so the marker region is exactly self-contained and re-runs are
	// idempotent. The first (regex) run consumes the map's closing brace, so
	// it appends "\n}"; the marker run leaves it in the surrounding file.
	goBlock := func(tail string) string {
		return blockStart + "\n" +
			"// Mirrored from the generated catalog in pkg/ui/web/app.js (kept in sync\n" +
			"// by the same tools/flags run). Used by ParseFlags/Merge to know which\n" +
			"// flags take no value.\n" +
			"var standaloneFlags = map[string]bool{\n" +
			// Go requires a trailing comma on multi-line composite
			// literals (the END marker comment separates the last entry
			// from the closing brace), so add one after the join.
			joinEntries(sorted(boolKeys(newStandalone)), func(f string) string { return "\t\"" + f + "\": true" }) + "," +
			tail
	}
	_ = goBlock
	goOut := string(goJS)
	if start, end := markerRegion(goOut); start >= 0 {
		goOut = goOut[:start] + goBlock("\n\t"+blockEnd) + goOut[end:]
	} else {
		m := reGoStandalone.FindStringIndex(goOut)
		if m == nil {
			return fmt.Errorf("could not locate standaloneFlags in model.go")
		}
		goOut = goOut[:m[0]] + goBlock("\n\t"+blockEnd+"\n}") + goOut[m[1]:]
	}
	if err := os.WriteFile(goFile, []byte(goOut), 0644); err != nil {
		return err
	}
	fmt.Println("wrote pkg/ui/web/app.js and pkg/model/model.go")
	return nil
}

func drySuffix(dry bool) string {
	if dry {
		return " [dry-run]"
	}
	return ""
}

// markerRegion returns the [start, end) byte range of a marker-wrapped
// region: from the beginning of the BEGIN marker line to (not including)
// the newline after the END marker text. The newline belongs to the
// surrounding file, so the replacement text must not end with one — the
// same bytes are produced on the first (marker-less, regex-based) run.
func markerRegion(s string) (int, int) {
	start := strings.Index(s, blockStart)
	if start < 0 {
		return -1, -1
	}
	lineStart := strings.LastIndex(s[:start], "\n") + 1
	if i := strings.Index(s[start:], blockEnd); i >= 0 {
		return lineStart, start + i + len(blockEnd)
	}
	return -1, -1
}

func boolKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func strKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func sorted(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// wrapList formats the catalog entries as comma-quoted lines wrapped at
// ~110 cols, without the outer brackets (the caller supplies them).
func wrapList(flags []string) string {
	var b strings.Builder
	line := ""
	for i, f := range flags {
		sep := ","
		if i == len(flags)-1 {
			sep = ""
		}
		tok := "'" + f + "'" + sep
		if line != "" && len(line)+len(tok)+1 > 110 {
			b.WriteString(line + "\n")
			line = "  " + tok
		} else {
			line += " " + tok
		}
	}
	b.WriteString(line)
	return "\n" + b.String()
}

func joinEntries(keys []string, f func(string) string) string {
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, f(k))
	}
	return strings.Join(parts, ",\n")
}

// deriveHint turns raw --help text into a short catalog hint (only used for
// flags without a hand-tuned hint).
var (
	reLink = regexp.MustCompile(`\[\(.*?\)\]\(.*?\)`)
	reEnv  = regexp.MustCompile(`\(env: [A-Z0-9_]+\)`)
)

func deriveHint(raw string) string {
	h := reLink.ReplaceAllString(raw, "")
	h = reEnv.ReplaceAllString(h, "")
	h = strings.ReplaceAll(h, "`", "")
	h = strings.Join(strings.Fields(h), " ")
	if len(h) > 100 {
		cut := strings.LastIndex(h[:100], " ")
		if cut > 60 {
			h = h[:cut] + "…"
		}
	}
	return h
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: flags probe <help.txt> <llama-server-binary> | flags parse <help.txt> <probes.tsv> [--dry-run] [--force]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "probe":
		if len(os.Args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: flags probe <help.txt> <llama-server-binary>")
			os.Exit(2)
		}
		err = cmdProbe(os.Args[2], os.Args[3])
	case "parse":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: flags parse <help.txt> <probes.tsv> [--dry-run] [--force]")
			os.Exit(2)
		}
		var force, dryRun bool
		for _, a := range os.Args[4:] {
			switch a {
			case "--force":
				force = true
			case "--dry-run":
				dryRun = true
			default:
				fmt.Fprintln(os.Stderr, "unknown option:", a)
				os.Exit(2)
			}
		}
		err = cmdParse(os.Args[2], os.Args[3], force, dryRun)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

