package ingest

import (
	"fmt"
	"unicode"
)

func checkExtractedText(text string, pageCount int) error {
	if looksLikeEncodedGarbage(text) {
		return fmt.Errorf("PDF 文字提取异常：正文像是字体编码错位后的乱码，无法用于建课。请换用可选中文字的文字版 PDF（例如从 Word 或浏览器「打印为 PDF」）")
	}
	chars := len([]rune(text))
	if pageCount >= 3 && chars < pageCount*30 {
		return fmt.Errorf("PDF 各页抽出的文字过少（%d 页仅 %d 字），可能是扫描件或字体无法解码，无法用于建课。请换用可选中文字的文字版 PDF", pageCount, chars)
	}
	return nil
}

// looksLikeEncodedGarbage 识别自定义字体把 glyph 码当成 Unicode 吐出的正文
//（例如 WORKFLOW→XPSLGMPX：拉丁字母很多但几乎没有元音）。
func looksLikeEncodedGarbage(text string) bool {
	var han, latin, vowel, upper, total int
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		switch {
		case unicode.Is(unicode.Han, r):
			han++
		case r >= 'A' && r <= 'Z':
			latin++
			upper++
			if isLatinVowel(r) {
				vowel++
			}
		case r >= 'a' && r <= 'z':
			latin++
			if isLatinVowel(r) {
				vowel++
			}
		}
	}
	if total < 80 || latin < 80 {
		return false
	}
	if han*100/total >= 8 {
		return false
	}
	if latin*100/total < 40 {
		return false
	}
	if vowel*100/latin >= 18 {
		return false
	}
	return upper*100/latin >= 50
}

func isLatinVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return true
	default:
		return false
	}
}
