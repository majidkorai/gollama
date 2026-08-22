package model

import (
	"strings"
)

// Flags is an ordered, typed representation of llama-server CLI flags.
//
// It records each flag's value (if any) and the order in which flags first
// appear, so parsing, merging, and serialization are deterministic. This
// replaces the old pile of positional heuristics (sanitizeFlags + ad-hoc
// merge loops) — see ROBUSTNESS_PLAN.md P5-T3.
//
// A flag is "standalone" (boolean, no value) when it is in the
// standaloneFlags set, or when its --no- negation's positive form is
// (see isStandaloneFlag / standaloneCounterpart).
type Flags struct {
	order  []string           // flag names (with leading --) in first-seen order
	values map[string]*string // flag name → value; nil for standalone flags
}

// ParseFlags parses a flat []string of CLI args into a Flags.
//
// It handles both "--key value" and "--key=value" forms and recognizes
// standalone (no-value) flags. Orphaned values — a value with no preceding
// flag key, or a value following a standalone flag — are dropped, matching
// the old sanitizeFlags behavior.
func ParseFlags(args []string) Flags {
	f := Flags{values: make(map[string]*string)}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			// Orphaned value without a preceding flag key — drop it.
			continue
		}
		key := arg
		var value *string
		switch {
		case strings.Contains(arg, "="):
			// "--key=value" form: the value is explicit.
			eq := strings.IndexByte(arg, '=')
			key = arg[:eq]
			v := arg[eq+1:]
			value = &v
		case !isStandaloneFlag(arg) && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--"):
			// "--key value" form: consume the next non-flag token.
			v := args[i+1]
			value = &v
			i++
		}
		if _, exists := f.values[key]; !exists {
			f.order = append(f.order, key)
		}
		f.values[key] = value
	}
	return f
}

// Args serializes the flags back to a flat []string, preserving order.
// Standalone flags are emitted alone; valued flags as "--key" "value".
func (f Flags) Args() []string {
	if len(f.order) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(f.order)*2)
	for _, key := range f.order {
		v, ok := f.values[key]
		if !ok || v == nil {
			out = append(out, key)
			continue
		}
		out = append(out, key, *v)
	}
	return out
}

// Merge returns f overridden by override:
//   - For valued flags, override wins — the base flag and its value are
//     dropped and the override's are appended at the end.
//   - For standalone flags, override sets/unsets: a standalone flag in
//     override removes its --no-/bare counterpart from base, then is
//     appended (e.g. --no-verbose unsets a base --verbose).
//
// Result order is: base flags (minus those overridden or unset), then
// override flags. This matches the old ProfileFlags / manager.Start merge
// behavior, now with correct standalone set/unset.
func (f Flags) Merge(override Flags) Flags {
	result := Flags{values: make(map[string]*string)}

	overrideKeys := make(map[string]bool, len(override.order))
	for _, k := range override.order {
		overrideKeys[k] = true
	}

	// Standalone flags in override unset their counterpart from base.
	unset := make(map[string]bool)
	for _, k := range override.order {
		if !isStandaloneFlag(k) {
			continue
		}
		if c := standaloneCounterpart(k); c != "" && isStandaloneFlag(c) {
			unset[c] = true
		}
	}

	for _, k := range f.order {
		if overrideKeys[k] || unset[k] {
			continue
		}
		result.order = append(result.order, k)
		result.values[k] = f.values[k]
	}
	for _, k := range override.order {
		if _, exists := result.values[k]; !exists {
			result.order = append(result.order, k)
		}
		result.values[k] = override.values[k]
	}
	return result
}

// standaloneCounterpart returns the paired standalone flag for name:
// --no-X ↔ --X. Returns "" when name is not a "--" flag.
func standaloneCounterpart(name string) string {
	switch {
	case strings.HasPrefix(name, "--no-"):
		return "--" + name[len("--no-"):]
	case strings.HasPrefix(name, "--"):
		return "--no-" + name[len("--"):]
	default:
		return ""
	}
}

// isStandaloneFlag reports whether name is a boolean (no-value) flag.
// A flag is standalone if it is in the standaloneFlags set, or if it is a
// --no-X form whose positive --X is a known standalone flag (deriving the
// negation form without hardcoding every --no-* variant). The derivation is
// one-way on purpose: --X is NOT made standalone just because --no-X is
// (e.g. --flash-attn stays valued even though --no-flash-attn is standalone).
func isStandaloneFlag(name string) bool {
	if standaloneFlags[name] {
		return true
	}
	if strings.HasPrefix(name, "--no-") {
		positive := "--" + name[len("--no-"):]
		if standaloneFlags[positive] {
			return true
		}
	}
	return false
}
