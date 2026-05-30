package channel

import "unicode/utf8"

const defaultChunkRunes = 3500

// SplitMessage 将长文本按 rune 数分段
func SplitMessage(text string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = defaultChunkRunes
	}
	if utf8.RuneCountInString(text) <= maxRunes {
		return []string{text}
	}
	var parts []string
	runes := []rune(text)
	for i := 0; i < len(runes); {
		end := i + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		parts = append(parts, string(runes[i:end]))
		i = end
	}
	if len(parts) > 1 {
		for i := range parts {
			parts[i] = "(" + itoa(i+1) + "/" + itoa(len(parts)) + ")\n" + parts[i]
		}
	}
	return parts
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
