package ingest

import (
	"bytes"
	"context"
	"fmt"

	pdfextract "github.com/giraffesyo/pdf"
)

func extractPDF(ctx context.Context, data []byte) (Source, error) {
	maxBytes := maxPDFBytes()
	maxPages := maxPDFPages()

	if len(data) == 0 {
		return Source{}, fmt.Errorf("PDF 文件为空")
	}
	if len(data) > maxBytes {
		return Source{}, fmt.Errorf("PDF 超过大小上限（%d MB）", maxBytes/(1024*1024))
	}

	doc, err := pdfextract.Extract(ctx, bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Source{}, fmt.Errorf("无法解析 PDF（扫描版或加密文件可能无法提取文字）: %w", err)
	}
	pageCount := len(doc.Pages)
	if pageCount == 0 {
		return Source{}, fmt.Errorf("PDF未提取到可用正文（扫描版或文字被转成图片/曲线时会出现此情况）")
	}
	if pageCount > maxPages {
		return Source{}, fmt.Errorf("PDF 页数超过上限（%d 页）", maxPages)
	}

	text, err := validateText(doc.Text(), maxPDFChars(), "PDF")
	if err != nil {
		return Source{}, err
	}
	if err := checkExtractedText(text, pageCount); err != nil {
		return Source{}, err
	}

	return Source{
		Kind: KindPDF,
		Text: text,
		Meta: Meta{PageCount: pageCount, CharCount: len([]rune(text))},
	}, nil
}
