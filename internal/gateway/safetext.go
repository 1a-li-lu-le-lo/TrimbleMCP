package gateway

import (
	"strings"
	"unicode"
)

// maxUntrustedRunes bounds any single upstream string returned to an agent.
const maxUntrustedRunes = 512

// cleanUntrusted neutralizes upstream, user-authored text (project names,
// file names, descriptions) before it reaches a model. It removes control
// characters, bidirectional overrides, zero-width characters, and the Unicode
// tag block that can hide instructions, and truncates long values. It does
// not, and cannot, make the text trustworthy: outputs also label these fields
// as untrusted data.
func cleanUntrusted(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= maxUntrustedRunes {
			b.WriteString("…")
			break
		}
		switch {
		case r == '\t' || r == ' ':
			b.WriteRune(' ')
		case unicode.IsControl(r):
			continue
		case r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069: // bidi embeddings, isolates
			continue
		case r == 0x200B || r == 0x200C || r == 0x200D || r == 0x2060 || r == 0xFEFF: // zero-width
			continue
		case r >= 0xE0000 && r <= 0xE007F: // tag characters
			continue
		case unicode.Is(unicode.Co, r): // private use
			continue
		default:
			b.WriteRune(r)
		}
		n++
	}
	return b.String()
}
