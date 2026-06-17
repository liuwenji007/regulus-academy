package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/config"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "build":
		os.Exit(runBuild(os.Args[2:]))
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Regulus Academy CLI

用法:
  regulus build <学习主题> [选项]

选项:
  --coach-root <dir>   regulus-coach 目录（默认 REGULUS_COACH_ROOT 或自动查找）
  --output-dir <dir>   输出 domains 父目录（默认 <coach-root>/domains）
  --force              已有内置 Domain 时仍用 LLM 重新生成并覆盖
  --profile <text>     可选学生画像，影响建树裁剪

示例:
  regulus build "想学 Rust"
  regulus build "Agent 原理" --output-dir ./my-coach/domains

环境变量:
  从当前目录 .env 读取 LLM_API_KEY、LLM_BASE_URL、LLM_MODEL 等（与 Web 相同）
`)
}

func runBuild(args []string) int {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	coachRoot := fs.String("coach-root", "", "regulus-coach 目录")
	outputDir := fs.String("output-dir", "", "输出 domains 父目录")
	force := fs.Bool("force", false, "覆盖已有内置 Domain")
	profile := fs.String("profile", "", "可选学生画像")
	_ = fs.Parse(args)

	topic := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if topic == "" {
		fmt.Fprintln(os.Stderr, "错误: 请提供学习主题，例如: regulus build \"想学 Rust\"")
		return 1
	}

	cfg := config.Load()
	client := llm.NewFromConfig(cfg.LLM)
	if !client.Configured() {
		fmt.Fprintln(os.Stderr, "错误: 未配置 LLM。请在 .env 中设置 LLM_API_KEY（或 DEEPSEEK_API_KEY）")
		return 1
	}

	if *coachRoot != "" {
		_ = os.Setenv("REGULUS_COACH_ROOT", *coachRoot)
	}
	root := domain.CoachRoot()
	outParent := *outputDir
	if outParent == "" {
		outParent = filepath.Join(root, "domains")
	}

	reg := domain.NewRegistry()
	ctx := context.Background()

	intent, err := reg.ParseIntent(ctx, client, topic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "意图分析失败: %v\n", err)
		return 1
	}

	if intent.Source == domain.SourceSkillPack && !*force {
		existing := filepath.Join(outParent, intent.Slug)
		if st, statErr := os.Stat(existing); statErr == nil && st.IsDir() {
			fmt.Printf("已有内置 Domain: %s\n", existing)
			fmt.Println("无需建课。若要重新生成，请加 --force")
			return 0
		}
		fmt.Printf("主题「%s」匹配内置 Skill 包（slug=%s）。\n", intent.DisplayName, intent.Slug)
		fmt.Printf("内置路径: %s\n", filepath.Join(root, "domains", intent.Slug))
		fmt.Println("若要 LLM 重新生成并写入 domains/，请加 --force")
		return 0
	}

	if intent.Source == domain.SourceSkillPack && *force {
		intent.Source = domain.SourceGenerated
	}

	builder := domain.NewTreeBuilder(reg)
	tree, nodes, err := builder.Build(ctx, client, intent, topic, strings.TrimSpace(*profile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "建课失败: %v\n", err)
		return 1
	}

	files, err := domain.ExportToFiles(tree, intent.Slug, "", "", 1, nodes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "导出文件失败: %v\n", err)
		return 1
	}

	dest, err := domain.WriteDomainFiles(outParent, intent.Slug, files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "写入失败: %v\n", err)
		return 1
	}

	nodeCount := countTreeNodes(tree)
	fmt.Printf("建课完成: %s（slug=%s，%d 个节点）\n", intent.DisplayName, intent.Slug, nodeCount)
	fmt.Printf("已写入: %s\n", dest)
	fmt.Println("在 Agent 中确认 regulus-coach/domains/ 下存在该目录后即可开始学习。")
	return 0
}

func countTreeNodes(tree *storage.KnowledgeTree) int {
	if tree == nil {
		return 0
	}
	n := 0
	for _, layer := range tree.Layers {
		n += len(layer.Nodes)
	}
	return n
}
