package ui

import (
	"os"
	"strings"
	"testing"
)

// TestPageMatchesReference guards the P5-T4 refactor: the assembled Page (from
// web/index.html + web/app.css + web/app.js via go:embed) must be byte-identical
// to the pre-refactor single const, captured in testdata/page_reference.html.
// If a web/ edit changes the served bytes, this fails — update the reference
// deliberately (and note the change) if the difference is intended.
func TestPageMatchesReference(t *testing.T) {
	ref, err := os.ReadFile("testdata/page_reference.html")
	if err != nil {
		t.Fatalf("read reference: %v", err)
	}
	if Page != string(ref) {
		// Report the first divergence to ease debugging.
		a, b := Page, string(ref)
		n := len(a)
		if len(b) < n {
			n = len(b)
		}
		for i := 0; i < n; i++ {
			if a[i] != b[i] {
				t.Fatalf("Page diverges from reference at byte %d:\n  page:     %q\n  reference: %q\n(page len %d, ref len %d)",
					i, around(a, i), around(b, i), len(a), len(b))
			}
		}
		t.Fatalf("Page length %d, reference length %d (one is a prefix of the other)", len(a), len(b))
	}
}

func around(s string, i int) string {
	lo, hi := i-20, i+20
	if lo < 0 {
		lo = 0
	}
	if hi > len(s) {
		hi = len(s)
	}
	return s[lo:hi]
}

// TestPageInlinesAssets sanity-checks that the CSS and JS are inlined (not left
// as placeholders) and that the page has the expected shell.
func TestPageInlinesAssets(t *testing.T) {
	if strings.Contains(Page, "__GOLLAMA_CSS__") || strings.Contains(Page, "__GOLLAMA_JS__") {
		t.Fatal("Page still contains placeholders — assembly failed")
	}
	for _, marker := range []string{"<!DOCTYPE html>", "</style>", "</script>", "</html>"} {
		if !strings.Contains(Page, marker) {
			t.Errorf("Page missing expected marker %q", marker)
		}
	}
}
