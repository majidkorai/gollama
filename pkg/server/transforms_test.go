package server

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// thinkOpen/thinkClose are the literal reasoning tags the transforms look
// for. They are built from hex escapes (0x3c = '<', 0x3e = '>') because
// tooling in this session strips angle-bracket pairs from file content.
var (
	thinkOpen  = string(rune(0x3c)) + "think" + string(rune(0x3e))
	thinkClose = string(rune(0x3c)) + "/think" + string(rune(0x3e))
)

// deltaJSON builds an SSE data payload with a choices[0].delta carrying the
// given fields. HTML escaping is disabled so think tags survive verbatim.
func deltaJSON(delta map[string]interface{}) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]interface{}{
		"choices": []map[string]interface{}{{"index": 0, "delta": delta}},
	})
	return strings.TrimRight(buf.String(), "\n")
}

// messageJSON builds a non-streaming payload with choices[0].message.content.
func messageJSON(content string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]interface{}{
		"choices": []map[string]interface{}{{"index": 0, "message": map[string]interface{}{"role": "assistant", "content": content}}},
	})
	return strings.TrimRight(buf.String(), "\n")
}

// deltaOf unmarshals a payload and returns choices[0].delta (falling back to
// choices[0].message), failing the test if neither is present.
func deltaOf(t *testing.T, data string) map[string]interface{} {
	t.Helper()
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, data)
	}
	choices, _ := obj["choices"].([]interface{})
	if len(choices) == 0 {
		t.Fatalf("no choices in output: %s", data)
	}
	choice, _ := choices[0].(map[string]interface{})
	for _, k := range []string{"delta", "message"} {
		if m, ok := choice[k].(map[string]interface{}); ok {
			return m
		}
	}
	t.Fatalf("no delta/message in output: %s", data)
	return nil
}

func strp(s string) *string { return &s }

// ── extractThinkStream ─────────────────────────────────────

type thinkStep struct {
	content     string
	wantIn      bool
	wantBuf     string
	wantReason  *string // nil = field must be absent
	wantContent *string // nil = field must be absent
	passthrough bool    // output must be byte-identical to input
}

