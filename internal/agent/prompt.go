package agent

import (
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Prompter 拼装消息
type Prompter struct {
	protocol string
}

// NewPrompter 创建 Prompter
func NewPrompter() (*Prompter, error) {
	p, err := domain.LoadProtocol()
	if err != nil {
		return nil, err
	}
	return &Prompter{protocol: p}, nil
}

// PromptInput 动态上下文
type PromptInput struct {
	DomainName  string
	Node        *domain.NodeSpec
	NodeTitle   string
	Layer       string
	Progress    []storage.UserProgress
	Reinforce   *string
	Phase       string
	Turn        string
	Exercise       *storage.ExerciseContext
	History        []llm.Message
	RecentMistakes      []string
	UserProfile         string
	PendingPrereqTitles []string
}

// BuildMessages 构建 LLM 消息列表
func (p *Prompter) BuildMessages(in PromptInput, schemaJSON string) []llm.Message {
	system := p.protocol + "\n\n" + buildContext(in)
	if schemaJSON != "" {
		system += "\n\n【输出格式】仅输出 JSON，不要 markdown 代码块：\n" + schemaJSON
	}
	msgs := []llm.Message{{Role: "system", Content: system}}
	msgs = append(msgs, trimHistory(in.History)...)
	if in.Turn != "" {
		msgs = append(msgs, llm.Message{Role: "user", Content: in.Turn})
	}
	return msgs
}

func buildContext(in PromptInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "【领域】%s\n", in.DomainName)
	if in.Node != nil {
		fmt.Fprintf(&b, "【当前节点】%s（%s）\n", in.Node.Node, in.Layer)
		if len(in.Node.CoreConcepts) > 0 {
			b.WriteString("【本节点】核心：")
			b.WriteString(strings.Join(in.Node.CoreConcepts, "；"))
			b.WriteString("\n")
		}
		if len(in.Node.CommonMistakes) > 0 {
			b.WriteString("易混：")
			b.WriteString(strings.Join(in.Node.CommonMistakes, "；"))
			b.WriteString("\n")
		}
		if len(in.Node.Boundaries) > 0 {
			b.WriteString("后续节点再学：")
			b.WriteString(strings.Join(in.Node.Boundaries, "；"))
			b.WriteString("\n")
		}
		if len(in.Node.ExerciseIdeas) > 0 {
			b.WriteString("【出题参考】")
			b.WriteString(strings.Join(in.Node.ExerciseIdeas, "；"))
			b.WriteString("\n")
		}
	}
	if len(in.RecentMistakes) > 0 {
		fmt.Fprintf(&b, "【本次薄弱】%s\n", strings.Join(in.RecentMistakes, "；"))
	}
	if len(in.Progress) > 0 {
		var done []string
		for _, pr := range in.Progress {
			if pr.Status == "completed" {
				done = append(done, pr.NodeKey)
			}
		}
		if len(done) > 0 {
			fmt.Fprintf(&b, "【进度】已完成：%s\n", strings.Join(done, ", "))
		}
	}
	if in.Reinforce != nil && *in.Reinforce != "" {
		fmt.Fprintf(&b, "【可选巩固】%s（仅出题时使用，勿向用户提及）\n", *in.Reinforce)
	}
	if strings.TrimSpace(in.UserProfile) != "" {
		fmt.Fprintf(&b, "【学生画像】%s\n", strings.TrimSpace(in.UserProfile))
	}
	if len(in.PendingPrereqTitles) > 0 {
		fmt.Fprintf(&b, "【前置未完成】用户尚未点亮：%s。开场先用 1～2 句补必要背景，再进入本节点；勿指责或阻止学习。\n",
			strings.Join(in.PendingPrereqTitles, "、"))
	}
	fmt.Fprintf(&b, "【本轮】%s\n", in.Phase)
	if in.Exercise != nil && in.Exercise.Question != "" {
		fmt.Fprintf(&b, "【当前练习题】%s\n", in.Exercise.Question)
		if in.Exercise.AnswerFormat != "" {
			fmt.Fprintf(&b, "【作答方式】%s\n", in.Exercise.AnswerFormat)
		}
	}
	return b.String()
}

func trimHistory(h []llm.Message) []llm.Message {
	const max = 8
	if len(h) <= max {
		return h
	}
	return h[len(h)-max:]
}
