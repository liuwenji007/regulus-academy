package ingest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
)

// FromURL 抓取网页并提取正文
func FromURL(ctx context.Context, rawURL string) (Source, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return Source{}, fmt.Errorf("URL 不能为空")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Source{}, fmt.Errorf("URL 格式无效")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return Source{}, fmt.Errorf("仅支持 http/https URL")
	}

	timeout := time.Duration(fetchTimeoutSec()) * time.Second
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Source{}, err
	}
	req.Header.Set("User-Agent", "RegulusAcademy/1.0 (+https://regulus.academy)")

	resp, err := client.Do(req)
	if err != nil {
		return Source{}, fmt.Errorf("抓取网页失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Source{}, fmt.Errorf("网页返回状态 %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxURLChars()*4)))
	if err != nil {
		return Source{}, fmt.Errorf("读取网页失败: %w", err)
	}

	article, err := readability.FromReader(strings.NewReader(string(body)), parsed)
	if err != nil {
		return Source{}, fmt.Errorf("提取网页正文失败: %w", err)
	}
	text, err := validateText(article.TextContent, maxURLChars(), "网页")
	if err != nil {
		return Source{}, err
	}

	return Source{
		Kind: KindURL,
		Text: text,
		Meta: Meta{URL: rawURL, CharCount: len([]rune(text))},
	}, nil
}
