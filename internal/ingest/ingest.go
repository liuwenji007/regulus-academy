package ingest

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode"
)

// MaxPDFBytes 返回 PDF 大小上限（供 API 层预检）
func MaxPDFBytes() int {
	return maxPDFBytes()
}

// SourceKind 材料来源类型
const (
	KindPDF = "pdf"
	KindURL = "url"
)

// Source 摄取后的纯文本材料
type Source struct {
	Kind     string
	Filename string
	Text     string
	Meta     Meta
}

// Meta 来源元信息
type Meta struct {
	PageCount int
	URL       string
	CharCount int
}

// Label 用于展示的来源描述
func (s Source) Label() string {
	switch s.Kind {
	case KindPDF:
		if s.Filename != "" {
			return s.Filename
		}
		return "PDF 文件"
	case KindURL:
		if s.Meta.URL != "" {
			return s.Meta.URL
		}
		return "网页"
	default:
		return "外部材料"
	}
}

// TopicHint 从文件名得到主题先验（去掉路径与 .pdf 后缀）
func (s Source) TopicHint() string {
	return topicHintFromFilename(s.Filename)
}

func topicHintFromFilename(name string) string {
	name = strings.TrimSpace(path.Base(name))
	if name == "" || name == "." {
		return ""
	}
	if ext := path.Ext(name); strings.EqualFold(ext, ".pdf") {
		name = strings.TrimSuffix(name, ext)
	}
	runes := []rune(name)
	if len(runes) > 200 {
		name = string(runes[:200])
	}
	return strings.TrimSpace(name)
}

func normalizeText(text string) string {
	text = strings.ToValidUTF8(text, "")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
	lines := strings.Split(text, "\n")
	var out []string
	prevBlank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !prevBlank {
				out = append(out, "")
				prevBlank = true
			}
			continue
		}
		out = append(out, line)
		prevBlank = false
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// FromPDFBytes 从内存中的 PDF 数据提取文本
func FromPDFBytes(ctx context.Context, data []byte, filename string) (Source, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	src, err := extractPDF(ctx, data)
	if err != nil {
		return Source{}, err
	}
	src.Filename = filename
	return src, nil
}

// FromPDF 从 PDF 二进制流提取纯文本
func FromPDF(r io.Reader) (Source, error) {
	maxBytes := maxPDFBytes()
	data, err := io.ReadAll(io.LimitReader(r, int64(maxBytes)+1))
	if err != nil {
		return Source{}, fmt.Errorf("读取 PDF 失败: %w", err)
	}
	return extractPDF(context.Background(), data)
}

// ReadLimited 读取 reader 并限制最大字节数
func ReadLimited(r io.Reader, max int) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, int64(max)+1))
}

func validateText(text string, maxChars int, label string) (string, error) {
	text = normalizeText(text)
	if text == "" {
		return "", fmt.Errorf("%s未提取到可用正文（扫描版或文字被转成图片/曲线时会出现此情况）", label)
	}
	if len([]rune(text)) > maxChars {
		return "", fmt.Errorf("%s正文过长（上限 %d 字符）", label, maxChars)
	}
	return text, nil
}
