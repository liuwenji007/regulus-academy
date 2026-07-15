package llm

import (
	"context"
	"os"
	"strconv"
	"strings"
)

const (
	// 教练批改等小 JSON 的默认输出上限（可被 REGULUS_LLM_MAX_TOKENS 覆盖）。
	defaultJSONMaxTokens = 8192
)

type jsonMaxTokensKey struct{}

// WithJSONMaxTokens 为单次 ChatJSON 设置输出上限；n=0 表示不设 max_tokens，交给模型默认/最大值。
func WithJSONMaxTokens(ctx context.Context, n int) context.Context {
	return context.WithValue(ctx, jsonMaxTokensKey{}, n)
}

func jsonMaxTokensFromContext(ctx context.Context) (int, bool) {
	if ctx == nil {
		return 0, false
	}
	v, ok := ctx.Value(jsonMaxTokensKey{}).(int)
	return v, ok
}

// MaxTokensFromEnv JSON 模式输出上限。
// REGULUS_LLM_MAX_TOKENS：未设置默认 8192；设为 0 表示不设上限（请求体不传 max_tokens）。
func MaxTokensFromEnv() int {
	return tokensFromEnv("REGULUS_LLM_MAX_TOKENS", defaultJSONMaxTokens)
}

// DomainBuildMaxTokensFromEnv 建树/regenerate 等大 JSON 输出上限。
// REGULUS_DOMAIN_BUILD_MAX_TOKENS：未设置时默认 0（不设上限）；设为 0 同上。
// 若未单独设置 DOMAIN_BUILD，但设置了 REGULUS_LLM_MAX_TOKENS，则沿用后者。
func DomainBuildMaxTokensFromEnv() int {
	if v := strings.TrimSpace(os.Getenv("REGULUS_DOMAIN_BUILD_MAX_TOKENS")); v != "" {
		return tokensFromEnv("REGULUS_DOMAIN_BUILD_MAX_TOKENS", 0)
	}
	if v := strings.TrimSpace(os.Getenv("REGULUS_LLM_MAX_TOKENS")); v != "" {
		return MaxTokensFromEnv()
	}
	return 0
}

func tokensFromEnv(key string, defaultVal int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}

func bumpMaxTokens(current int) int {
	if current <= 0 {
		return 0
	}
	bumped := current * 2
	if bumped <= current {
		bumped = current + 8192
	}
	return bumped
}

func isTruncatedJSONErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unexpected end of JSON input") ||
		strings.Contains(msg, "unexpected EOF")
}

func jsonRetryUserMessage(err error) string {
	if isTruncatedJSONErr(err) {
		return "你上次 JSON 输出被截断（未完整闭合）。请精简 nodes 里 exercise_ideas、teaching_beats、common_mistakes 等字段篇幅，确保整份 JSON 一次输出完整；只输出 JSON，不要 markdown 代码块。"
	}
	msg := err.Error()
	if strings.Contains(msg, "invalid character") {
		if strings.Contains(msg, "after object key:value") {
			return "你上次 JSON 字符串内双引号未转义，解析失败。字符串内的 \" 须写成 \\\"；代码示例优先用反引号或「」。请只输出合法 JSON。"
		}
		if strings.Contains(msg, "after array element") {
			return "你上次 JSON 数组元素之间缺英文逗号或夹了自然语言（如 and），解析失败。数组须写成 [\"a\",\"b\"]。请只输出合法 JSON。"
		}
		return "你上次输出不是合法 JSON（引号未转义或逗号缺失等）。请只输出纯 JSON 对象，检查字符串内 \\\" 与数组英文逗号，不要附加说明文字。"
	}
	return "你上次输出无法被程序解析为 JSON（" + msg + "）。请只输出合法 JSON，不要 markdown 代码块。"
}
