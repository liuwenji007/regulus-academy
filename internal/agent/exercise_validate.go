package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
)

// IsJSONSchemaDocument 判断模型是否把 JSON Schema 原文回显（而非题目实例）。
func IsJSONSchemaDocument(raw string) bool {
	extracted := llm.ExtractJSON(raw)
	if extracted == "" {
		return false
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal([]byte(extracted), &probe); err != nil {
		return false
	}
	_, hasType := probe["type"]
	_, hasProps := probe["properties"]
	_, hasRequired := probe["required"]
	if !(hasType && hasProps && hasRequired) {
		return false
	}
	// 真题实例顶层 question 为字符串；schema 里 question 在 properties 下且为对象
	if q, ok := probe["question"]; ok {
		var s string
		if err := json.Unmarshal(q, &s); err == nil && strings.TrimSpace(s) != "" {
			return false
		}
	}
	return true
}

// ValidateExerciseOutput 校验出题结果可用（至少要有题干）。
func ValidateExerciseOutput(out ExerciseOutput) error {
	if strings.TrimSpace(out.Question) == "" {
		return fmt.Errorf("出题结果缺少题干")
	}
	return nil
}

const exerciseSchemaEchoRetryHint = `你上次输出了 JSON Schema（含 type/properties/required），那是字段说明不是题目。
请重新输出**一道具体练习题实例**，例如：
{"question":"……题干……","question_type":"short_answer","answer_format":"text","reinforced_concepts":["……"]}
不要复述 schema，不要 markdown 代码块。`

func parseExerciseOutputRaw(raw string) (ExerciseOutput, bool) {
	extracted := llm.ExtractJSON(raw)
	var out ExerciseOutput
	if err := json.Unmarshal([]byte(extracted), &out); err != nil {
		return ExerciseOutput{}, false
	}
	if ValidateExerciseOutput(out) != nil {
		return ExerciseOutput{}, false
	}
	return out, true
}

func (c *Coach) generateExerciseOutput(ctx context.Context, msgs []llm.Message, temp float64) (ExerciseOutput, error) {
	client := c.llmClient(ctx)
	raw, err := client.ChatWithTemp(ctx, msgs, temp)
	if err != nil {
		return ExerciseOutput{}, err
	}
	if out, ok := parseExerciseOutputRaw(raw); ok {
		return out, nil
	}

	// 仅对「回显 JSON Schema」重试；其它坏输出（如误返回掌握度/批改 JSON）立即失败，避免多耗一次调用、打乱 mock/编排。
	if !IsJSONSchemaDocument(raw) {
		log.Printf("coach.exercise: invalid exercise JSON (prefix %q)", truncateRunes(raw, 48))
		return ExerciseOutput{}, fmt.Errorf("出题失败：未能得到有效题干，请重试")
	}

	log.Printf("coach.exercise: model echoed JSON schema, retrying with instance hint")
	retryMsgs := append(append([]llm.Message{}, msgs...),
		llm.Message{Role: "assistant", Content: raw},
		llm.Message{Role: "user", Content: exerciseSchemaEchoRetryHint},
	)
	raw2, err2 := client.ChatWithTemp(ctx, retryMsgs, temp)
	if err2 != nil {
		return ExerciseOutput{}, fmt.Errorf("重试出题失败: %w", err2)
	}
	if out, ok := parseExerciseOutputRaw(raw2); ok {
		return out, nil
	}
	if IsJSONSchemaDocument(raw2) {
		return ExerciseOutput{}, fmt.Errorf("模型回显了题目格式说明而非具体题目，请再点一次「开始练习」")
	}
	return ExerciseOutput{}, fmt.Errorf("出题失败：未能得到有效题干，请重试")
}
