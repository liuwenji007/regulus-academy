package ingest

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/ledongthuc/pdf"
)

// FromPDF 从 PDF 二进制流提取纯文本
func FromPDF(r io.Reader) (Source, error) {
	maxBytes := maxPDFBytes()
	maxPages := maxPDFPages()

	data, err := io.ReadAll(io.LimitReader(r, int64(maxBytes)+1))
	if err != nil {
		return Source{}, fmt.Errorf("读取 PDF 失败: %w", err)
	}
	if len(data) == 0 {
		return Source{}, fmt.Errorf("PDF 文件为空")
	}
	if len(data) > maxBytes {
		return Source{}, fmt.Errorf("PDF 超过大小上限（%d MB）", maxBytes/(1024*1024))
	}

	tmp, err := os.CreateTemp("", "regulus-ingest-*.pdf")
	if err != nil {
		return Source{}, fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return Source{}, fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return Source{}, fmt.Errorf("关闭临时文件失败: %w", err)
	}

	f, reader, err := pdf.Open(tmpPath)
	if err != nil {
		return Source{}, fmt.Errorf("无法解析 PDF（扫描版可能无法提取文字）: %w", err)
	}
	defer f.Close()

	pageCount := reader.NumPage()
	if pageCount > maxPages {
		return Source{}, fmt.Errorf("PDF 页数超过上限（%d 页）", maxPages)
	}

	var buf bytes.Buffer
	for i := 1; i <= pageCount; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		if text != "" {
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString(text)
		}
	}

	text, err := validateText(buf.String(), maxPDFChars(), "PDF")
	if err != nil {
		return Source{}, err
	}

	return Source{
		Kind: KindPDF,
		Text: text,
		Meta: Meta{PageCount: pageCount, CharCount: len([]rune(text))},
	}, nil
}
