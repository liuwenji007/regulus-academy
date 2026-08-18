package llm

import (
	"strings"
	"unicode/utf8"
)

// sanitizeLLMJSON 把模型常见的非法 JSON 修到 encoding/json 可解析：
// 字符串内未转义控制字符（如 PDF 抽字带来的 \x18）、非法转义（如 \'）。
func sanitizeLLMJSON(s string) string {
	if s == "" {
		return s
	}
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	var b strings.Builder
	b.Grow(len(s) + 16)
	inString := false
	for i := 0; i < len(s); {
		c := s[i]
		if !inString {
			if c == '"' {
				inString = true
				b.WriteByte(c)
				i++
				continue
			}
			if c < 0x20 && c != '\t' && c != '\n' && c != '\r' {
				i++
				continue
			}
			b.WriteByte(c)
			i++
			continue
		}

		if c == '\\' {
			if i+1 >= len(s) {
				i++
				continue
			}
			next := s[i+1]
			switch next {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				b.WriteByte('\\')
				b.WriteByte(next)
				i += 2
			case 'u':
				if i+6 <= len(s) && isJSONHex4(s[i+2:i+6]) {
					b.WriteString(s[i : i+6])
					i += 6
				} else {
					i++
				}
			default:
				i++
			}
			continue
		}
		if c == '"' {
			inString = false
			b.WriteByte(c)
			i++
			continue
		}
		if c < 0x20 {
			writeJSONControlEscape(&b, c)
			i++
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

func isJSONHex4(s string) bool {
	if len(s) < 4 {
		return false
	}
	for i := 0; i < 4; i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}

func writeJSONControlEscape(b *strings.Builder, c byte) {
	switch c {
	case '\n':
		b.WriteString(`\n`)
	case '\r':
		b.WriteString(`\r`)
	case '\t':
		b.WriteString(`\t`)
	default:
		const hex = "0123456789abcdef"
		b.WriteString(`\u00`)
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0F])
	}
}