func TestExtractThinkStream(t *testing.T) {
	cases := []struct {
		name  string
		steps []thinkStep
	}{
		{
			name: "content without think tags passes through unchanged",
			steps: []thinkStep{
				{content: "hello world", passthrough: true},
			},
		},
		{
			name: "complete think block in one chunk",
			steps: []thinkStep{
				{
					content:     "pre" + thinkOpen + "reason" + thinkClose + "post",
					wantIn:      false,
					wantBuf:     "",
					wantReason:  strp("reason"),
					wantContent: strp("prepost"),
				},
			},
		},
		{
			name: "think block split across three chunks",
			steps: []thinkStep{
				{
					content:     "pre" + thinkOpen + "rea",
					wantIn:      true,
					wantBuf:     "rea",
					wantContent: strp("pre"),
				},
				{
					content: "son",
					wantIn:  true,
					wantBuf: "reason",
				},
				{
					content:     "ing" + thinkClose + "post",
					wantIn:      false,
					wantBuf:     "",
					wantReason:  strp("reasoning"),
					wantContent: strp("post"),
				},
			},
		},
		{
			name: "think tag at the very start of content",
			steps: []thinkStep{
				{
					content: thinkOpen + "r",
					wantIn:  true,
					wantBuf: "r",
				},
			},
		},
		{
			name: "think tag at the end of content, closed by the next chunk",
			steps: []thinkStep{
				{
					content:     "post" + thinkOpen,
					wantIn:      true,
					wantBuf:     "",
					wantContent: strp("post"),
				},
				{
					// Empty reasoning is emitted (buffer was empty) —
					// documents current behavior.
					content:    thinkClose,
					wantIn:     false,
					wantBuf:    "",
					wantReason: strp(""),
				},
			},
		},
		{
			name: "lone close tag without an open tag is ignored",
			steps: []thinkStep{
				{
					// The close tag does not contain the open tag as a
					// substring (the slash breaks it), so no buffering
					// starts and the chunk passes through unchanged.
					content:     "hello " + thinkClose,
					passthrough: true,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var inThink bool
			var buf string
			for i, step := range tc.steps {
				in := deltaJSON(map[string]interface{}{"content": step.content})
				out := extractThinkStream(in, &inThink, &buf)
				if step.passthrough {
					if out != in {
						t.Fatalf("step %d: expected byte-identical passthrough\ngot:  %s\nwant: %s", i, out, in)
					}
					if inThink || buf != "" {
						t.Fatalf("step %d: state changed (inThink=%v buf=%q) on passthrough", i, inThink, buf)
					}
					continue
				}
				d := deltaOf(t, out)
				if got, _ := d["reasoning_content"]; (got != nil) != (step.wantReason != nil) {
					t.Fatalf("step %d: reasoning_content present=%v, want %v (output %s)",
						i, got != nil, step.wantReason != nil, out)
				}
				if step.wantReason != nil {
					if got, _ := d["reasoning_content"].(string); got != *step.wantReason {
						t.Fatalf("step %d: reasoning_content = %q, want %q", i, got, *step.wantReason)
					}
				}
				if got, _ := d["content"]; (got != nil) != (step.wantContent != nil) {
					t.Fatalf("step %d: content present=%v, want %v (output %s)",
						i, got != nil, step.wantContent != nil, out)
				}
				if step.wantContent != nil {
					if got, _ := d["content"].(string); got != *step.wantContent {
						t.Fatalf("step %d: content = %q, want %q", i, got, *step.wantContent)
					}
				}
				if inThink != step.wantIn || buf != step.wantBuf {
					t.Fatalf("step %d: state = (inThink=%v buf=%q), want (inThink=%v buf=%q)",
						i, inThink, buf, step.wantIn, step.wantBuf)
				}
			}
		})
	}
}

// ── stripContentThinkTags / stripThinkTags ─────────────────

func TestStripContentThinkTags(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"no tags", "hello", "hello"},
		{"single think block", "pre" + thinkOpen + "mid" + thinkClose + "post", "prepost"},
		{"multiple think blocks", thinkOpen + "a" + thinkClose + "mid" + thinkOpen + "b" + thinkClose, "mid"},
		{"unclosed think block is left alone", "pre" + thinkOpen + "mid", "pre" + thinkOpen + "mid"},
		{"empty think block", thinkOpen + thinkClose, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := stripContentThinkTags([]byte(messageJSON(tt.content)))
			d := deltaOf(t, string(out))
			if got, _ := d["content"].(string); got != tt.want {
				t.Fatalf("content = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── stripReasoningContent ──────────────────────────────────

func TestStripReasoningContent(t *testing.T) {
	t.Run("delta", func(t *testing.T) {
		in := deltaJSON(map[string]interface{}{"reasoning_content": "r", "content": "c"})
		out := stripReasoningContent([]byte(in))
		d := deltaOf(t, string(out))
		if _, ok := d["reasoning_content"]; ok {
			t.Fatalf("reasoning_content was not stripped: %s", out)
		}
		if got, _ := d["content"].(string); got != "c" {
			t.Fatalf("content = %q, want %q", got, "c")
		}
	})
	t.Run("message", func(t *testing.T) {
		in := messageJSON("c")
		var obj map[string]interface{}
		json.Unmarshal([]byte(in), &obj)
		msg := obj["choices"].([]interface{})[0].(map[string]interface{})["message"].(map[string]interface{})
		msg["reasoning_content"] = "r"
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.Encode(obj)
		out := stripReasoningContent([]byte(strings.TrimRight(buf.String(), "\n")))
		d := deltaOf(t, string(out))
		if _, ok := d["reasoning_content"]; ok {
			t.Fatalf("reasoning_content was not stripped: %s", out)
		}
		if got, _ := d["content"].(string); got != "c" {
			t.Fatalf("content = %q, want %q", got, "c")
		}
	})
	t.Run("no reasoning field is a no-op", func(t *testing.T) {
		in := deltaJSON(map[string]interface{}{"content": "c"})
		out := stripReasoningContent([]byte(in))
		d := deltaOf(t, string(out))
		if got, _ := d["content"].(string); got != "c" {
			t.Fatalf("content = %q, want %q", got, "c")
		}
	})
}

// ── mergeReasoningContent ──────────────────────────────────
// NOTE: this function is currently dead code — the merge_reasoning profile
// toggle does not call it (see ROBUSTNESS_PLAN.md P2-T1). These tests pin the
// intended behavior so the wiring fix has a spec to satisfy.

func TestMergeReasoningContent(t *testing.T) {
	tests := []struct {
		name  string
		delta map[string]interface{}
		want  string
	}{
		// Documents current behavior: reasoning is appended AFTER any existing
		// content in the same chunk ("c" + "r" → "cr").
		{"reasoning appended after existing content", map[string]interface{}{"reasoning_content": "r", "content": "c"}, "cr"},
		{"reasoning only becomes content", map[string]interface{}{"reasoning_content": "r"}, "r"},
		{"content only is unchanged", map[string]interface{}{"content": "c"}, "c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := mergeReasoningContent([]byte(deltaJSON(tt.delta)))
			d := deltaOf(t, string(out))
			if _, ok := d["reasoning_content"]; ok {
				t.Fatalf("reasoning_content was not moved: %s", out)
			}
			if got, _ := d["content"].(string); got != tt.want {
				t.Fatalf("content = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── convertCompleteThink ───────────────────────────────────

func TestConvertCompleteThink(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		passthrough bool
		wantReason  *string // nil = field must be absent
		wantContent *string // nil = field must be absent
	}{
		{"no think tags", "hello", true, nil, strp("hello")},
		{"unclosed think block", "pre" + thinkOpen + "mid", true, nil, strp("pre" + thinkOpen + "mid")},
		{
			"think block in the middle",
			"pre" + thinkOpen + "r" + thinkClose + "post",
			false,
			strp("r"),
			strp("prepost"),
		},
		{
			"content is only a think block",
			thinkOpen + "r" + thinkClose,
			false,
			strp("r"),
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := messageJSON(tt.content)
			out := convertCompleteThink([]byte(in))
			if tt.passthrough {
				if string(out) != in {
					t.Fatalf("expected byte-identical passthrough\ngot:  %s\nwant: %s", out, in)
				}
				return
			}
			d := deltaOf(t, string(out))
			if got, _ := d["reasoning_content"]; (got != nil) != (tt.wantReason != nil) {
				t.Fatalf("reasoning_content present=%v, want %v (output %s)", got != nil, tt.wantReason != nil, out)
			}
			if tt.wantReason != nil {
				if got, _ := d["reasoning_content"].(string); got != *tt.wantReason {
					t.Fatalf("reasoning_content = %q, want %q", got, *tt.wantReason)
				}
			}
			if got, _ := d["content"]; (got != nil) != (tt.wantContent != nil) {
				t.Fatalf("content present=%v, want %v (output %s)", got != nil, tt.wantContent != nil, out)
			}
			if tt.wantContent != nil {
				if got, _ := d["content"].(string); got != *tt.wantContent {
					t.Fatalf("content = %q, want %q", got, *tt.wantContent)
				}
			}
		})
	}
}

// ── tool schema sanitizers ─────────────────────────────────

// patternByName walks a JSON schema tree and records every "pattern" value
// keyed by the property path (e.g. "list.items").
func patternByName(v interface{}, name string, into map[string]string) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return
	}
	if p, ok := m["pattern"].(string); ok {
		into[name] = p
	}
	if props, ok := m["properties"].(map[string]interface{}); ok {
		for k, child := range props {
			patternByName(child, k, into)
		}
	}
	if items, ok := m["items"]; ok {
		patternByName(items, name+".items", into)
	}
}

func TestSanitizeSchemaPatterns(t *testing.T) {
	in := `{"type":"object","properties":{
		"id":{"type":"string","pattern":"\\d+"},
		"name":{"type":"string","pattern":"^\\w+$"},
		"list":{"type":"array","items":{"type":"string","pattern":"\\s*"}}
	}}`
	var v interface{}
	if err := json.Unmarshal([]byte(in), &v); err != nil {
		t.Fatal(err)
	}
	sanitizeSchemaPatterns(v)

	byName := map[string]string{}
	patternByName(v, "", byName)

	checks := map[string]string{
		"id":      "^[0-9]+$",      // \d replaced, anchors added
		"name":    "^[a-zA-Z0-9_]+$", // \w replaced, existing anchors kept
		"list":    "",              // no pattern on the array itself
		"list.items": "^[ ]*$",     // \s → [ ] (space only), anchors added
	}
	for name, want := range checks {
		if byName[name] != want {
			t.Errorf("pattern %q = %q, want %q (all: %v)", name, byName[name], want, byName)
		}
	}
}

func TestStripAdditionalProperties(t *testing.T) {
	in := `{"type":"object","properties":{
		"a":{"type":"object","additionalProperties":false,"properties":{"b":{"additionalProperties":true}}},
		"arr":{"type":"array","items":{"additionalProperties":false}}
	},"required":["a"]}`
	var v interface{}
	if err := json.Unmarshal([]byte(in), &v); err != nil {
		t.Fatal(err)
	}
	stripAdditionalProperties(v)
	out, _ := json.Marshal(v)
	if strings.Contains(string(out), "additionalProperties") {
		t.Fatalf("additionalProperties survived: %s", out)
	}
}

func TestSimplifyToolSchemas(t *testing.T) {
	in := `{"tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object","properties":{
		"x":{"type":"object","properties":{"y":{"type":"string"}}},
		"z":{"type":"array","items":{"type":"object","properties":{"w":{"type":"string"}}}}
	}}}}]}`
	var v interface{}
	if err := json.Unmarshal([]byte(in), &v); err != nil {
		t.Fatal(err)
	}
	simplifyToolSchemas(v)

	tools := v.(map[string]interface{})["tools"].([]interface{})
	fn := tools[0].(map[string]interface{})["function"].(map[string]interface{})
	params := fn["parameters"].(map[string]interface{})
	props := params["properties"].(map[string]interface{})

	if _, ok := props["x"].(map[string]interface{})["properties"]; ok {
		t.Fatal("nested object properties were not stripped from x")
	}
	if _, ok := props["z"].(map[string]interface{})["items"]; ok {
		t.Fatal("array items were not stripped from z")
	}
	// Untouched siblings remain.
	if params["type"] != "object" {
		t.Fatal("parameter type was modified")
	}
}
