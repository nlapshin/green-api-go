package jsonfmt

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

func PrettyJSON(raw []byte) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}
	if !json.Valid(raw) {
		return SafeSnippet(raw, 4096)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return SafeSnippet(raw, 4096)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return SafeSnippet(raw, 4096)
	}
	return string(b)
}

func SafeSnippet(b []byte, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 256
	}
	s := string(b)
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	var sb strings.Builder
	n := 0
	for _, r := range s {
		if r < 0x20 && r != '\t' {
			r = ' '
		}
		sb.WriteRune(r)
		n++
		if n >= maxRunes {
			sb.WriteString("…")
			break
		}
	}
	out := strings.TrimSpace(sb.String())
	if !utf8.ValidString(out) {
		return "…"
	}
	return out
}
